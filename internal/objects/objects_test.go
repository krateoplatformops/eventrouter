//go:build integration
// +build integration

package objects

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
)

func TestListObjects(t *testing.T) {
	kubeconfig, err := ioutil.ReadFile(clientcmd.RecommendedHomeFile)
	assert.Nil(t, err, "expecting nil error loading kubeconfig")

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	assert.Nil(t, err, "expecting nil error creating clientConfig")

	restConfig, err := clientConfig.ClientConfig()
	assert.Nil(t, err, "expecting nil error getting restConfig")

	resolver, err := NewObjectResolver(restConfig)
	assert.Nil(t, err, "expecting nil error creating object resolver")

	all, err := resolver.List(context.TODO(), schema.GroupVersionKind{
		Group:   "eventrouter.krateo.io",
		Version: "v1alpha1",
		Kind:    "Registration",
	}, "")
	assert.Nil(t, err, "expecting nil error listing registrations")

	if all == nil {
		return
	}

	for _, el := range all.Items {
		fmt.Println(unstructured.NestedString(el.Object, "spec", "serviceName"))
		fmt.Println(unstructured.NestedString(el.Object, "spec", "endpoint"))
		fmt.Println("------")
	}
}
