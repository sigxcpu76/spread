package entity

import (
	"errors"
	"fmt"

	"rsprd.com/spread/pkg/deploy"

	kube "k8s.io/kubernetes/pkg/api"
)

// An Entity is a component (potentially containing sub-entities) that can be deployed to Kubernetes.
type Entity interface {
	deploy.Deployable
	Type() Type
	Objects() []deploy.KubeObject
	Source() string
	Attach(Entity) error
	DefaultMeta() kube.ObjectMeta
}

// base provides fields that are shared between all Entitys.
type base struct {
	entityType Type
	objects    deploy.Deployment
	source     string
	defaults   kube.ObjectMeta
}

func newBase(t Type, defaults kube.ObjectMeta, source string, objects []deploy.KubeObject) (base base, err error) {
	base.defaults = defaults

	deployment := deploy.Deployment{}
	for _, obj := range objects {
		if obj == nil {
			err = ErrorNilObject
			return
		}
		base.setDefaults(obj)
		err = deployment.Add(obj)
		if err != nil {
			err = fmt.Errorf("error adding '%s': %v", source, err)
			return
		}
	}

	base.source = source
	base.entityType = t
	base.objects = deployment
	return
}

// Objects returns slice of objects attached to Entity
func (base base) Objects() []deploy.KubeObject {
	return base.objects.Objects()
}

// Source returns an import source specific identifier
func (base base) Source() string {
	return base.source
}

// DefaultMeta returns the ObjectMeta that the Entity was created with
func (base base) DefaultMeta() kube.ObjectMeta {
	return base.defaults
}

// Type returns itself for trivial implementation of Entity
func (base base) Type() Type {
	return base.entityType
}

// validAttach checks object types to see if the attach is allowed. Objects can
// only be attached to objects higher in the hierarchy. However, to the nature of iota Application is 0, RC is 1, ...
func (base base) validAttach(e Entity) bool {
	return e.Type() <= EntityImage && base.Type() < e.Type()
}

// setDefaults sets the bases defaults on an object
func (base base) setDefaults(obj deploy.KubeObject) {
	setMetaDefaults(obj, base.defaults)
}

// Type identifies the entity's type.
type Type int

const (
	EntityApplication           Type = iota // Application (top of tree)
	EntityReplicationController             // Wrapper for kube.ReplicationController
	EntityPod                               // Wrapper for kube.Pod
	EntityContainer                         // Wrapper for kube.Container
	EntityImage                             // Represented by kube.Container's image field
)

// metaDefaults applies a set of defaults on a KubeObject. Non-empty fields on object override defaults.
func setMetaDefaults(obj deploy.KubeObject, defaults kube.ObjectMeta) {
	meta := obj.GetObjectMeta()

	// if namespace is not set, use default
	namespace := kube.NamespaceDefault
	if len(defaults.Namespace) > 0 {
		namespace = defaults.Namespace
	}

	if len(meta.GetNamespace()) == 0 {
		meta.SetNamespace(namespace)
	}

	// if name and generateName are not set use default generateName
	if len(defaults.GenerateName) > 0 && len(meta.GetName()) == 0 && len(meta.GetGenerateName()) == 0 {
		meta.SetGenerateName(defaults.GenerateName)
	}

	// set default labels
	labels := map[string]string{}
	if defaults.Labels != nil {
		labels = defaults.Labels
	}
	for k, v := range meta.GetLabels() {
		labels[k] = v
	}
	meta.SetLabels(labels)

	// set default annotations
	annotations := map[string]string{}
	if defaults.Annotations != nil {
		annotations = defaults.Annotations
	}
	for k, v := range meta.GetAnnotations() {
		annotations[k] = v
	}
	meta.SetAnnotations(annotations)
}

var (
	ErrorEntityNotReady = errors.New("entity not ready to be deployed")
	ErrorNilObject      = errors.New("an object was nil, this is not allowed.")
)
