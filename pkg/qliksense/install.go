package qliksense

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"os"
	_ "os/exec"
	"strings"
)

// The `Porter.sh` action for Install
type InstallAction struct {
	Steps []InstallStep `yaml:"install"`
}

// The `Porter.sh` step for Install for Kustomize
type InstallStep struct {
	InstallArguments `yaml:"qliksense"`
}

type InstallArguments struct {
	Step `yaml:",inline"`
	Cr   CR `yaml:"cr"`
}
type CR struct {
	AcceptEULA string `yaml:"acceptEULA"`
}

// The public method invoked by `porter` when performing an `Install` step that has a `qliksense` mixin step
func (m *Mixin) Install() error {
	payload, err := m.getPayloadData()
	if err != nil {
		fmt.Println("gooooo, error", err)
		return err
	}
	//fmt.Println(string(payload))
	var action InstallAction
	err = yaml.Unmarshal(payload, &action)
	if err != nil {
		return err
	}
	if len(action.Steps) != 1 {
		return errors.Errorf("expected a single step, but got %d", len(action.Steps))
	}

	step := action.Steps[0]
	m.executeQliksense(&step.Cr)
	for _, output := range step.Outputs {
		err = m.Context.WriteMixinOutputToFile(output.Name, []byte(fmt.Sprintf("%v", output)))
		if err != nil {
			return errors.Wrapf(err, "unable to write output '%s'", output.Name)
		}
	}
	return nil
}

func (m *Mixin) executeQliksense(cr *CR) error {
	fmt.Println("applying patch ...")
	tmpl := `
manifestsRoot: "/cnab/app"
configs:
- dataKey: acceptEULA
  value:
    qliksense: ` + cr.AcceptEULA
	cmd := m.NewCommand("qliksense-operator")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "YAML_CONF="+tmpl)

	cmd.Stdout = m.Out
	cmd.Stderr = m.Err
	prettyCmd := fmt.Sprintf("%s %s", cmd.Path, strings.Join(cmd.Args, " "))
	if m.Debug {
		fmt.Println("DEBUG: " + prettyCmd)
	}
	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
		return errors.Wrap(err, fmt.Sprintf("couldn't run command %s", prettyCmd))
	}
	err = cmd.Wait()
	fmt.Println("applying patch finished.")
	return nil
}
