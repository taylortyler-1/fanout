// Package cli implements the fanout command line.
package cli

import (
	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X .../internal/cli.version=...".
var version = "dev"

// NewRootCmd returns the fanout root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fanout",
		Short: "Package a repo of Agent Skills into platform-specific plugins",
		Long: `fanout takes a repository of Agent Skills (https://agentskills.io) and
packages them into plugins for specific agent platforms.

Plugin metadata is read from an optional fanout.yaml in the repo root
and can be overridden with flags.`,
		Version:      version,
		SilenceUsage: true,
	}
	root.AddCommand(newBuildCmd(), newValidateCmd(), newPlatformsCmd())
	return root
}
