// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package controller

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"

	"github.com/vmware-tanzu-labs/operator-builder/internal/utils"
	workloadv1 "github.com/vmware-tanzu-labs/operator-builder/internal/workload/v1"
)

var _ machinery.Template = &Controller{}

// Controller scaffolds the workload's controller.
type Controller struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	machinery.ResourceMixin

	PackageName       string
	RBACRules         *[]workloadv1.RBACRule
	OwnershipRules    *[]workloadv1.OwnershipRule
	HasChildResources bool
	IsStandalone      bool
	IsComponent       bool
	Collection        *workloadv1.WorkloadCollection
}

func (f *Controller) SetTemplateDefaults() error {
	f.Path = filepath.Join(
		"controllers",
		f.Resource.Group,
		fmt.Sprintf("%s_controller.go", utils.ToFileName(f.Resource.Kind)),
	)

	f.TemplateBody = controllerTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint: lll
const controllerTemplate = `{{ .Boilerplate }}

package {{ .Resource.Group }}

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"{{ .Repo }}/apis/common"
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{ if .IsComponent -}}
	{{ .Collection.Spec.API.Group }}{{ .Collection.Spec.API.Version }} "{{ .Repo }}/apis/{{ .Collection.Spec.API.Group }}/{{ .Collection.Spec.API.Version }}"
	{{ end }}
	{{- if .HasChildResources -}}
	"{{ .Resource.Path }}/{{ .PackageName }}"
	{{ end -}}
	"{{ .Repo }}/internal/controllers/phases"
	"{{ .Repo }}/internal/controllers/utils"
	"{{ .Repo }}/internal/dependencies"
	"{{ .Repo }}/internal/mutate"
	"{{ .Repo }}/internal/resources"
	"{{ .Repo }}/internal/wait"
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object.
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Name       string
	Log        logr.Logger
	Context    context.Context
	Controller controller.Controller
	Watches    []client.Object
	Component  *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}
	{{- if .IsComponent }}
	Collection *{{ .Collection.Spec.API.Group }}{{ .Collection.Spec.API.Version }}.{{ .Collection.Spec.API.Kind }}
	{{ end }}
}


func New{{ .Resource.Kind }}Reconciler(mgr ctrl.Manager) *{{ .Resource.Kind }}Reconciler {
	return &{{ .Resource.Kind }}Reconciler{
		Name:      "{{ .Resource.Kind }}",
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("{{ .Resource.Group }}").WithName("{{ .Resource.Kind }}"),
		Component: &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{},
	}
}

// +kubebuilder:rbac:groups={{ .Resource.Group }}.{{ .Resource.Domain }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.Group }}.{{ .Resource.Domain }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch
{{ range .RBACRules -}}
// +kubebuilder:rbac:groups={{ .Group }},resources={{ .Resource }},verbs={{ .VerbString }}
{{ end }}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WebApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *{{ .Resource.Kind }}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Context = ctx
	log := r.Log.WithValues("{{ .Resource.Kind | lower }}", req.NamespacedName)

	// get and store the component
	if err := r.Get(r.Context, req.NamespacedName, r.Component); err != nil {
		if !apierrs.IsNotFound(err) {
			log.V(0).Error(
				err, "unable to fetch resource",
				"kind", "{{ .Resource.Kind }}",
			)

			return ctrl.Result{}, fmt.Errorf("unable to fetch resource, %w", err)
		}

		log.V(0).Info("unable to fetch {{ .Resource.Kind }}")

		return ctrl.Result{}, fmt.Errorf("resource not found, %w", err)
	}

	{{ if .IsComponent }}
	// get and store the collection
	var collectionList {{ .Collection.Spec.API.Group }}{{ .Collection.Spec.API.Version }}.{{ .Collection.Spec.API.Kind }}List

	if err := r.List(r.Context, &collectionList); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to list collection {{ .Collection.Spec.API.Kind }}, %w", err)
	}

	if len(collectionList.Items) == 0 {
		log.V(0).Info("no collections available; initiating controller requeue")

		return ctrl.Result{Requeue: true}, nil
	} else if len(collectionList.Items) > 1 {
		log.V(0).Info("multiple collections found; expected 1; cannot proceed")

		return ctrl.Result{}, nil
	}

	r.Collection = &collectionList.Items[0]
	{{ end }}

	// execute the phases
	for _, phase := range utils.Phases(r.Component) {
		log.V(7).Info(
			"enter phase",
			"phase", reflect.TypeOf(phase).String(),
		)

		proceed, err := phase.Execute(r)
		result, err := phases.HandlePhaseExit(r, phase, proceed, err)

		if err != nil || !proceed {
			log.V(2).Info(
				"not ready; requeuing",
				"phase", reflect.TypeOf(phase),
			)

			// return only if we have an error or are told not to proceed
			if err != nil {
				return result, fmt.Errorf("unable to complete %T phase for %s, %w", phase, r.Component.GetName(), err)
			}

			if !proceed {
				return result, nil
			}
		}

		log.V(5).Info(
			"completed phase",
			"phase", reflect.TypeOf(phase).String(),
		)
	}

	return phases.DefaultReconcileResult(), nil
}

// GetResources resources runs the methods to properly construct the resources in memory.
func (r *{{ .Resource.Kind }}Reconciler) GetResources() ([]client.Object, error) {
	{{- if .HasChildResources }}
	resourceObjects := []client.Object{}

	// create resources in memory
	for _, f := range {{ .PackageName }}.CreateFuncs {
		resource, err := f(r.Component{{ if .IsComponent }}, r.Collection){{ else }}){{ end }}
		if err != nil {
			return nil, err
		}

		// run through the mutation functions to mutate the resources
		mutatedResources, skip, err := r.Mutate(resource)
		if err != nil {
			return []client.Object{}, err
		}

		if skip {
			continue
		}

		resourceObjects = append(resourceObjects, mutatedResources...)
	}

	return resourceObjects, nil
{{- else -}}
	return []client.Object{}, nil
{{ end -}}
}

// CreateOrUpdate creates a resource if it does not already exist or updates a resource
// if it does already exist.
func (r *{{ .Resource.Kind }}Reconciler) CreateOrUpdate(resource client.Object) error {
	// set ownership on the underlying resource being created or updated
	if err := ctrl.SetControllerReference(r.Component, resource, r.Scheme()); err != nil {
		r.GetLogger().V(0).Error(
			err, "unable to set owner reference on resource",
			"name", resource.GetName(),
			"namespace", resource.GetNamespace(),
		)

		return fmt.Errorf("unable to set owner reference on %s, %w", resource.GetName(), err)
	}

	// get the resource from the cluster
	clusterResource, err := resources.Get(r, resource)
	if err != nil {
		return fmt.Errorf("unable to retrieve resource %s, %w", resource.GetName(), err)
	}

	// create the resource if we have a nil object, or update the resource if we have one
	// that exists in the cluster already
	if clusterResource == nil {
		if err := resources.Create(r, resource); err != nil {
			return fmt.Errorf("unable to create resource %s, %w", resource.GetName(), err)
		}
	} else {
		if err := resources.Update(r, resource, clusterResource); err != nil {
			return fmt.Errorf("unable to update resource %s, %w", resource.GetName(), err)
		}
	}

	return utils.Watch(r, resource)
}

// GetLogger returns the logger from the reconciler.
func (r *{{ .Resource.Kind }}Reconciler) GetLogger() logr.Logger {
	return r.Log
}

// GetContext returns the context from the reconciler.
func (r *{{ .Resource.Kind }}Reconciler) GetContext() context.Context {
	return r.Context
}

// GetName returns the name of the reconciler.
func (r *{{ .Resource.Kind }}Reconciler) GetName() string {
	return r.Name
}

// GetComponent returns the component the reconciler is operating against.
func (r *{{ .Resource.Kind }}Reconciler) GetComponent() common.Component {
	return r.Component
}

// GetController returns the controller object associated with the reconciler.
func (r *{{ .Resource.Kind }}Reconciler) GetController() controller.Controller {
	return r.Controller
}

// GetWatches returns the objects which are current being watched by the reconciler.
func (r *{{ .Resource.Kind }}Reconciler) GetWatches() []client.Object {
	return r.Watches
}

// SetWatch appends a watch to the list of currently watched objects.
func (r *{{ .Resource.Kind }}Reconciler) SetWatch(watch client.Object) {
	r.Watches = append(r.Watches, watch)
}

// CheckReady will return whether a component is ready.
func (r *{{ .Resource.Kind }}Reconciler) CheckReady() (bool, error) {
	return dependencies.{{ .Resource.Kind }}CheckReady(r)
}

// Mutate will run the mutate phase of a resource.
func (r *{{ .Resource.Kind }}Reconciler) Mutate(
	object client.Object,
) ([]client.Object, bool, error) {
	return mutate.{{ .Resource.Kind }}Mutate(r, object)
}

// Wait will run the wait phase of a resource.
func (r *{{ .Resource.Kind }}Reconciler) Wait(
	object client.Object,
) (bool, error) {
	return wait.{{ .Resource.Kind }}Wait(r, object)
}

func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	options := controller.Options{
		RateLimiter: utils.NewDefaultRateLimiter(5*time.Microsecond, 5*time.Minute),
	}

	baseController, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		WithEventFilter(utils.ComponentPredicates()).
		For(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
		Build(r)
	if err != nil {
		return fmt.Errorf("unable to setup controller, %w", err)
	}

	r.Controller = baseController

	return nil
}
`
