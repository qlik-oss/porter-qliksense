package qliksense

import (
	_ "os/exec"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

type UpgradeAction struct {
	Steps []UpgradeStep `yaml:"upgrade"`
}

// UpgradeStep represents the structure of an Upgrade action
type UpgradeStep struct {
	UpgradeArguments `yaml:"qliksense"`
}

// UpgradeArguments are the arguments available for the Upgrade action
type UpgradeArguments struct {
	Step `yaml:",inline"`
	Cr   CR `yaml:"cr"`
}

// Upgrade deletes a provided set of Kustomize releases, supplying optional flags/params
func (m *Mixin) Upgrade() error {
	payload, err := m.getPayloadData()
	if err != nil {
		return err
	}

	var action UpgradeAction
	err = yaml.Unmarshal(payload, &action)
	if err != nil {
		return err
	}
	if len(action.Steps) != 1 {
		return errors.Errorf("expected a single step, but got %d", len(action.Steps))
	}
	step := action.Steps[0]
	m.executeQliksense(&step.Cr)
	return nil
}
