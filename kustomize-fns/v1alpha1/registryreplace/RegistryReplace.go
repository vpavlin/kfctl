package main

import (
  //"fmt"
  "strings"

  "sigs.k8s.io/yaml"
	"sigs.k8s.io/kustomize/v3/pkg/types"
  "sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
  "sigs.k8s.io/kustomize/v3/pkg/transformers"
  "sigs.k8s.io/kustomize/v3/pkg/transformers/config"

)

type Registry struct {
  Src string `yaml:"src"`
  Dest string `yaml:"dest"`
}

type Spec struct {
  FieldSpecs []config.FieldSpec `yaml:"fieldSpecs"`
  Registries []Registry `yaml:"registries"`
}

type plugin struct {
  rmf              *resmap.Factory
  ldr              ifc.Loader
  c *resmap.Configurable

	types.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec Spec `yaml:"spec"`
}

//nolint: golint
//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) error {
    p.ldr = ldr
    p.rmf = rf
    return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
  
  for _, fs := range p.Spec.FieldSpecs {
    for _, r := range m.Resources() {
      if r.OrgId().IsSelected(&fs.Gvk) {
        if err := transformers.MutateField(
					r.Map(), fs.PathSlice(),
					false, p.set); err != nil {
					return err
				}
      }
    }
    
  }
  
  return nil
}

func (p *plugin) set(node interface{}) (interface{}, error) {
  for _, registry := range p.Spec.Registries {
    if ok := strings.Contains(node.(string), registry.Src); ok {
      //fmt.Errorf("Will replace %s -> %s\n", node, strings.ReplaceAll(node.(string), registry.Src, registry.Dest))
      return strings.ReplaceAll(node.(string), registry.Src, registry.Dest), nil
    }
  }
  
  return node, nil
}
