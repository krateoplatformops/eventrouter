package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/krateoplatformops/eventrouter/apis/v1alpha1"
	"github.com/krateoplatformops/eventrouter/internal/objects"
	"github.com/krateoplatformops/eventrouter/internal/support"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		c.log.Error().Err(err).
			Str("involvedObject", ref.Name).
			Msg("Unable to patch with labels.")
		return
	}

	all, err := c.getAllRegistrations(context.Background())
	if err != nil {
		c.log.Error().Err(err).
			Str("involvedObject", ref.Name).
			Msg("Unable to list registrations.")
		return
	}

	msg := NewEventInfo(deploymentId, &evt)

	c.notifyAll(all, msg)
}

func (c *pusher) notifyAll(all map[string]v1alpha1.RegistrationSpec, evt EventInfo) {
	for _, el := range all {
		go func(it v1alpha1.RegistrationSpec) {
			err := c.notify(it, evt)
			if err != nil {
				c.log.Error().Err(err).Msgf("Unable to notify %s", it.ServiceName)
			}
		}(el)
	}
}

func (c *pusher) notify(reg v1alpha1.RegistrationSpec, evt EventInfo) error {
	dat, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("cannot encode notification (deploymentId:%s, destinationURL:%s): %w",
			evt.DeploymentId, reg.Endpoint, err)
	}

	ctx, cncl := context.WithTimeout(context.Background(), time.Second*40)
	defer cncl()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reg.Endpoint, bytes.NewBuffer(dat))
	if err != nil {
		return fmt.Errorf("cannot create notification (deploymentId:%s, destinationURL:%s): %w",
			evt.DeploymentId, reg.Endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send notification (deploymentId:%s, destinationURL:%s): %w",
			evt.DeploymentId, reg.Endpoint, err)
	}

	return nil
}

func (c *pusher) getAllRegistrations(ctx context.Context) (map[string]v1alpha1.RegistrationSpec, error) {
	all, err := c.objectResolver.List(ctx, schema.GroupVersionKind{
		Group:   "eventrouter.krateo.io",
		Version: "v1alpha1",
		Kind:    "Registration",
	}, "")

	res := map[string]v1alpha1.RegistrationSpec{}
	if err != nil {
		return res, err
	}

	if all == nil {
		return res, nil
	}

	for _, el := range all.Items {
		serviceName, _, err := unstructured.NestedString(el.Object, "spec", "serviceName")
		if err != nil {
			c.log.Error().Err(err).
				Str("registration", el.GetName()).
				Msg("Reading 'serviceName' attribute.")
			continue
		}

		endpoint, _, err := unstructured.NestedString(el.Object, "spec", "endpoint")
		if err != nil {
			c.log.Error().Err(err).
				Str("registration", el.GetName()).
				Msg("Reading 'endpoint' attribute.")
			continue
		}

		res[el.GetName()] = v1alpha1.RegistrationSpec{
			ServiceName: serviceName,
			Endpoint:    endpoint,
		}
	}

	return res, nil
}
