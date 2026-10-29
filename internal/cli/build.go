package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/taylortyler-1/fanout/internal/config"
	"github.com/taylortyler-1/fanout/internal/platform"
	"github.com/taylortyler-1/fanout/internal/plugin"
	"github.com/taylortyler-1/fanout/internal/skill"
)

const defaultVersion = "0.1.0"

type buildOptions struct {
	configPath   string
	platforms    []string
	platformDirs []string
	outDir       string
	force        bool
	skills       []string
	name         string
	version      string
	description  string
}

func newBuildCmd() *cobra.Command {
	var o buildOptions
	cmd := &cobra.Command{
		Use:   "build [skills-repo]",
		Short: "Build plugins from a skills repo",
		Long: `Build discovers every SKILL.md under the skills repo (default: current
directory), validates each skill, and writes one plugin per platform to
<out>/<platform>.

Use "all" as a platform to build every builtin platform, and
--platform-dir to build from a custom platform template directory.`,
		Example: `  fanout build
  fanout build ./my-skills -p claude-code -p pi -o dist
  fanout build -p all --name my-skills --plugin-version 1.2.0 --force`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := "."
			if len(args) == 1 {
				repo = args[0]
			}
			return runBuild(cmd, repo, o)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.configPath, "config", "c", "", "config file (default <skills-repo>/fanout.yaml if present)")
	f.StringSliceVarP(&o.platforms, "platform", "p", nil, `target platform, repeatable or comma-separated; "all" for every builtin (default from config, else claude-code)`)
	f.StringSliceVar(&o.platformDirs, "platform-dir", nil, "custom platform definition directory, repeatable")
	f.StringVarP(&o.outDir, "out", "o", "dist", "output directory; each platform is written to <out>/<platform>")
	f.BoolVarP(&o.force, "force", "f", false, "replace existing platform output directories")
	f.StringSliceVarP(&o.skills, "skill", "s", nil, "only include these skills, repeatable (default from config, else all)")
	f.StringVar(&o.name, "name", "", "plugin name (default from config, else the repo directory name)")
	f.StringVar(&o.version, "plugin-version", "", "plugin version (default from config, else "+defaultVersion+")")
	f.StringVar(&o.description, "description", "", "plugin description (default from config)")
	return cmd
}

func runBuild(cmd *cobra.Command, repo string, o buildOptions) error {
	repo, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	cfg, err := config.Load(repo, o.configPath)
	if err != nil {
		return err
	}
	meta := cfg.Metadata
	if o.name != "" {
		meta.Name = o.name
	}
	if meta.Name == "" {
		meta.Name = kebab(filepath.Base(repo))
	}
	if o.version != "" {
		meta.Version = o.version
	}
	if meta.Version == "" {
		meta.Version = defaultVersion
	}
	if o.description != "" {
		meta.Description = o.description
	}
	if err := meta.Validate(); err != nil {
		return err
	}

	platforms, err := resolvePlatforms(orDefault(o.platforms, cfg.Platforms), o.platformDirs)
	if err != nil {
		return err
	}

	outDir, err := filepath.Abs(o.outDir)
	if err != nil {
		return err
	}
	skills, err := loadSkills(cmd, repo, append(cfg.ExcludeDirs(repo), outDir)...)
	if err != nil {
		return err
	}
	skills, err = selectSkills(skills, orDefault(o.skills, cfg.Skills))
	if err != nil {
		return err
	}

	// Check every target before writing or deleting anything.
	for _, p := range platforms {
		dst := filepath.Join(outDir, p.Name)
		if err := checkOutput(dst, repo, skills, o.force); err != nil {
			return err
		}
	}
	for _, p := range platforms {
		dst := filepath.Join(outDir, p.Name)
		if err := buildPlatform(p, meta, skills, dst); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "built %s %s@%s with %d skill(s) -> %s\n",
			p.Name, meta.Name, meta.Version, len(skills), relPath(dst)); err != nil {
			return err
		}
	}
	return printTryCommands(cmd, platforms, outDir)
}

// tryCommands load a builtin platform's output for one session.
var tryCommands = map[string]string{
	"claude-code": "claude --plugin-dir %s",
	"pi":          "pi -e %s",
}

