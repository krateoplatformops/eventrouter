package router

import (
	"fmt"
	"time"

	"github.com/krateoplatformops/eventrouter/internal/objects"
	"github.com/rs/zerolog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// EventHandler is the interface used to receive events
type EventHandler interface {
	Handle(e corev1.Event)
}

// EventRouter is responsible for maintaining a stream of kubernetes
// system Events and pushing them to another channel for storage
type EventRouter struct {
	log            zerolog.Logger
	handler        EventHandler
	informer       cache.SharedInformer
	throttlePeriod time.Duration
	fn             EventHandler
}

type EventRouterOpts struct {
	RESTClient     rest.Interface
	Handler        EventHandler
	ResyncInterval time.Duration
	ThrottlePeriod time.Duration
	Log            zerolog.Logger
	Namespace      string
}

// NewEventRouter will create a new event router using the input params
func NewEventRouter(opts EventRouterOpts) *EventRouter {
	lw := cache.NewListWatchFromClient(
		opts.RESTClient,
		"events",
		opts.Namespace, // v1.NamespaceAll,
		fields.Everything(),
	)

	si := cache.NewSharedInformer(lw, &corev1.Event{}, opts.ResyncInterval)

	return &EventRouter{
		informer:       si,
		handler:        opts.Handler,
		log:            opts.Log,
		throttlePeriod: opts.ThrottlePeriod,
	}
}

// Run starts the EventRouter/Controller.
func (er *EventRouter) Run(stopCh <-chan struct{}) {
	er.informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    er.OnAdd,
			UpdateFunc: er.OnUpdate,
			DeleteFunc: er.OnDelete,
		},
	)

	defer utilruntime.HandleCrash()

	er.informer.Run(stopCh)

	// here is where we kick the caches into gear
	if !cache.WaitForCacheSync(stopCh, er.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	<-stopCh
}

// OnAdd is called when an event is created, or during the initial list
func (er *EventRouter) OnAdd(obj interface{}) {
	event := obj.(*corev1.Event)
	er.onEvent(event)
}

// OnUpdate is called any time there is an update to an existing event
func (er *EventRouter) OnUpdate(objOld interface{}, objNew interface{}) {
	event := objNew.(*corev1.Event)
	er.onEvent(event)
}

// OnDelete should only occur when the system garbage collects events via TTL expiration
func (er *EventRouter) OnDelete(obj interface{}) {
	if !er.log.Debug().Enabled() {
		return
	}

	e := obj.(*corev1.Event)
	// NOTE: This should *only* happen on TTL expiration there
	// is no reason to push this to a collector
	er.log.Debug().Msgf("Event deleted from the system: %v", e)
}

func (er *EventRouter) onEvent(event *corev1.Event) {
	if wasPatchedByKrateo(event) {
		return
	}

	// It's probably an old event we are catching, it's not the best way but anyways
	if er.throttlePeriod > 0 && time.Since(event.LastTimestamp.Time) > er.throttlePeriod {
		return
	}

	er.log.Debug().
		Str("msg", event.Message).
		Str("namespace", event.Namespace).
		Str("reason", event.Reason).
		Str("involvedObject", event.InvolvedObject.Name).
		Msg("Received event")

	if !objects.Accept(&event.InvolvedObject) {
		return
	}

	er.handler.Handle(*event.DeepCopy())

	return
}
