// Package plugin renders a platform scaffold and copies skills into it.
package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/taylortyler-1/fanout/internal/fsutil"
	"github.com/taylortyler-1/fanout/internal/platform"
	"github.com/taylortyler-1/fanout/internal/skill"
)

const templateExt = ".tmpl"

var nameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Author identifies the plugin author.
type Author struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
	URL   string `yaml:"url"`
}

// Metadata describes the generated plugin. It is shared by every platform.
type Metadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Author      Author   `yaml:"author"`
	Homepage    string   `yaml:"homepage"`
	Repository  string   `yaml:"repository"`
	License     string   `yaml:"license"`
	Keywords    []string `yaml:"keywords"`
}

// Validate checks the fields every platform depends on.
func (m Metadata) Validate() error {
	if !nameRE.MatchString(m.Name) {
		return fmt.Errorf("plugin name %q must be kebab-case (lowercase letters, digits, hyphens)", m.Name)
	}
	if m.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	return nil
}

// Data is the value passed to every template.
type Data struct {
	Plugin    Metadata
	Platform  *platform.Platform
	Skills    []*skill.Skill
	SkillsDir string
}

var funcs = template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
	"join": strings.Join,
}

// Build writes the plugin for p into outDir, which must not exist or be empty.
func Build(p *platform.Platform, meta Metadata, skills []*skill.Skill, outDir string) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	data := Data{Plugin: meta, Platform: p, Skills: skills, SkillsDir: p.SkillsDir}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	err := fs.WalkDir(p.Files, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(outDir, filepath.FromSlash(name))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if !strings.HasSuffix(name, templateExt) {
			return copyFSFile(p.Files, name, dst)
		}
		dst = strings.TrimSuffix(dst, templateExt)
		out, err := render(p.Files, name, data)
		if err != nil {
			return err
		}
		if strings.HasSuffix(dst, ".json") && !json.Valid(out) {
			return fmt.Errorf("template %s rendered invalid JSON:\n%s", name, out)
		}
		return os.WriteFile(dst, out, 0o644)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", p.Name, err)
	}
	for _, s := range skills {
		dst := filepath.Join(outDir, filepath.FromSlash(p.SkillsDir), s.Name)
		if err := copyDir(s.Dir, dst); err != nil {
			return fmt.Errorf("copy skill %s: %w", s.Name, err)
		}
	}
	return nil
}

func render(fsys fs.FS, name string, data Data) ([]byte, error) {
	src, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	t, err := template.New(path.Base(name)).Funcs(funcs).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func copyFSFile(fsys fs.FS, name, dst string) error {
	src, err := fsys.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	return writeFile(dst, src, 0o644)
}

// copyDir copies the files under src to dst, preserving permissions.
// Symlinks are followed and their targets copied as ordinary files and
// directories, so the plugin is self-contained. VCS metadata and OS junk
// files are skipped, as are special files such as sockets.
func copyDir(src, dst string) error {
	return fsutil.Walk(src, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%s: broken symlink", p)
			}
			return err
		}
		if info.Name() == ".git" || info.Name() == ".DS_Store" {
			if info.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		return writeFile(target, f, info.Mode().Perm())
	})
}

func writeFile(dst string, r io.Reader, perm fs.FileMode) error {
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		return errors.Join(err, f.Close())
	}
	return f.Close()
}
