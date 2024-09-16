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
	"k8s.io/klog/v2"
)

type ObjectResolver struct {
	dynamicClient   dynamic.Interface
	discoveryClient *discovery.DiscoveryClient
	mapper          *restmapper.DeferredDiscoveryRESTMapper
	mapperCache     discovery.CachedDiscoveryInterface
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

	mapperCache := memory.NewMemCacheClient(discoveryClient)

	return &ObjectResolver{
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		mapper:          restmapper.NewDeferredDiscoveryRESTMapper(mapperCache),
		mapperCache:     mapperCache,
	}, nil
}

func (r *ObjectResolver) InvalidateRESTMapperCache() {
	r.mapperCache.Invalidate()
}

func (r *ObjectResolver) List(ctx context.Context, gvk schema.GroupVersionKind, ns string) (*unstructured.UnstructuredList, error) {
	dri, err := r.getResourceInterfaceForGVR(gvk, ns)
	if err != nil {
		if isNoKindMatchError(err) {
			return nil, nil
		}
		return nil, err
	}

	all, err := dri.List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return all, nil
}

func (r *ObjectResolver) ResolveReference(ctx context.Context, ref *corev1.ObjectReference) (*unstructured.Unstructured, error) {
	dri, err := r.getResourceInterfaceForGVR(ref.GroupVersionKind(), ref.Namespace)
	if err != nil {
		if isNoKindMatchError(err) {
			klog.V(4).ErrorS(err, "can't find any match for this kind",
				"gvk", ref.GroupVersionKind(), "name", ref.Name, "namespace", ref.Namespace)
		}
		return nil, err
	}

	res, err := dri.Get(ctx, ref.Name, metav1.GetOptions{})
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
			klog.V(4).ErrorS(err, "can't find any match for this kind",
				"gvk", opts.GVK, "name", opts.Name, "namespace", opts.Namespace)
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

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		dr = r.dynamicClient.Resource(mapping.Resource)
	} else {
		dr = r.dynamicClient.Resource(mapping.Resource).
			Namespace(namespace)
	}

	return dr, nil
}

func isNoKindMatchError(err error) bool {
	var noKindMatchError *meta.NoKindMatchError
	return errors.As(err, &noKindMatchError)
}
