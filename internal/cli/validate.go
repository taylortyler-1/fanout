package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/taylortyler-1/fanout/internal/config"
)

func newValidateCmd() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "validate [skills-repo]",
		Short: "Validate the skills in a repo against the Agent Skills spec",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := "."
			if len(args) == 1 {
				repo = args[0]
			}
			repo, err := filepath.Abs(repo)
			if err != nil {
				return err
			}
			cfg, err := config.Load(repo, configPath)
			if err != nil {
				return err
			}
			skills, err := loadSkills(cmd, repo, cfg.ExcludeDirs(repo)...)
			if err != nil {
				return err
			}
			for _, s := range skills {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "✓ %s (%s)\n", s.Name, relPath(s.Dir)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "config file (default <skills-repo>/fanout.yaml if present)")
	return cmd
}
