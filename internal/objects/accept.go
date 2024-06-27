package objects

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func Accept(ref *corev1.ObjectReference) bool {
	accept := []string{
		".krateo.io",
	}

	apiGroup := ref.GroupVersionKind().Group

	ok := true
	for _, el := range accept {
		ok = ok || strings.HasSuffix(apiGroup, el)
	}

	return ok
}
