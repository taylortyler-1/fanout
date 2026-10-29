package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/taylortyler-1/fanout/internal/config"
	"github.com/taylortyler-1/fanout/internal/skill"
)

const exampleSkill = "explain-code"

const configTemplate = `# fanout.yaml: metadata for the plugins fanout builds.
# See https://github.com/taylortyler-1/fanout#configure-your-build
name: %s
version: ` + defaultVersion + `
description: %s
author:
  name: %q
license: MIT
platforms: [claude-code, pi]
`

const skillTemplate = `---
name: ` + exampleSkill + `
description: Explain how code works. Use when walking through unfamiliar code.
---

Explain the code step by step, including important assumptions and edge cases.
`

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a starter skills repo with fanout.yaml and an example skill",
		Long: `Init writes a fanout.yaml and an example skill to dir (default: current
directory), creating it if needed. Existing files are never overwritten.`,
		Example: `  fanout init my-skills
  fanout init`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			return runInit(cmd, dir)
		},
	}
}

func runInit(cmd *cobra.Command, dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	name := kebab(filepath.Base(abs))
	if name == "" {
		name = "my-skills"
	}
	files := map[string]string{
		config.FileName: fmt.Sprintf(configTemplate, name, "Agent Skills packaged with fanout", authorName()),
		filepath.Join(exampleSkill, skill.FileName): skillTemplate,
	}
	// Check everything before writing so a refusal leaves dir untouched.
	for rel := range files {
		if _, err := os.Lstat(filepath.Join(abs, rel)); err == nil {
			return fmt.Errorf("%s already exists; init only creates new repos", relPath(filepath.Join(abs, rel)))
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for rel, content := range files {
		path := filepath.Join(abs, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := createFileExclusive(path, content); err != nil {
			return err
		}
	}

	shown := relPath(abs)
	buildArg := ""
	if shown != "." {
		buildArg = " " + shellQuote(shown)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), `Created %s with:
  %s            plugin name, author, license, and target platforms
  %s/%s  an example skill to edit or replace

Next:
  fanout validate%s
  fanout build%s
`, shown, config.FileName, exampleSkill, skill.FileName, buildArg, buildArg)
	return err
}

// createFileExclusive enforces no-clobber even if a file or symlink appears
// after the preflight check.
func createFileExclusive(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteString(content)
	return errors.Join(writeErr, f.Close())
}

// authorName uses the git identity when available so the generated metadata
// is ready to build without edits.
func authorName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if name := strings.TrimSpace(string(out)); err == nil && name != "" {
		return name
	}
	return "Your Name"
}
