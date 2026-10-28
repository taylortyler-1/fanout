package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/taylortyler-1/fanout/internal/platform"
)

func newPlatformsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "platforms",
		Short: "List builtin target platforms",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ps, err := platform.Builtin()
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, p := range ps {
				if _, err := fmt.Fprintf(w, "%s\t%s\n", p.Name, p.Description); err != nil {
					return err
				}
			}
			return w.Flush()
		},
	}
}
