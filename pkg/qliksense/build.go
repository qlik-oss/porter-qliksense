package qliksense

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
	var (
		version, chartFile, scannedText string
		err                             error
		versionFile, porterDockerFile   *os.File
		scanner                         *bufio.Scanner
		parts                           []string
		porterBytes                     []byte
		porterFileYaml                  porterYaml
	)
	if porterBytes, err = ioutil.ReadFile(porterFile); err != nil {
		return err
	}

	if err = yaml.Unmarshal(porterBytes, &porterFileYaml); err != nil {
		return err
	}
	os.Mkdir(chartCache, os.ModePerm)
	if len(porterFileYaml.Dockerfile) > 0 {
		if porterDockerFile, err = os.Open(porterFileYaml.Dockerfile); err != nil {
			return err
		}
		defer porterDockerFile.Close()

		scanner = bufio.NewScanner(porterDockerFile)

		for scanner.Scan() {
			scannedText = scanner.Text()
			if strings.Contains(scannedText, qseokVersion) {
				parts = strings.Split(scannedText, "=")
				if len(parts) > 1 {
					version = parts[len(parts)-1]
				}
			}
		}
		if err = scanner.Err(); err != nil {
			return err
		}
	}

	fmt.Fprintf(m.Out, dockerfileLines)
	if len(version) == 0 {
		version, _ = GetTransformerVersion()
	}
	if len(version) > 0 {
		if versionFile, err = os.Create(filepath.Join(chartCache, "VERSION")); err != nil {
			return err
		}
		defer versionFile.Close()
		versionFile.WriteString(version)
	}
	if _, err = os.Stat(chartFile); err != nil && !os.IsNotExist(err) {
		return errors.Errorf("Unable to determine chart file %v exists", chartFile)
	}

	if len(version) > 0 {
		chartFile = filepath.Join(chartCache, helmDir, chartName+"-"+version+".tgz")
		fmt.Fprintln(m.Out, strings.ReplaceAll("ADD "+chartFile+" /tmp/.chartcache/", "\\", "/"))
	} else {
		chartFile = filepath.Join(chartCache, helmDir, chartName+"-latest.tgz")
		fmt.Fprintln(m.Out, strings.ReplaceAll("ADD "+filepath.Join(chartCache, helmDir, chartName+"-*.tgz")+" /tmp/.chartcache/", "\\", "/"))
	}
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
		if err != nil {
			if chart.ChartName == chartName {
				return chart.ChartVersion, nil
			}
		}
	}
	return "", nil
}
