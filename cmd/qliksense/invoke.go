package main

import (
	"github.com/qlik-oss/porter-qliksense/pkg/qliksense"
	"github.com/spf13/cobra"
)

func buildInvokeCommand(m *qliksense.Mixin) *cobra.Command {
	var action string
	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Execute the invoke functionality of this mixin",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch action {
			case "about":
				return m.About()
			default:
				return m.Execute()
			}
		},
	}

	// Define a flag for --action so that its presence doesn't cause errors, but ignore it since exec doesn't need it

	cmd.Flags().StringVar(&action, "action", "", "Custom action name to invoke.")
	return cmd
}
