// Package skill discovers, parses, and validates Agent Skills
// (https://agentskills.io/specification).
package skill

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/taylortyler-1/fanout/internal/fsutil"
)

// FileName is the required entrypoint file of every skill directory.
const FileName = "SKILL.md"

const (
	maxNameLen          = 64
	maxDescriptionLen   = 1024
	maxCompatibilityLen = 500
)

var nameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Skill is a parsed skill directory.
type Skill struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`

	// Dir is the absolute path of the skill directory.
	Dir string `yaml:"-"`
	// Body is the Markdown content following the frontmatter.
	Body string `yaml:"-"`
}

// Parse reads and parses the SKILL.md file in dir.
func Parse(dir string) (*Skill, error) {
	path := filepath.Join(dir, FileName)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	front, body, err := splitFrontmatter(string(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	var s Skill
	if err := yaml.Unmarshal([]byte(front), &s); err != nil {
		return nil, fmt.Errorf("%s: invalid frontmatter: %w", FileName, err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	s.Dir = abs
	s.Body = body
	return &s, nil
}

func splitFrontmatter(s string) (front, body string, err error) {
	s = strings.TrimPrefix(s, "\uFEFF")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		return "", "", errors.New("missing YAML frontmatter (file must start with ---)")
	}
	lines := strings.SplitAfter(rest, "\n")
	for i, l := range lines {
		if strings.TrimRight(l, " \t\n") == "---" {
			return strings.Join(lines[:i], ""), strings.Join(lines[i+1:], ""), nil
		}
	}
	return "", "", errors.New("unterminated YAML frontmatter (missing closing ---)")
}

// Validate checks the skill against the Agent Skills specification.
func (s *Skill) Validate() error {
	var errs []error
	switch {
	case s.Name == "":
		errs = append(errs, errors.New("name is required"))
	case utf8.RuneCountInString(s.Name) > maxNameLen:
		errs = append(errs, fmt.Errorf("name must be at most %d characters", maxNameLen))
	case !nameRE.MatchString(s.Name):
		errs = append(errs, fmt.Errorf("name %q must be lowercase letters, digits, and single hyphens, not starting or ending with a hyphen", s.Name))
	}
	if s.Name != "" && filepath.Base(s.Dir) != s.Name {
		errs = append(errs, fmt.Errorf("name %q must match its directory name %q", s.Name, filepath.Base(s.Dir)))
	}
	switch n := utf8.RuneCountInString(strings.TrimSpace(s.Description)); {
	case n == 0:
		errs = append(errs, errors.New("description is required"))
	case n > maxDescriptionLen:
		errs = append(errs, fmt.Errorf("description must be at most %d characters (got %d)", maxDescriptionLen, n))
	}
	if n := utf8.RuneCountInString(s.Compatibility); n > maxCompatibilityLen {
		errs = append(errs, fmt.Errorf("compatibility must be at most %d characters (got %d)", maxCompatibilityLen, n))
	}
	return errors.Join(errs...)
}

// ParseError records a skill directory that could not be parsed.
type ParseError struct {
	Dir string
	Err error
}

func (e *ParseError) Error() string { return e.Err.Error() }
func (e *ParseError) Unwrap() error { return e.Err }

// Discover walks root and returns every directory containing a SKILL.md,
// sorted by name. Hidden directories, node_modules, and any directory in
// skip (absolute paths) are not descended into. Skills are not nested: the
// contents of a skill directory are not searched further.
//
// Symlinked directories are followed, and a skill is named after the path it
// was found at. A skill reachable through several paths is returned once, at
// its first path in lexical order. Dangling symlinks and symlink cycles are
// ignored.
func Discover(root string, skip ...string) ([]*Skill, []*ParseError, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	var skills []*Skill
	var perrs []*ParseError
	seen := map[string]bool{}
	err = fsutil.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			if path != root && (errors.Is(err, fs.ErrNotExist) || errors.Is(err, fsutil.ErrCycle)) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if path != root {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || slices.Contains(skip, path) {
				return fs.SkipDir
			}
		}
		if _, err := os.Stat(filepath.Join(path, FileName)); err != nil {
			return nil
		}
		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		if seen[real] {
			return fs.SkipDir
		}
		seen[real] = true
		s, err := Parse(path)
		if err != nil {
			perrs = append(perrs, &ParseError{Dir: path, Err: err})
		} else {
			skills = append(skills, s)
		}
		return fs.SkipDir
	})
	if err != nil {
		return nil, nil, err
	}
	slices.SortFunc(skills, func(a, b *Skill) int { return strings.Compare(a.Name, b.Name) })
	return skills, perrs, nil
}
