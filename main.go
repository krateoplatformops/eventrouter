package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/krateoplatformops/eventrouter/internal/env"
	httputil "github.com/krateoplatformops/eventrouter/internal/helpers/http"
	"github.com/krateoplatformops/eventrouter/internal/helpers/queue"
	"github.com/krateoplatformops/eventrouter/internal/router"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const (
	serviceName = "EventRouter"
)

var (
	Version string
	Build   string
)

func main() {
	// Flags
	kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag, "", "absolute path to the kubeconfig file")
	debug := flag.Bool("debug",
		env.Bool("EVENT_ROUTER_DEBUG", false), "dump verbose output")
	insecure := flag.Bool("insecure", env.Bool("EVENT_ROUTER_INSECURE", false),
		"allow insecure server connections when using SSL")
	resyncInterval := flag.Duration("resync-interval",
		env.Duration("EVENT_ROUTER_RESYNC_INTERVAL", time.Minute*3), "resync interval")
	throttlePeriod := flag.Duration("throttle-period",
		env.Duration("EVENT_ROUTER_THROTTLE_PERIOD", 0), "throttle period")
	namespace := flag.String("namespace",
		env.String("EVENT_ROUTER_NAMESPACE", ""), "namespace to list and watch")
	queueMaxCapacity := flag.Int("queue-max-capacity",
		env.Int("EVENT_ROUTER_QUEUE_MAX_CAPACITY", 10), "notification queue buffer size")
	queueWorkerThreads := flag.Int("queue-worker-threads",
		env.Int("EVENT_ROUTER_QUEUE_WORKER_THREADS", 50), "number of worker threads in the notification queue")

	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
	}

	klog.InitFlags(nil)

	flag.Parse()

	// Kubernetes configuration
	var cfg *rest.Config
	var err error
	if len(*kubeconfig) > 0 {
		cfg, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("unable to init kubeconfig: %s", err.Error())
	}

	if klog.V(4).Enabled() {
		cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return &httputil.Tracer{RoundTripper: rt}
		}
	}

	// creates the clientset from kubeconfig
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("unable to create kubernetes clientset: %s", err.Error())
	}

	// setup notification worker queue
	q := queue.NewQueue(*queueMaxCapacity, *queueWorkerThreads)
	q.Run()
	defer q.Terminate()

	handler, err := router.NewPusher(router.PusherOpts{
		RESTConfig: cfg,
		Queue:      q,
		Verbose:    *debug,
		Insecure:   *insecure,
	})
	if err != nil {
		klog.Fatalf("unable to create the event notifier: %s", err.Error())
	}

	eventRouter := router.NewEventRouter(router.EventRouterOpts{
		RESTClient:     clientSet.CoreV1().RESTClient(),
		Handler:        handler,
		Namespace:      *namespace,
		ThrottlePeriod: *throttlePeriod,
	})

	stop := sigHandler()

	// Startup the EventRouter
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		klog.InfoS(fmt.Sprintf("Starting %s", serviceName),
			"debug", *debug,
			"resyncInterval", *resyncInterval,
			"throttlePeriod", *throttlePeriod,
			"namespace", *namespace,
			"queueMaxCapacity", *queueMaxCapacity,
			"queueWorkerThreads", *queueWorkerThreads)

		eventRouter.Run(stop)
	}()

	wg.Wait()
	klog.Infof("%s done", serviceName)
	os.Exit(1)
}

// setup a signal hander to gracefully exit
func sigHandler() <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGTERM, // Termination Request
			syscall.SIGSEGV, // FullDerp
			syscall.SIGABRT, // Abnormal termination
			syscall.SIGILL,  // illegal instruction
			syscall.SIGFPE)  // floating point - this is why we can't have nice things
		sig := <-c
		klog.Infof("Signal (%v) detected, shutting down", sig)
		close(stop)
	}()
	return stop
}
