// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package resources

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"

	workloadv1 "github.com/vmware-tanzu-labs/operator-builder/internal/workload/v1"
)

var _ machinery.Template = &Definition{}

// Types scaffolds the child resource definition files.
type Definition struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	machinery.ResourceMixin

	ClusterScoped bool
	SourceFile    workloadv1.SourceFile
	PackageName   string
	SpecFields    []*workloadv1.APISpecField
	IsComponent   bool
	Collection    *workloadv1.WorkloadCollection
}

func (f *Definition) SetTemplateDefaults() error {
	f.Path = filepath.Join(
		"apis",
		f.Resource.Group,
		f.Resource.Version,
		f.PackageName,
		f.SourceFile.Filename,
	)

	f.TemplateBody = definitionTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const definitionTemplate = `{{ .Boilerplate }}

package {{ .PackageName }}

import (
	{{ if .SourceFile.HasStatic }}
	"text/template"
	{{ end }}
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	{{- if .SourceFile.HasStatic }}
	k8s_yaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	{{ end }}

	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- if .IsComponent }}
	{{ .Collection.Spec.API.Group }}{{ .Collection.Spec.API.Version }} "{{ .Repo }}/apis/{{ .Collection.Spec.API.Group }}/{{ .Collection.Spec.API.Version }}"
	{{ end -}}
)

{{ range .SourceFile.Children }}
// Create{{ .UniqueName }} creates the {{ .Name }} {{ .Kind }} resource.
func Create{{ .UniqueName }} (
	parent *{{ $.Resource.ImportAlias }}.{{ $.Resource.Kind }},
	{{- if $.IsComponent }}
	collection *{{ $.Collection.Spec.API.Group }}{{ $.Collection.Spec.API.Version }}.{{ $.Collection.Spec.API.Kind }},
	{{ end -}}
) (metav1.Object, error) {
	{{- .SourceCode }}

	{{ if not $.ClusterScoped }}
	resourceObj.SetNamespace(parent.Namespace)
	{{ end }}

	return resourceObj, nil
}
{{ end }}
`
