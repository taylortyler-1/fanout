// Package platform defines plugin targets. Each platform is a directory
// containing a platform.yaml descriptor and a files/ tree of Go text
// templates (*.tmpl) and static files that make up the plugin scaffold.
package platform

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed all:platforms
var builtinFS embed.FS

const (
	descriptorFile = "platform.yaml"
	filesDir       = "files"
)

// Platform describes how to package skills for one agent platform.
type Platform struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// SkillsDir is where skill directories are copied, relative to the
	// plugin root.
	SkillsDir string `yaml:"skillsDir"`

	// Files is the scaffold tree rendered into the plugin root.
	Files fs.FS `yaml:"-"`
}

// Load reads a platform from fsys, which is rooted at the platform directory.
func Load(fsys fs.FS) (*Platform, error) {
	b, err := fs.ReadFile(fsys, descriptorFile)
	if err != nil {
		return nil, err
	}
	var p Platform
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%s: %w", descriptorFile, err)
	}
	if !regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`).MatchString(p.Name) {
		return nil, fmt.Errorf("%s: name must be kebab-case (lowercase letters, digits, hyphens)", descriptorFile)
	}
	if p.SkillsDir == "" {
		p.SkillsDir = "skills"
	}
	if !fs.ValidPath(p.SkillsDir) || strings.Contains(p.SkillsDir, `\`) || strings.Contains(p.SkillsDir, ":") {
		return nil, fmt.Errorf("%s: skillsDir %q must be a relative path inside the plugin", descriptorFile, p.SkillsDir)
	}
	files, err := fs.Sub(fsys, filesDir)
	if err != nil {
		return nil, err
	}
	p.Files = files
	return &p, nil
}

// LoadDir reads a platform definition from a directory on disk.
func LoadDir(dir string) (*Platform, error) {
	p, err := Load(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("platform %s: %w", dir, err)
	}
	return p, nil
}

// Builtin returns the platforms embedded in the binary, sorted by name.
func Builtin() ([]*Platform, error) {
	entries, err := fs.ReadDir(builtinFS, "platforms")
	if err != nil {
		return nil, err
	}
	var ps []*Platform
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, err := fs.Sub(builtinFS, "platforms/"+e.Name())
		if err != nil {
			return nil, err
		}
		p, err := Load(sub)
		if err != nil {
			return nil, fmt.Errorf("builtin platform %s: %w", e.Name(), err)
		}
		ps = append(ps, p)
	}
	slices.SortFunc(ps, func(a, b *Platform) int { return strings.Compare(a.Name, b.Name) })
	return ps, nil
}

// Lookup returns the builtin platform with the given name.
func Lookup(name string) (*Platform, error) {
	ps, err := Builtin()
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		if p.Name == name {
			return p, nil
		}
	}
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Name
	}
	return nil, errors.New("unknown platform " + name + " (available: " + strings.Join(names, ", ") + ")")
}
