//go:generate packr2
package qliksense

import (
	"bufio"
	"github.com/deislabs/porter/pkg/context"
	"github.com/ghodss/yaml" // We are not using go-yaml because of serialization problems with jsonschema, don't use this library elsewhere
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"strings"
)

type Mixin struct {
	*context.Context
	//add whatever other context/state is needed here
}

// New azure mixin client, initialized with useful defaults.
func New() (*Mixin, error) {
	return &Mixin{
		Context: context.New(),
	}, nil

}

func (m *Mixin) getPayloadData() ([]byte, error) {
	reader := bufio.NewReader(m.In)
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "could not read the payload from STDIN")
	}
	err = m.ValidatePayload(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m *Mixin) ValidatePayload(b []byte) error {
	// Load the step as a go dump
	s := make(map[string]interface{})
	err := yaml.Unmarshal(b, &s)
	if err != nil {
		return errors.Wrap(err, "could not marshal payload as yaml")
	}
	manifestLoader := gojsonschema.NewGoLoader(s)

	// Load the step schema
	schema, err := m.GetSchema()
	if err != nil {
		return err
	}
	schemaLoader := gojsonschema.NewStringLoader(schema)

	validator, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return errors.Wrap(err, "unable to compile the mixin step schema")
	}

	// Validate the manifest against the schema
	result, err := validator.Validate(manifestLoader)
	if err != nil {
		return errors.Wrap(err, "unable to validate the mixin step schema")
	}
	if !result.Valid() {
		errs := make([]string, 0, len(result.Errors()))
		for _, err := range result.Errors() {
			errs = append(errs, err.String())
		}
		return errors.New(strings.Join(errs, "\n\t* "))
	}

	return nil
}
