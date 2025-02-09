// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

//go:generate go run sigs.k8s.io/kustomize/plugin/pluginator
package main

import (
	"errors"
	"fmt"

	"sigs.k8s.io/kustomize/pkg/gvk"
	"sigs.k8s.io/kustomize/pkg/ifc"
	"sigs.k8s.io/kustomize/pkg/resid"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/transformers"
	"sigs.k8s.io/kustomize/pkg/transformers/config"
	"sigs.k8s.io/yaml"
)

// Add the given prefix and suffix to the field.
type plugin struct {
	Prefix     string             `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Suffix     string             `json:"suffix,omitempty" yaml:"suffix,omitempty"`
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

// Not placed in a file yet due to lack of demand.
var prefixSuffixFieldSpecsToSkip = []config.FieldSpec{
	{
		Gvk: gvk.Gvk{Kind: "CustomResourceDefinition"},
	},
}

func (p *plugin) Config(
	ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.Prefix = ""
	p.Suffix = ""
	p.FieldSpecs = nil
	err = yaml.Unmarshal(c, p)
	if err != nil {
		return
	}
	if p.FieldSpecs == nil {
		return errors.New("fieldSpecs is not expected to be nil")
	}
	return
}

func (p *plugin) Transform(m resmap.ResMap) error {
	if len(p.Prefix) == 0 && len(p.Suffix) == 0 {
		return nil
	}
	for _, r := range m.Resources() {
		if p.shouldSkip(r.OrgId()) {
			continue
		}
		fs, ok := p.shouldInclude(r.OrgId())
		if !ok {
			continue
		}
		if smellsLikeANameChange(fs) {
			r.AddNamePrefix(p.Prefix)
			r.AddNameSuffix(p.Suffix)
		}
		err := transformers.MutateField(
			r.Map(),
			fs.PathSlice(),
			fs.CreateIfNotPresent,
			p.addPrefixSuffix)
		if err != nil {
			return err
		}
	}
	return nil
}

func smellsLikeANameChange(fs *config.FieldSpec) bool {
	return fs.Path == "metadata/name"
}

func (p *plugin) shouldInclude(
	id resid.ResId) (*config.FieldSpec, bool) {
	for _, path := range p.FieldSpecs {
		if id.IsSelected(&path.Gvk) {
			return &path, true
		}
	}
	return nil, false
}

func (p *plugin) shouldSkip(
	id resid.ResId) bool {
	for _, path := range prefixSuffixFieldSpecsToSkip {
		if id.IsSelected(&path.Gvk) {
			return true
		}
	}
	return false
}

func (p *plugin) addPrefixSuffix(
	in interface{}) (interface{}, error) {
	s, ok := in.(string)
	if !ok {
		return nil, fmt.Errorf("%#v is expected to be %T", in, s)
	}
	return fmt.Sprintf("%s%s%s", p.Prefix, s, p.Suffix), nil
}
