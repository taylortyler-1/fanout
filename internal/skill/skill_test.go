package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, root, dir, content string) string {
	t.Helper()
	d := filepath.Join(root, dir)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func TestParse(t *testing.T) {
	root := t.TempDir()
	d := writeSkill(t, root, "pdf", "\uFEFF---\r\nname: pdf\r\ndescription: Work with PDFs.\r\nmetadata:\r\n  author: me\r\n---\r\n# Body\r\n")
	s, err := Parse(d)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "pdf" || s.Description != "Work with PDFs." || s.Metadata["author"] != "me" {
		t.Errorf("unexpected skill: %+v", s)
	}
	if s.Body != "# Body\n" {
		t.Errorf("body = %q", s.Body)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestParseErrors(t *testing.T) {
	for name, content := range map[string]string{
		"missing":      "# no frontmatter\n",
		"unterminated": "---\nname: x\n",
		"bad yaml":     "---\nname: [\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(writeSkill(t, t.TempDir(), "x", content)); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		dir, name, desc string
		want            string
	}{
		{"ok-skill", "ok-skill", "d", ""},
		{"x", "", "d", "name is required"},
		{"Bad", "Bad", "d", "lowercase"},
		{"a--b", "a--b", "d", "lowercase"},
		{"-a", "-a", "d", "lowercase"},
		{"dir", "other", "d", "must match its directory"},
		{"x", "x", "  ", "description is required"},
		{"x", "x", strings.Repeat("a", 1025), "at most 1024"},
		{strings.Repeat("a", 65), strings.Repeat("a", 65), "d", "at most 64"},
	}
	for _, tt := range tests {
		s := &Skill{Name: tt.name, Description: tt.desc, Dir: "/repo/" + tt.dir}
		err := s.Validate()
		if tt.want == "" {
			if err != nil {
				t.Errorf("%s: unexpected error %v", tt.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%q: error %v, want containing %q", tt.name, err, tt.want)
		}
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skills/b", "---\nname: b\ndescription: d\n---\n")
	writeSkill(t, root, "a", "---\nname: a\ndescription: d\n---\n")
	writeSkill(t, root, "a/nested", "---\nname: nested\ndescription: d\n---\n")
	writeSkill(t, root, ".hidden/c", "---\nname: c\ndescription: d\n---\n")
	writeSkill(t, root, "dist/b", "---\nname: b\ndescription: d\n---\n")
	writeSkill(t, root, "broken", "nope")

	skills, perrs, err := Discover(root, filepath.Join(root, "dist"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range skills {
		names = append(names, s.Name)
	}
	if got := strings.Join(names, ","); got != "a,b" {
		t.Errorf("skills = %s, want a,b", got)
	}
	if len(perrs) != 1 || filepath.Base(perrs[0].Dir) != "broken" {
		t.Errorf("parse errors = %v", perrs)
	}
}

func TestDiscoverSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeSkill(t, outside, "linked", "---\nname: linked\ndescription: d\n---\n")
	writeSkill(t, root, "skills/real", "---\nname: real\ndescription: d\n---\n")
	symlink(t, filepath.Join(outside, "linked"), filepath.Join(root, "skills", "linked"))
	symlink(t, filepath.Join(root, "skills", "real"), filepath.Join(root, "skills", "zz-alias"))
	symlink(t, filepath.Join(root, "nope"), filepath.Join(root, "skills", "dangling"))
	symlink(t, root, filepath.Join(root, "skills", "loop"))

	skills, perrs, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(perrs) != 0 {
		t.Errorf("parse errors = %v", perrs)
	}
	var names []string
	for _, s := range skills {
		names = append(names, s.Name)
	}
	if got := strings.Join(names, ","); got != "linked,real" {
		t.Errorf("skills = %s, want linked,real", got)
	}
}
