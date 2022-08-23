package objects

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type ObjectResolver struct {
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	mapper          restmapper.DeferredDiscoveryRESTMapper
}

func NewObjectResolver(restConfig *rest.Config) (*ObjectResolver, error) {
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	restMapperCache := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(restMapperCache)

	return &ObjectResolver{
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		mapper:          *mapper,
	}, nil
}

func (r *ObjectResolver) ResolveReference(ctx context.Context, ref *corev1.ObjectReference) (*unstructured.Unstructured, error) {
	dri, err := r.getResourceInterfaceForGVR(ref.GroupVersionKind(), ref.Namespace)
	if err != nil {
		if isNoKindMatchError(err) {
			return nil, nil
		}
		return nil, err
	}

	res := &unstructured.Unstructured{}
	res, err = dri.Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return res, nil
}

type PatchOpts struct {
	PatchData []byte
	GVK       schema.GroupVersionKind
	Name      string
	Namespace string
}

func (r *ObjectResolver) Patch(ctx context.Context, opts PatchOpts) error {
	dri, err := r.getResourceInterfaceForGVR(opts.GVK, opts.Namespace)
	if err != nil {
		if isNoKindMatchError(err) {
			return nil
		}
		return err
	}

	_, err = dri.Patch(ctx, opts.Name, types.MergePatchType, opts.PatchData, metav1.PatchOptions{
		FieldManager: "krateo",
	})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (r *ObjectResolver) getResourceInterfaceForGVR(gvk schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	mapping, err := r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	// obtain REST interface for the GVR
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = r.dynamicClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = r.dynamicClient.Resource(mapping.Resource)
	}

	return dr, nil
}

func isNoKindMatchError(err error) bool {
	var noKindMatchError *meta.NoKindMatchError
	return errors.As(err, &noKindMatchError)
}
