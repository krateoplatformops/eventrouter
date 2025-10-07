//go:build integration
// +build integration

package objects

import (
	"context"
	"fmt"
	"os"
	"testing"

	xenv "github.com/krateoplatformops/plumbing/env"

	"github.com/krateoplatformops/plumbing/e2e"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	testenv     env.Environment
	clusterName string
)

const (
	namespace = "eventrouter-test"
)

func TestMain(m *testing.M) {
	xenv.SetTestMode(true)

	clusterName = "kind"
	testenv = env.New()

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		e2e.CreateNamespace(namespace),
	).Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}

func TestListObjects(t *testing.T) {

	f := features.New("Setup").
		Setup(e2e.Logger("test")).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			resolver, err := NewObjectResolver(cfg.Client().RESTConfig())
			assert.Nil(t, err, "expecting nil error creating object resolver")

			all, err := resolver.List(context.TODO(), schema.GroupVersionKind{
				Group:   "eventrouter.krateo.io",
				Version: "v1alpha1",
				Kind:    "Registration",
			}, "")
			assert.Nil(t, err, "expecting nil error listing registrations")

			if all == nil {
				return ctx
			}

			for _, el := range all.Items {
				fmt.Println(unstructured.NestedString(el.Object, "spec", "serviceName"))
				fmt.Println(unstructured.NestedString(el.Object, "spec", "endpoint"))
				fmt.Println("------")
			}
			return ctx
		}).
		Feature()

	testenv.Test(t, f)
}
