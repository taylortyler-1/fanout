package plugin

import (
	"bytes"
	"encoding/json"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/taylortyler-1/fanout/internal/platform"
	"github.com/taylortyler-1/fanout/internal/skill"
)

func testSkill(t *testing.T) *skill.Skill {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "my-skill")
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, skill.FileName), []byte("---\nname: my-skill\ndescription: Says \"hi\".\n---\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "scripts", "run.sh"), []byte("#!/bin/sh\n"), 0o755)
	os.WriteFile(filepath.Join(dir, ".DS_Store"), nil, 0o644)
	s, err := skill.Parse(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBuildBuiltinPlatforms(t *testing.T) {
	s := testSkill(t)
	ps, err := platform.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	metas := map[string]Metadata{
		"minimal": {Name: "demo", Version: "1.0.0"},
		"full": {
			Name: "demo", Version: "1.0.0", Description: `A "quoted" plugin`,
			Author:   Author{Name: "Ann", Email: "a@example.com", URL: "https://example.com"},
			Homepage: "https://example.com", Repository: "https://github.com/x/y",
			License: "MIT", Keywords: []string{"a", "b"},
		},
	}
	for _, p := range ps {
		for mname, meta := range metas {
			t.Run(p.Name+"/"+mname, func(t *testing.T) {
				out := t.TempDir()
				if err := Build(p, meta, []*skill.Skill{s}, out); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(filepath.Join(out, p.SkillsDir, "my-skill", "scripts", "run.sh"))
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm()&0o100 == 0 {
					t.Error("script lost its executable bit")
				}
				if _, err := os.Stat(filepath.Join(out, p.SkillsDir, "my-skill", ".DS_Store")); err == nil {
					t.Error(".DS_Store was copied")
				}
			})
		}
	}
}

func TestClaudeCodeManifest(t *testing.T) {
	p, err := platform.Lookup("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	meta := Metadata{Name: "demo", Version: "2.0.0", Description: "d", Author: Author{Name: "Ann"}, Keywords: []string{"k"}}
	if err := Build(p, meta, []*skill.Skill{testSkill(t)}, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "demo" || got["version"] != "2.0.0" || got["author"].(map[string]any)["name"] != "Ann" {
		t.Errorf("plugin.json = %s", b)
	}
	if _, ok := got["homepage"]; ok {
		t.Error("empty homepage should be omitted")
	}
	b, err = os.ReadFile(filepath.Join(out, ".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatal(err)
	}
	var marketplace struct {
		Name, Description string
		Plugins           []struct{ Name, Source string }
	}
	if err := json.Unmarshal(b, &marketplace); err != nil {
		t.Fatal(err)
	}
	if marketplace.Name != meta.Name || marketplace.Description != meta.Description ||
		len(marketplace.Plugins) != 1 || marketplace.Plugins[0].Source != "./" ||
		marketplace.Plugins[0].Name != meta.Name {
		t.Errorf("marketplace.json = %s", b)
	}
}

func TestClaudeCodeIcon(t *testing.T) {
	p, err := platform.Lookup("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := Build(p, Metadata{Name: "demo", Version: "1.0.0"}, nil, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, ".claude-plugin", "icon.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) >= 2*1024*1024 {
		t.Fatal("Claude listing icon must be under 2 MB")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != cfg.Height || cfg.Width < 512 || cfg.Width > 2048 {
		t.Fatalf("Claude listing icon must be square, 512–2048 px: %dx%d", cfg.Width, cfg.Height)
	}
	original, err := fs.ReadFile(p.Files, ".claude-plugin/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, original) {
		t.Fatal("icon changed during scaffold copying")
	}
}

func TestPiManifest(t *testing.T) {
	p, err := platform.Lookup("pi")
	if err != nil {
		t.Fatal(err)
	}
	for _, full := range []bool{false, true} {
		meta := Metadata{Name: "demo", Version: "2.0.0"}
		if full {
			meta.Description = `A "quoted" package`
			meta.Author = Author{Name: "Ann", Email: "a@example.com", URL: "https://example.com"}
			meta.Homepage = "https://example.com"
			meta.Repository = "https://github.com/x/y"
			meta.License = "MIT"
			meta.Keywords = []string{"docs", "pi-package"}
		}
		out := t.TempDir()
		if err := Build(p, meta, []*skill.Skill{testSkill(t)}, out); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(out, "package.json"))
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Name, Version, Description, License, Homepage, Repository string
			Author                                                    Author
			Keywords, Files                                           []string
			Pi                                                        struct{ Skills []string }
		}
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if got.Name != meta.Name || got.Version != meta.Version || got.Description != meta.Description ||
			got.License != meta.License || got.Author != meta.Author ||
			got.Homepage != meta.Homepage || got.Repository != meta.Repository {
			t.Errorf("metadata mismatch: %s", b)
		}
		if len(got.Pi.Skills) != 1 || got.Pi.Skills[0] != "./skills" {
			t.Errorf("invalid Pi skills manifest: %s", b)
		}
		if len(got.Files) != 2 || got.Files[0] != "skills" || got.Files[1] != "README.md" {
			t.Errorf("invalid npm files allowlist: %s", b)
		}
		wantKeywords := 1
		if full {
			wantKeywords = 2
		}
		if len(got.Keywords) != wantKeywords || got.Keywords[0] != "pi-package" {
			t.Errorf("missing or duplicate pi-package keyword: %s", b)
		}
	}
}

func TestCopyDirFollowsSymlinks(t *testing.T) {
	shared := t.TempDir()
	os.MkdirAll(filepath.Join(shared, "refs"), 0o755)
	os.WriteFile(filepath.Join(shared, "refs", "guide.md"), []byte("guide"), 0o644)
	os.WriteFile(filepath.Join(shared, "common.sh"), []byte("#!/bin/sh\n"), 0o755)

	src := filepath.Join(t.TempDir(), "s")
	os.MkdirAll(src, 0o755)
	os.Symlink(filepath.Join(shared, "refs"), filepath.Join(src, "references"))
	os.Symlink(filepath.Join(shared, "common.sh"), filepath.Join(src, "common.sh"))

	dst := filepath.Join(t.TempDir(), "out")
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"references", "references/guide.md", "common.sh"} {
		info, err := os.Lstat(filepath.Join(dst, rel))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s was copied as a symlink", rel)
		}
	}
	if info, _ := os.Stat(filepath.Join(dst, "common.sh")); info.Mode().Perm()&0o100 == 0 {
		t.Error("linked script lost its executable bit")
	}

	os.Symlink(filepath.Join(shared, "missing"), filepath.Join(src, "broken"))
	if err := copyDir(src, filepath.Join(t.TempDir(), "out")); err == nil {
		t.Error("expected error for broken symlink")
	}
}

func TestBuildRejectsInvalidJSON(t *testing.T) {
	p := &platform.Platform{
		Name:      "broken",
		SkillsDir: "skills",
		Files:     fstest.MapFS{"x.json.tmpl": {Data: []byte(`{"name": {{ .Plugin.Name }}}`)}},
	}
	err := Build(p, Metadata{Name: "demo", Version: "1"}, nil, t.TempDir())
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestMetadataValidate(t *testing.T) {
	if err := (Metadata{Name: "Not Kebab", Version: "1"}).Validate(); err == nil {
		t.Error("expected name error")
	}
	if err := (Metadata{Name: "ok"}).Validate(); err == nil {
		t.Error("expected version error")
	}
}
