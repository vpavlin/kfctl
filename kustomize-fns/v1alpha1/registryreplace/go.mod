module gitlab.com/vpavlin/kfctl/registryreplace

go 1.15

require (
	sigs.k8s.io/kustomize/v3 v3.2.0
	sigs.k8s.io/kustomize/api v0.7.2
	gopkg.in/yaml.v2 v2.3.0
	sigs.k8s.io/yaml v1.2.0
	golang.org/x/net v0.0.0-20200501053045-e0ff5e5a1de5
)

replace (
	sigs.k8s.io/kustomize/v3 => sigs.k8s.io/kustomize/v3 v3.2.0
	github.com/go-openapi/swag => github.com/go-openapi/swag v0.17.0
	github.com/go-openapi/jsonreference => github.com/go-openapi/jsonreference v0.17.0
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.18.0
	github.com/go-openapi/swag => github.com/go-openapi/swag v0.17.0
	github.com/go-openapi/jsonpointer => github.com/go-openapi/jsonpointer v0.17.0
)
