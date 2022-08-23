package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/krateoplatformops/eventrouter/internal/router"
	"github.com/krateoplatformops/eventrouter/internal/support"
	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
		support.EnvBool("EVENT_ROUTER_DEBUG", false), "dump verbose output")
	resyncInterval := flag.Duration("resync-interval",
		support.EnvDuration("EVENT_ROUTER_RESYNC_INTERVAL", time.Minute*3), "resync interval")
	throttlePeriod := flag.Duration("throttle-period",
		support.EnvDuration("EVENT_ROUTER_THROTTLE_PERIOD", 0), "throttle period")
	namespace := flag.String("namespace",
		support.EnvString("EVENT_ROUTER_NAMESPACE", ""), "namespace to list and watch")
	eventServiceUrl := flag.String("event-service-url",
		support.EnvString("EVENT_SERVICE_URL", "https://api.krateo.site/event/"), "event service url")

	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Initialize the logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Default level for this log is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log := zerolog.New(os.Stdout).With().
		Str("service", serviceName).
		Timestamp().
		Logger()

	// Kubernetes configuration
	var cfg *rest.Config
	var err error
	if len(*kubeconfig) > 0 {
		cfg, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatal().Err(err).Msg("building kube config")
	}

	// creates the clientset from kubeconfig
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("creating the kubernetes clientset")
	}

	handler, err := router.NewNotifier(router.NotifierOpts{
		RESTConfig:      cfg,
		EventServiceUrl: *eventServiceUrl,
		Log:             log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("creating the event notifier")
	}

	eventRouter := router.NewEventRouter(router.EventRouterOpts{
		RESTClient:     clientSet.CoreV1().RESTClient(),
		Handler:        handler,
		Namespace:      *namespace,
		Log:            log,
		ThrottlePeriod: *throttlePeriod,
	})

	stop := sigHandler(log)

	// Startup the EventRouter
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Info().
			Str("version", Version).
			Bool("debug", *debug).
			Dur("resyncInterval", *resyncInterval).
			Dur("throttlePeriod", *throttlePeriod).
			Str("eventServiceUrl", *eventServiceUrl).
			Str("namespace", *namespace).
			Msgf("Starting %s", serviceName)

		eventRouter.Run(stop)
	}()

	wg.Wait()
	log.Warn().Msgf("%s done", serviceName)
	os.Exit(1)
}

// setup a signal hander to gracefully exit
func sigHandler(log zerolog.Logger) <-chan struct{} {
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
		log.Warn().Msgf("Signal (%v) detected, shutting down", sig)
		close(stop)
	}()
	return stop
}
