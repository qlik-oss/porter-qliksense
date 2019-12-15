package qliksense

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	_ "os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// AboutAction The `Porter.sh` action for Qliksense About
type AboutAction struct {
	Steps []AboutStep `yaml:"about"`
}

// AboutStep The `Porter.sh` step for Install for Kustomize
type AboutStep struct {
	AboutArguments `yaml:"qliksense"`
}

// AboutArguments ...
type AboutArguments struct {
	Step    `yaml:",inline"`
	Version string `yaml:"version"`
}

type VersionOutput struct {
	QliksenseVersion string   `yaml:"qlikSenseVersion"`
	Images           []string `yaml:"images"`
}

// About The public method invoked by `porter` when performing an `Install` step that has a `qliksense` mixin step
func (m *Mixin) About() error {
	var (
		payload          []byte
		version          string
		err              error
		action           AboutAction
		versionOut       VersionOutput
		file             *os.File
		realVersion, out []byte
		scanner          *bufio.Scanner
		images           []string
	)

	if payload, err = m.getPayloadData(); err != nil {
		return err
	}
	if err = yaml.Unmarshal(payload, &action); err != nil {
		return err
	}
	if len(action.Steps) != 1 {
		return errors.Errorf("expected a single step, but got %d", len(action.Steps))
	}

	if version = action.Steps[0].AboutArguments.Version; version == "bundled" {
		realVersion, err = ioutil.ReadFile(filepath.Join(chartCache, "VERSION"))

		if file, err = os.Open(filepath.Join(chartCache, "images-"+string(realVersion)+".txt")); err != nil {
			return err
		}
		defer file.Close()

		scanner = bufio.NewScanner(file)
		images = make([]string, 0)
		for scanner.Scan() {
			images = append(images, scanner.Text())
		}
		if err = scanner.Err(); err != nil {
			return err
		}
		versionOut = VersionOutput{
			QliksenseVersion: string(realVersion),
			Images:           images,
		}
		if out, err = yaml.Marshal(versionOut); err != nil {
			return err
		}
		fmt.Println(string(out))
	} else {
		// TODO: Means we are fetching from Git
	}
	return nil
}
