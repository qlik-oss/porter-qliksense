package qliksense

import (
	"fmt"
)

func (m *Mixin) PrintSchema() error {
	schema, err := m.GetSchema()
	if err != nil {
		return err
	}

	fmt.Fprintf(m.Out, schema)

	return nil
}

func (m *Mixin) GetSchema() (string, error) {
	b, err := m.schema.Find("qliksense.json")
	if err != nil {
		return "", err
	}

	return string(b), nil
}
