package router

import (
	"context"
	"net/http"

	"github.com/krateoplatformops/eventrouter/apis/v1alpha1"
	httpHelper "github.com/krateoplatformops/eventrouter/internal/helpers/http"
	"github.com/krateoplatformops/eventrouter/internal/helpers/queue"
	"github.com/krateoplatformops/eventrouter/internal/objects"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type PusherOpts struct {
	RESTConfig *rest.Config
	Queue      queue.Queuer
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
	verbose        bool
}

func (c *pusher) Handle(evt corev1.Event) {
	ref := &evt.InvolvedObject

	deploymentId, err := findDeploymentID(c.objectResolver, ref)
	if err != nil {
		klog.ErrorS(err, "looking for deploymentId",
			"involvedObject", ref.Name)
		return
	}

	if c.verbose {
		klog.V(4).InfoS(evt.Message,
			"name", evt.Name,
			"kind", ref.Kind,
			"apiGroup", evt.InvolvedObject.GroupVersionKind().Group,
			"reason", evt.Reason,
			"deploymentId", deploymentId)
	}

	err = patchWithLabels(c.objectResolver, &evt, deploymentId)
	if err != nil {
		klog.ErrorS(err, "unable to patch with labels",
			"involvedObject", ref.Name)
		return
	}

	all, err := c.getAllRegistrations(context.Background())
	if err != nil {
		klog.ErrorS(err, "unable to list registrations",
			"involvedObject", ref.Name)
		return
	}

	msg := NewEventInfo(deploymentId, &evt)

	c.notifyAll(all, msg)
}

func (c *pusher) notifyAll(all map[string]v1alpha1.RegistrationSpec, evt EventInfo) {
	for _, el := range all {
		job := newAdvisor(advOpts{
			httpClient:       c.httpClient,
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
			klog.ErrorS(err, "unable to read 'serviceName' attribute",
				"registration", el.GetName())
			continue
		}

		endpoint, _, err := unstructured.NestedString(el.Object, "spec", "endpoint")
		if err != nil {
			klog.ErrorS(err, "unable to read 'endpoint' attribute",
				"registration", el.GetName())
			continue
		}

		res[el.GetName()] = v1alpha1.RegistrationSpec{
			ServiceName: serviceName,
			Endpoint:    endpoint,
		}
	}

	return res, nil
}
