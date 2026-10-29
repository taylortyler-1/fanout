package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/taylortyler-1/fanout/internal/skill"
)

// loadSkills discovers and validates every skill in repo, reporting all
// problems before failing.
func loadSkills(cmd *cobra.Command, repo string, skip ...string) ([]*skill.Skill, error) {
	skills, perrs, err := skill.Discover(repo, skip...)
	if err != nil {
		return nil, err
	}
	stderr := cmd.ErrOrStderr()
	failed := len(perrs)
	for _, pe := range perrs {
		_, _ = fmt.Fprintf(stderr, "✗ %s\n    %v\n", relPath(pe.Dir), pe.Err)
	}
	seen := map[string]string{}
	for _, s := range skills {
		err := s.Validate()
		if prev, ok := seen[s.Name]; ok && s.Name != "" {
			err = errors.Join(err, fmt.Errorf("duplicate skill name %q (also in %s)", s.Name, relPath(prev)))
		}
		seen[s.Name] = s.Dir
		if err != nil {
			failed++
			_, _ = fmt.Fprintf(stderr, "✗ %s\n    %s\n", relPath(s.Dir), strings.ReplaceAll(err.Error(), "\n", "\n    "))
		}
	}
	if failed > 0 {
		return nil, fmt.Errorf("%d skill(s) failed validation", failed)
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found under %s (looking for %s files)\nRun \"fanout init\" to create a starter skills repo", relPath(repo), skill.FileName)
	}
	return skills, nil
}
