package qliksense

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/cli"
)

// This is an example. Replace thie following with whatever steps are needed to
// install required components into

const (
	dockerfileLines = `
RUN echo "deb http://deb.debian.org/debian stretch-backports main" >> /etc/apt/sources.list && \
    apt-get update && \
    apt-get install --no-install-recommends libgpgme11-dev libassuan-dev libbtrfs-dev libdevmapper-dev rdfind -y && \
    rm -rf /var/lib/apt/lists/*
COPY --from=qlik/qliksense-cloud-tools:latest /usr/local/bin /usr/local/bin
COPY --from=qlik/qliksense-operator:latest /usr/local/bin/qliksense-operator /usr/local/bin

`
	chartName    = "qliksense"
	chartCache   = ".chartcache"
	qseokVersion = "QSEOK_VERSION"
	porterFile   = "porter.yaml"
)

var (
	settings    *cli.EnvSettings
	helmDir     = filepath.Join("helm", "repository")
	versionFile = filepath.Join("transformers", "qseokversion.yaml")
)

type porterYaml struct {
	Dockerfile string `yaml:"dockerfile"`
}

type patch struct {
	Target struct {
		Kind          string `yaml:"kind"`
		LabelSelector string `yaml:"labelSelector"`
	} `yaml:"target"`
	Patch string `yaml:"patch"`
}
type selectivePatch struct {
	APIVersion string `yaml:"apiVersion"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Enabled bool    `yaml:"enabled"`
	Patches []patch `yaml:"patches"`
}

type helmChart struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	ReleaseNamespace string `yaml:"releaseNamespace"`
	ChartHome        string `yaml:"chartHome"`
	ChartRepo        string `yaml:"chartRepo"`
	ChartName        string `yaml:"chartName"`
	ChartVersion     string `yaml:"chartVersion"`
}

// Build will generate the necessary Dockerfile lines
// for an invocation image using this mixin
func (m *Mixin) Build() error {

	fmt.Fprintf(m.Out, dockerfileLines)
	return nil
}

// GetTransformerVersion ...
func GetTransformerVersion() (string, error) {
	var patchInst patch
	var bytes []byte
	var err error
	var selPatch selectivePatch
	var chart helmChart

	if _, err = os.Stat(versionFile); err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", errors.Errorf("Unable to determine About file %v exists", versionFile)
	}
	if bytes, err = ioutil.ReadFile(versionFile); err != nil {
		return "", err
	}
	if err = yaml.Unmarshal(bytes, &selPatch); err != nil {
		return "", err
	}
	for _, patchInst = range selPatch.Patches {
		err = yaml.Unmarshal([]byte(patchInst.Patch), &chart)
		if err == nil {
			if chart.ChartName == chartName {
				return chart.ChartVersion, nil
			}
		}
	}
	return "", nil
}
