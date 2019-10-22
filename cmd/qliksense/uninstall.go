package main

import (
	"github.com/qlik-oss/porter-qliksense/pkg/qliksense"
	"github.com/spf13/cobra"
)

func buildUninstallCommand(m *qliksense.Mixin) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Execute the uninstall functionality of this mixin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return m.Execute()
		},
	}
	return cmd
}
