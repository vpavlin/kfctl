package main_test

import (
	"testing"

	"sigs.k8s.io/kustomize/v3/pkg/kusttest"
	"sigs.k8s.io/kustomize/v3/pkg/plugins/testenv"
)

var (
	config = `
apiVersion: v1alpha1
kind: RegistryReplace
metadata:
  name: myTransformer
spec:
  fieldSpecs:
  - kind: Deployment
    group: apps
    version: v1
    path: spec/template/spec/containers[]/image
  - kind: ImageStream
    group: image.openshift.io
    version: v1
    path: spec/tags[]/from/name
  - kind: ImageStream
    group: image.openshift.io
    version: v1
    path: spec/tags[]/annotations/openshift.io\/imported-from
  - kind: ConfigMap
    version: v1
    path: data/image
  - kind: ConfigMap
    version: v1
    path: data/someTextField
  registries:
  - src: quay.io
    dest: private-registry.example.com
  - src: docker.io
    dest: private-registry.example.com
`

	input = `
apiVersion: image.openshift.io/v1
kind: ImageStream
metadata:
  name: s2i-minimal-notebook
spec:
  tags:
  - annotations:
      openshift.io/imported-from: quay.io/thoth-station/s2i-minimal-notebook
    from:
      name: quay.io/thoth-station/s2i-minimal-notebook:v0.0.4
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      containers:
      - image: quay.io/someimage/myapp:somerevision
      - image: docker.io/someimage/imagename:somerevision
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp2
spec:
  template:
    spec:
      containers:
      - image: docker.io/somerepo/myapp:somerevision
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm
data:
  image: quay.io/someimage/imagename
  someTextField: |
    image: quay.io/someimage/imageinstring
`
	expectedOutput = `
apiVersion: image.openshift.io/v1
kind: ImageStream
metadata:
  name: s2i-minimal-notebook
spec:
  tags:
  - annotations:
      openshift.io/imported-from: private-registry.example.com/thoth-station/s2i-minimal-notebook
    from:
      name: private-registry.example.com/thoth-station/s2i-minimal-notebook:v0.0.4
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      containers:
      - image: private-registry.example.com/someimage/myapp:somerevision
      - image: private-registry.example.com/someimage/imagename:somerevision
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp2
spec:
  template:
    spec:
      containers:
      - image: private-registry.example.com/somerepo/myapp:somerevision
---
apiVersion: v1
data:
  image: private-registry.example.com/someimage/imagename
  someTextField: |
    image: private-registry.example.com/someimage/imageinstring
kind: ConfigMap
metadata:
  name: cm
`
)

func TestChecksumerTransformer(t *testing.T) {
	//currentWorkingDirectory, _ := os.Getwd()
	tc := testenv.NewEnvForTest(t).Set()
	tc.BuildGoPlugin("", "v1alpha1", "RegistryReplace")
    defer tc.Reset()
    
    th := kusttest_test.NewKustTestPluginHarness(t, "/app")

    rm := th.LoadAndRunTransformer(config, input)

	th.AssertActualEqualsExpected(rm, expectedOutput)
}
