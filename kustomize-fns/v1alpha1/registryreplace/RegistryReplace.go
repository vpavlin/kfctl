package main

import (
	//"fmt"

	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/prometheus/common/log"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	imageListFile  = "/tmp/imagesx.yaml"
	configTypeFile = "File"
	configTypeCM   = "ConfigMap"
)

type Registry struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

type Spec struct {
	FieldSpecs      []config.FieldSpec `yaml:"fieldSpecs"`
	Registries      []Registry         `yaml:"registries"`
	ImageListOutput ImageListOutput    `yaml:"imageListOutput"`
}

type plugin struct {
	rmf *resmap.Factory
	ldr ifc.Loader
	c   *resmap.Configurable

	types.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec             Spec        `yaml:"spec"`
	ImageList        []ImagePair `yaml:"imageList"`
}

type ImagePair struct {
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`
}

type FileConfig struct {
	Path string `json:"path" yaml:"path`
}

type ConfigMapConfig struct {
	Name string `json:"path" yaml:"path"`
}

type ImageListOutput struct {
	Type      string          `json:"type" yaml:"type"`
	File      FileConfig      `json:"file,omitempty" yaml:"file,omitempty"`
	ConfigMap ConfigMapConfig `json:"configMap,omitempty" yaml:"configMap,omitempty"`
}

//nolint: golint
//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func main() {
	if len(os.Args) != 2 {
		fmt.Println("received too few args:", os.Args)
		fmt.Println("always invoke this via kustomize plugins")
		os.Exit(1)
	}

	// ignore the first file name argument
	// load the second argument, the file path
	content, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("unable to read in plugin config", os.Args[1])
		os.Exit(1)
	}

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println("could not read stdin")
		os.Exit(1)
	}

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	p := &plugin{}
	err = p.Config(rf, content)
	if err != nil {
		fmt.Printf("could not load plugin config: %v\n", err)
		os.Exit(1)
	}

	rm, err := rf.NewResMapFromBytes(input)
	if err != nil {
		fmt.Printf("could not create resmap from input: %v\n", err)
		os.Exit(1)
	}

	err = p.Transform(rm)
	if err != nil {
		fmt.Printf("transformation failed: %v\n", err)
		os.Exit(1)
	}

	data, err := rm.AsYaml()
	if err != nil {
		fmt.Printf("could not marshal resmap: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(data))

}

func (p *plugin) Config(rf *resmap.Factory, c []byte) error {
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
		if err := writeImageList(p.Spec.ImageListOutput, p.ImageList); err != nil {
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

func writeImageList(imgListOutput ImageListOutput, imageList []ImagePair) error {
	var imageListCurrent []ImagePair

	log.Warn("output: %v", imgListOutput)

	if imgListOutput.Type == configTypeFile {
		if err := loadImgListFile(&imageListCurrent, imgListOutput.File.Path); err != nil {
			return err
		}
	} else if imgListOutput.Type == configTypeCM {
		if err := loadImgListCM(&imageListCurrent, imgListOutput.ConfigMap.Name); err != nil {
			return err
		}
	} else {
		if err := loadImgListFile(&imageListCurrent, imageListFile); err != nil {
			return err
		}
	}

	for _, imgNew := range imageList {
		add := true
		for _, imgOld := range imageListCurrent {
			if imgNew.From == imgOld.From && imgNew.To == imgOld.To {
				add = false
			}
		}
		if add {
			imageListCurrent = append(imageListCurrent, imgNew)
		}
	}

	if imgListOutput.Type == configTypeFile {
		if err := writeImgListFile(&imageListCurrent, imgListOutput.File.Path); err != nil {
			return err
		}
	} else if imgListOutput.Type == configTypeCM {
		if err := writeImgListCM(&imageListCurrent, imgListOutput.ConfigMap.Name); err != nil {
			return err
		}
	} else {
		if err := writeImgListFile(&imageListCurrent, imageListFile); err != nil {
			return err
		}
	}

	return nil
}

func loadImgListFile(imageListCurrent *[]ImagePair, path string) error {
	if _, err := os.Stat(path); err == nil {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(data, &imageListCurrent); err != nil {
			return err
		}
	}

	return nil
}

func writeImgListFile(imageListCurrent *[]ImagePair, path string) error {
	var imageListYaml []byte
	if len(*imageListCurrent) > 0 {
		var err error
		imageListYaml, err = yaml.Marshal(imageListCurrent)
		if err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(path, imageListYaml, 0644); err != nil {
		return err
	}

	return nil
}

func loadImgListCM(imageListCurrent *[]ImagePair, cm string) error {
	fmt.Printf("ConfigMap not implemented yet: %v\n", cm)
	return nil
}

func writeImgListCM(imageListCurrent *[]ImagePair, cm string) error {
	fmt.Printf("ConfigMap not implemented yet: %v\n", cm)
	return nil
}
