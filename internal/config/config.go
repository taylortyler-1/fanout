// Package config loads the optional fanout.yaml at the root of a skills repo.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/taylortyler-1/fanout/internal/plugin"
)

// FileName is the config file looked up in the skills repo root.
const FileName = "fanout.yaml"

// Config is the contents of fanout.yaml.
type Config struct {
	plugin.Metadata `yaml:",inline"`
	// Skills limits the build to these skill names. Empty means all.
	Skills []string `yaml:"skills"`
	// Platforms are the default build targets. Empty means claude-code.
	Platforms []string `yaml:"platforms"`
	// Exclude lists directories, relative to the repo, that are not searched
	// for skills. Nil means DefaultExclude.
	Exclude []string `yaml:"exclude"`
}

// DefaultExclude keeps fanout's default output directory out of discovery.
var DefaultExclude = []string{"dist"}

// ExcludeDirs returns the absolute paths of the excluded directories.
func (c Config) ExcludeDirs(repo string) []string {
	ex := c.Exclude
	if ex == nil {
		ex = DefaultExclude
	}
	dirs := make([]string, len(ex))
	for i, e := range ex {
		dirs[i] = filepath.Join(repo, filepath.FromSlash(e))
	}
	return dirs
}

// Load reads path. If path is empty, it reads FileName from repo if present
// and otherwise returns a zero Config.
func Load(repo, path string) (Config, error) {
	explicit := path != ""
	if !explicit {
		path = filepath.Join(repo, FileName)
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}
