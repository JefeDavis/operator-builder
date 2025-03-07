// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package cli

import (
	"path/filepath"
	"text/template"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"

	"github.com/vmware-tanzu-labs/operator-builder/internal/utils"
)

var _ machinery.Template = &Main{}

// Main scaffolds the main package for the companion CLI.
type Main struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin

	// RootCmd is the root command for the companion CLI
	RootCmd        string
	RootCmdVarName string
}

func (f *Main) SetTemplateDefaults() error {
	f.Path = filepath.Join("cmd", f.RootCmd, "main.go")

	f.TemplateBody = cliMainTemplate

	return nil
}

func (*Main) GetFuncMap() template.FuncMap {
	return utils.RemoveStringHelper()
}

const cliMainTemplate = `{{ .Boilerplate }}

package main

import (
	"{{ .Repo }}/cmd/{{ .RootCmd }}/commands"
)

func main() {
	{{ .RootCmd | removeString "-" }} := commands.New{{ .RootCmdVarName }}Command()
	{{ .RootCmd | removeString "-" }}.Run()
}
`
