package router

import (
	"context"

	"github.com/krateoplatformops/eventrouter/internal/objects"
	corev1 "k8s.io/api/core/v1"
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

func findCompositionID(resolver *objects.ObjectResolver, ref *corev1.ObjectReference) (string, error) {
	obj, err := resolver.ResolveReference(context.Background(), ref)
	if err != nil {
		return "", err
	}
	if obj == nil {
		return "", nil
	}

	labels := obj.GetLabels()
	if len(labels) == 0 {
		return "", nil
	}

	return labels[keyCompositionID], nil
}
