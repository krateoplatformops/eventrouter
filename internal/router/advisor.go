package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/krateoplatformops/eventrouter/apis/v1alpha1"
	"k8s.io/klog/v2"
)

type advOpts struct {
	httpClient       *http.Client
	registrationSpec v1alpha1.RegistrationSpec
	eventInfo        EventInfo
}

func newAdvisor(opts advOpts) *advisor {
	return &advisor{
		httpClient: opts.httpClient,
		reg:        opts.registrationSpec,
		evt:        opts.eventInfo,
	}
}

type advisor struct {
	httpClient *http.Client
	reg        v1alpha1.RegistrationSpec
	evt        EventInfo
}

func (c *advisor) Job() {
	err := c.notify()
	if err != nil {
		klog.Errorf("unable to notify %s: %s", c.reg.ServiceName, err.Error())
	}
}

func (c *advisor) notify() error {
	dat, err := json.Marshal(c.evt)
	if err != nil {
		return fmt.Errorf("cannot encode notification (deploymentId:%s, destinationURL:%s): %w",
			c.evt.DeploymentId, c.reg.Endpoint, err)
	}

	ctx, cncl := context.WithTimeout(context.Background(), time.Second*40)
	defer cncl()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.reg.Endpoint, bytes.NewBuffer(dat))
	if err != nil {
		return fmt.Errorf("cannot create notification (deploymentId:%s, destinationURL:%s): %w",
			c.evt.DeploymentId, c.reg.Endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	_, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send notification (deploymentId:%s, destinationURL:%s): %w",
			c.evt.DeploymentId, c.reg.Endpoint, err)
	}

	return nil
}
