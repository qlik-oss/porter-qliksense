module github.com/qlik-oss/porter-qliksense

go 1.13

require (
	github.com/Azure/go-autorest v11.1.2+incompatible // indirect
	github.com/PaesslerAG/gval v1.0.1 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/deislabs/porter v0.17.0-beta.1
	github.com/ghodss/yaml v1.0.0
	github.com/gobuffalo/envy v1.8.1 // indirect
	github.com/gobuffalo/logger v1.0.3 // indirect
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/gofrs/flock v0.7.1
	github.com/pkg/errors v0.8.1
	github.com/qlik-oss/qliksense-operator v0.2.0
	github.com/rogpeppe/go-internal v1.5.0 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/crypto v0.0.0-20191202143827-86a70503ff7e // indirect
	golang.org/x/sys v0.0.0-20191128015809-6d18c012aee9 // indirect
	gopkg.in/yaml.v2 v2.2.7
	helm.sh/helm/v3 v3.0.0
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
)
