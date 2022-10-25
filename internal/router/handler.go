package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/krateoplatformops/eventrouter/internal/objects"
	"github.com/krateoplatformops/eventrouter/internal/support"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type PusherOpts struct {
	RESTConfig *rest.Config
	Log        zerolog.Logger
}

func NewPusher(opts PusherOpts) (EventHandler, error) {
	transport := http.DefaultTransport
	if opts.Log.Debug().Enabled() {
		transport = &support.HttpTracer{RoundTripper: http.DefaultTransport}
	}

	objectResolver, err := objects.NewObjectResolver(opts.RESTConfig)
	if err != nil {
		return nil, err
	}

	return &pusher{
		objectResolver: objectResolver,
		log:            opts.Log,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   40 * time.Second,
		},
	}, nil
}

var _ EventHandler = (*pusher)(nil)

type pusher struct {
	objectResolver *objects.ObjectResolver
	httpClient     *http.Client
	log            zerolog.Logger
}

func (c *pusher) Handle(evt corev1.Event) {
	ref := &evt.InvolvedObject

	deploymentId, err := findDeploymentID(c.objectResolver, ref)
	if err != nil {
		c.log.Warn().Str("involvedObject", ref.Name).Msgf("looking for deploymentId: %s", err.Error())
		return
	}

	if c.log.Debug().Enabled() {
		c.log.Debug().
			Str("name", evt.Name).
			Str("kind", ref.Kind).
			Str("apiGroup", evt.InvolvedObject.GroupVersionKind().Group).
			Str("reason", evt.Reason).
			Str("deploymentId", deploymentId).
			Msg(evt.Message)
	}

	err = patchWithLabels(c.objectResolver, &evt, deploymentId)
	if err != nil {
		c.log.Warn().Str("involvedObject", ref.Name).Msg(err.Error())
		return
	}

	msg := NewEventInfo(deploymentId, &evt)
	if err := c.notify(msg); err != nil {
		c.log.Warn().Str("involvedObject", ref.Name).Msgf("sending notification: %s", err.Error())
	}
}

func (c *pusher) notify(url string, evt EventInfo) error {
	dat, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("sending notification (deploymentId:%s): %w", evt.DeploymentId, err)
	}

	ctx, cncl := context.WithTimeout(context.Background(), time.Second*40)
	defer cncl()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(dat))
	if err != nil {
		return fmt.Errorf("sending notification (url:%s, deploymentId:%s): %w", url, evt.DeploymentId, err)
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending notification (url:%s,deploymentId:%s): %w", url, evt.DeploymentId, err)
	}

	return nil
}
