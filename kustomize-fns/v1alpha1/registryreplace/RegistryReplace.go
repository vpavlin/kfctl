package main

import (
	//"fmt"

	"io/ioutil"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

type Registry struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

type Spec struct {
	FieldSpecs []config.FieldSpec `yaml:"fieldSpecs"`
	Registries []Registry         `yaml:"registries"`
}

type plugin struct {
	rmf *resmap.Factory
	ldr ifc.Loader
	c   *resmap.Configurable

	types.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec             Spec        `yaml:"spec"`
	ImageList        []ImagePair `yaml:imageList`
}

type ImagePair struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
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

	if len(p.ImageList) > 0 {
		if err := writeImageList("/tmp/images.yaml", p.ImageList); err != nil {
			return err
		}
	}

	return nil
}

func (p *plugin) set(node interface{}) (interface{}, error) {
	for _, registry := range p.Spec.Registries {
		if ok := strings.Contains(node.(string), registry.Src); ok {
			//fmt.Errorf("Will replace %s -> %s\n", node, strings.ReplaceAll(node.(string), registry.Src, registry.Dest))
			newImage := strings.ReplaceAll(node.(string), registry.Src, registry.Dest)
			p.ImageList = append(p.ImageList, ImagePair{From: node.(string), To: newImage})
			return newImage, nil

		}
	}

	return node, nil
}

func writeImageList(path string, imageList []ImagePair) error {
	var imageListCurrent []ImagePair
	if _, err := os.Stat(path); err == nil {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(data, &imageListCurrent); err != nil {
			return err
		}
	}

	var newImages []ImagePair
	for _, imgNew := range imageList {
		add := true
		for _, imgOld := range imageListCurrent {
			if imgNew.From == imgOld.From && imgNew.To == imgOld.To {
				add = false
			}
		}
		if add {
			newImages = append(newImages, imgNew)
		}
	}

	var imageListYaml []byte
	if len(newImages) > 0 {
		var err error
		imageListYaml, err = yaml.Marshal(&newImages)
		if err != nil {
			return err
		}
	}

	f, err := os.OpenFile(path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(imageListYaml); err != nil {
		return err
	}

	return nil
}
