package router

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/krateoplatformops/eventrouter/internal/objects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	keyCompositionID = "krateo.io/composition-id"
	keyPatchedBy     = "krateo.io/patched-by"
)

func patchWithLabels(resolver *objects.ObjectResolver, evt *corev1.Event, compositionId string) error {
	extras, err := createPatchData(map[string]string{
		keyCompositionID: compositionId,
		keyPatchedBy:     "krateo",
	})
	if err != nil {
		return fmt.Errorf("creating patch data: %w", err)
	}

	err = resolver.Patch(context.Background(), objects.PatchOpts{
		GVK:       schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Event"},
		Name:      evt.Name,
		Namespace: evt.Namespace,
		PatchData: extras,
	})
	if err != nil {
		if len(compositionId) > 0 {
			return fmt.Errorf("patching event with composition id '%s': %w", compositionId, err)
		} else {
			return fmt.Errorf("patching event: %w", err)
		}
	}

	return nil
}

func createPatchData(labels map[string]string) ([]byte, error) {
	patch := struct {
		Metadata struct {
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
	}{}
	patch.Metadata.Labels = map[string]string{}

	for k, v := range labels {
		if len(v) > 0 {
			patch.Metadata.Labels[k] = v
		}
	}

	return json.Marshal(patch)
}

func wasPatchedByKrateo(obj *corev1.Event) bool {
	labels := obj.GetLabels()
	if len(labels) == 0 {
		return false
	}

	_, ok := labels[keyPatchedBy]
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
