package main

import (
	"github.com/qlik-oss/porter-qliksense/pkg/qliksense"
	"github.com/spf13/cobra"
)

func buildSchemaCommand(m *qliksense.Mixin) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the json schema for the mixin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return m.PrintSchema()
		},
	}
	return cmd
}
