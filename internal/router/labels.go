package router

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/krateoplatformops/eventrouter/internal/objects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

const (
	keyCompositionID = "krateo.io/composition-id"
)

func hasCompositionId(obj *corev1.Event) bool {
	labels := obj.GetLabels()
	if len(labels) == 0 {
		return false
	}

	val, ok := labels[keyCompositionID]
	if len(val) == 0 {
		return false
	}
	return ok
}

func findCompositionID(resolver *objects.ObjectResolver, ref *corev1.ObjectReference) (cid string, err error) {
	var obj *unstructured.Unstructured

	retryErr := retry.OnError(retry.DefaultRetry,
		func(e error) bool {
			if e != nil {
				resolver.InvalidateRESTMapperCache()
				return true
			}
			return false
		},
		func() error {
			obj, err = resolver.ResolveReference(context.Background(), ref)
			return err
		})
	if retryErr != nil {
		return "", retryErr
	}

	if obj == nil {
		klog.V(4).InfoS("object not found resolving reference",
			"name", ref.Name,
			"kind", ref.Kind,
			"apiVersion", ref.APIVersion)
		return "", nil
	}

	labels := obj.GetLabels()
	if len(labels) == 0 {
		klog.V(4).InfoS("no labels found in resolved reference",
			"name", ref.Name,
			"kind", ref.Kind,
			"apiVersion", ref.APIVersion)
		return "", nil
	}

	klog.V(4).InfoS("labels found in resolved reference",
		"labels", spew.Sdump(labels))

	return labels[keyCompositionID], nil
}
