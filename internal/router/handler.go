package router

import (
	"context"
	"net/http"

	"github.com/krateoplatformops/eventrouter/apis/v1alpha1"
	httpHelper "github.com/krateoplatformops/eventrouter/internal/helpers/http"
	"github.com/krateoplatformops/eventrouter/internal/helpers/queue"
	"github.com/krateoplatformops/eventrouter/internal/objects"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type PusherOpts struct {
	RESTConfig *rest.Config
	Queue      queue.Queuer
	Log        zerolog.Logger
	Verbose    bool
	Insecure   bool
}

func NewPusher(opts PusherOpts) (EventHandler, error) {
	objectResolver, err := objects.NewObjectResolver(opts.RESTConfig)
	if err != nil {
		return nil, err
	}

	return &pusher{
		objectResolver: objectResolver,
		notifyQueue:    opts.Queue,
		log:            opts.Log,
		verbose:        opts.Verbose,
		httpClient: httpHelper.ClientFromOpts(httpHelper.ClientOpts{
			Verbose:  opts.Verbose,
			Insecure: opts.Insecure,
		}),
	}, nil
}

var _ EventHandler = (*pusher)(nil)

type pusher struct {
	objectResolver *objects.ObjectResolver
	notifyQueue    queue.Queuer
	httpClient     *http.Client
	log            zerolog.Logger
	verbose        bool
}

func (c *pusher) Handle(evt corev1.Event) {
	ref := &evt.InvolvedObject

	deploymentId, err := findDeploymentID(c.objectResolver, ref)
	if err != nil {
		c.log.Warn().Str("involvedObject", ref.Name).Msgf("looking for deploymentId: %s", err.Error())
		return
	}

	if c.verbose {
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
		job := newAdvisor(advOpts{
			httpClient:       c.httpClient,
			log:              c.log,
			registrationSpec: el,
			eventInfo:        evt,
		})

		c.notifyQueue.Push(job)
	}
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
