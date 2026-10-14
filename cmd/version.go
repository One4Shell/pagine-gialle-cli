package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/h4shell/paginegialle-cli/internal/version"
)

func init() {
	rootCmd.AddCommand(newVersionCmd())
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(os.Stdout, "%s version %s (commit %s, built %s)\n",
				rootCmd.Name(), version.Version, version.Commit, version.Date)
			return err
		},
	}
}
