package qliksense

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"reflect"
	"sort"

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
	Profile string `yaml:"profile"`
}

type VersionOutput struct {
	QliksenseVersion string   `yaml:"qlikSenseVersion"`
	Images           []string `yaml:"images"`
}

// About The public method invoked by `porter` when performing an `Install` step that has a `qliksense` mixin step
func (m *Mixin) About() error {
	var (
		payload              []byte
		realVersion, version string
		err                  error
		action               AboutAction
		versionOut           VersionOutput
		out, kuzManifest     []byte
	)
	if payload, err = m.getPayloadData(); err != nil {
		return err
	}
	if err = yaml.Unmarshal(payload, &action); err != nil {
		return err
	}
	if len(action.Steps) != 1 {
		return fmt.Errorf("expected a single step, but got %d", len(action.Steps))
	}
	if version = action.Steps[0].AboutArguments.Version; version == "bundled" {
		if realVersion, err = GetTransformerVersion(); err != nil {
			log.Printf("error reading the VERSION file, error: %v\n", err)
			return err
		}
		if kuzManifest, err = getKustomizeBuildOutput(action.Steps[0].AboutArguments.Profile); err != nil {
			log.Printf("error executing kustomize, error: %v\n", err)
			return err
		}
		versionOut = VersionOutput{
			QliksenseVersion: string(realVersion),
			Images:           getImageList(kuzManifest),
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

func getKustomizeBuildOutput(path string) ([]byte, error) {
	cmd := exec.Command("kustomize", "build", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("executing kustomize process failed with error: %v, stderr: %v\n", err, string(stderr.Bytes()))
		return nil, err
	}
	return stdout.Bytes(), nil
}

func getImageList(yamlContent []byte) []string {
	decoder := yaml.NewDecoder(bytes.NewReader(yamlContent))
	var resource map[string]interface{}
	imageMap := make(map[string]bool)
	for {
		err := decoder.Decode(&resource)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("error decoding yaml: %v\n", err)
			}
			break
		}
		traverseYamlDecodedMapRecursively(reflect.ValueOf(resource), []string{}, func(path []string, val interface{}) {
			if len(path) >= 2 && path[len(path)-1] == "image" &&
				(path[len(path)-2] == "containers" || path[len(path)-2] == "initContainers") {
				if image, ok := val.(string); ok {
					imageMap[image] = true
				}
			}
		})
	}
	var sortedImageList []string
	for image, _ := range imageMap {
		sortedImageList = append(sortedImageList, image)
	}
	sort.Strings(sortedImageList)
	return sortedImageList
}

func traverseYamlDecodedMapRecursively(val reflect.Value, path []string, visitorFunc func(path []string, val interface{})) {
	kind := val.Kind()
	switch kind {
	case reflect.Interface:
		traverseYamlDecodedMapRecursively(val.Elem(), path, visitorFunc)
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			traverseYamlDecodedMapRecursively(val.Index(i), path, visitorFunc)
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			traverseYamlDecodedMapRecursively(val.MapIndex(key), append(path, key.Interface().(string)), visitorFunc)
		}
	default:
		if kind != reflect.Invalid {
			visitorFunc(path, val.Interface())
		}
	}
}
