package http

import (
	"crypto/tls"
	"net/http"
	"time"
)

type ClientOpts struct {
	Verbose  bool
	Insecure bool
	Timeout  time.Duration
}

func ClientFromOpts(opts ClientOpts) *http.Client {
	transport := defaultTransport()

	if opts.Insecure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		transport = &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsConfig,
		}
	}

	if opts.Verbose {
		transport = &verboseTracer{transport}
	}

	timeout := 20 * time.Second
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// defaultTransport is the default implementation of Transport and is
// used by DefaultClient. It establishes network connections as needed
// and caches them for reuse by subsequent calls. It uses HTTP proxies
// as directed by the $HTTP_PROXY and $NO_PROXY (or $http_proxy and
// $no_proxy) environment variables.
func defaultTransport() http.RoundTripper {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 50
	t.MaxConnsPerHost = 50
	t.MaxIdleConnsPerHost = 50
	t.TLSHandshakeTimeout = 20 * time.Second
	t.ExpectContinueTimeout = 2 * time.Second
	return t
}