func printTryCommands(cmd *cobra.Command, platforms []*platform.Platform, outDir string) error {
	var lines []string
	for _, p := range platforms {
		format, ok := tryCommands[p.Name]
		if !ok {
			continue
		}
		dst := relPath(filepath.Join(outDir, p.Name))
		if !filepath.IsAbs(dst) {
			dst = "." + string(filepath.Separator) + dst
		}
		lines = append(lines, "  "+fmt.Sprintf(format, shellQuote(dst)))
	}
	if len(lines) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "\nTry it for one session:\n%s\n", strings.Join(lines, "\n"))
	return err
}

func selectSkills(all []*skill.Skill, names []string) ([]*skill.Skill, error) {
	if len(names) == 0 {
		return all, nil
	}
	var out []*skill.Skill
	for _, n := range names {
		i := slices.IndexFunc(all, func(s *skill.Skill) bool { return s.Name == n })
		if i < 0 {
			return nil, fmt.Errorf("skill %q not found", n)
		}
		if !slices.Contains(out, all[i]) {
			out = append(out, all[i])
		}
	}
	return out, nil
}

func resolvePlatforms(names, dirs []string) ([]*platform.Platform, error) {
	if len(names) == 0 && len(dirs) == 0 {
		names = []string{"claude-code"}
	}
	var ps []*platform.Platform
	add := func(p *platform.Platform) error {
		if slices.ContainsFunc(ps, func(q *platform.Platform) bool { return q.Name == p.Name }) {
			return fmt.Errorf("platform %q selected more than once", p.Name)
		}
		ps = append(ps, p)
		return nil
	}
	if slices.Contains(names, "all") {
		builtin, err := platform.Builtin()
		if err != nil {
			return nil, err
		}
		for _, p := range builtin {
			if err := add(p); err != nil {
				return nil, err
			}
		}
		names = slices.DeleteFunc(slices.Clone(names), func(n string) bool { return n == "all" })
		names = slices.DeleteFunc(names, func(n string) bool {
			return slices.ContainsFunc(builtin, func(p *platform.Platform) bool { return p.Name == n })
		})
	}
	for _, n := range names {
		p, err := platform.Lookup(n)
		if err != nil {
			return nil, err
		}
		if err := add(p); err != nil {
			return nil, err
		}
	}
	for _, d := range dirs {
		p, err := platform.LoadDir(d)
		if err != nil {
			return nil, err
		}
		if err := add(p); err != nil {
			return nil, err
		}
	}
	return ps, nil
}

// canonicalPath resolves symlinks even when the final path does not exist yet.
func canonicalPath(p string) (string, error) {
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(p)
	if err == nil {
		return real, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent := filepath.Dir(p)
	if parent == p {
		return "", err
	}
	real, err = canonicalPath(parent)
	return filepath.Join(real, filepath.Base(p)), err
}

func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// checkOutput never mutates the filesystem. Resolve aliases before checking
// containment so --force cannot delete source skills through a symlink.
func checkOutput(dst, repo string, skills []*skill.Skill, force bool) error {
	realDst, err := canonicalPath(dst)
	if err != nil {
		return err
	}
	realRepo, err := canonicalPath(repo)
	if err != nil {
		return err
	}
	if containsPath(realDst, realRepo) {
		return fmt.Errorf("output directory %s would contain the skills repo; choose another --out", relPath(dst))
	}
	for _, s := range skills {
		realSkill, err := canonicalPath(s.Dir)
		if err != nil {
			return err
		}
		// Staging happens next to dst, so its parent must not be inside a skill.
		if containsPath(realDst, realSkill) || containsPath(realSkill, filepath.Dir(realDst)) || containsPath(realSkill, realDst) {
			return fmt.Errorf("output directory %s overlaps skill %s; choose another --out", relPath(dst), s.Name)
		}
	}
	entries, err := os.ReadDir(dst)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("output directory %s is not empty; use --force to replace it", relPath(dst))
	}
	return nil
}

// Render and copy completely before replacing an existing plugin. A template
// or copy failure leaves the previous output intact. Publication is per platform.
func buildPlatform(p *platform.Platform, meta plugin.Metadata, skills []*skill.Skill, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(dst), ".fanout-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(stage) }()
	if err := plugin.Build(p, meta, skills, stage); err != nil {
		return err
	}
	if err := os.Chmod(stage, 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return os.Rename(stage, dst)
}

// orDefault returns a if it is non-empty, else b.
func orDefault(a, b []string) []string {
	if len(a) > 0 {
		return a
	}
	return b
}
