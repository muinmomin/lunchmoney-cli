package cli

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "lm",
		Short:         "Lunch Money CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(newTxCmd())
	rootCmd.AddCommand(newCategoryCmd())
	rootCmd.AddCommand(newBalancesCmd())

	return rootCmd
}
