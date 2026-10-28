package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taylortyler-1/fanout/internal/skill"
)

func put(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func execute(args ...string) (string, error) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestCLIFlow(t *testing.T) {
	repo := t.TempDir()
	put(t, filepath.Join(repo, "hello", "SKILL.md"), "---\nname: hello\ndescription: Hello world\n---\nInstructions.\n")
	put(t, filepath.Join(repo, "fanout.yaml"), "name: demo\nversion: 1.2.3\n")
	if _, err := execute("validate", repo); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "plugins")
	args := []string{"build", repo, "-p", "all", "-o", out}
	if _, err := execute(args...); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"claude-code", "pi"} {
		if _, err := os.Stat(filepath.Join(out, p, "skills", "hello", "SKILL.md")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := execute(args...); err == nil {
		t.Fatal("expected refusal to overwrite")
	}
	if _, err := execute(append(args, "--force")...); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("build", repo, "-o", out, "--skill", "missing"); err == nil {
		t.Fatal("expected missing skill error")
	}
	if out, err := execute("--version"); err != nil || !strings.Contains(out, version) {
		t.Fatalf("version: %q, %v", out, err)
	}
}

func TestCheckOutputProtectsSources(t *testing.T) {
	repo := t.TempDir()
	skillDir := filepath.Join(repo, "hello")
	put(t, filepath.Join(skillDir, "SKILL.md"), "sentinel")
	skills := []*skill.Skill{{Name: "hello", Dir: skillDir}}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(repo, alias); err != nil {
		t.Fatal(err)
	}
	for _, dst := range []string{
		repo,
		filepath.Dir(repo),
		skillDir,
		filepath.Join(skillDir, "dist", "claude-code"),
		alias,
		filepath.Join(alias, "hello", "dist", "claude-code"),
	} {
		t.Run(dst, func(t *testing.T) {
			if err := checkOutput(dst, repo, skills, true); err == nil {
				t.Fatalf("allowed unsafe output %s", dst)
			}
		})
	}
	if err := checkOutput(filepath.Join(repo, "dist", "claude-code"), repo, skills, true); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md")); err != nil || string(b) != "sentinel" {
		t.Fatalf("source modified: %q, %v", b, err)
	}
}

func TestFailedBuildPreservesOutput(t *testing.T) {
	repo := t.TempDir()
	put(t, filepath.Join(repo, "hello", "SKILL.md"), "---\nname: hello\ndescription: Hello\n---\n")
	custom := t.TempDir()
	put(t, filepath.Join(custom, "platform.yaml"), "name: custom\n")
	put(t, filepath.Join(custom, "files", "manifest.json.tmpl"), "not json")
	out := t.TempDir()
	sentinel := filepath.Join(out, "custom", "existing.txt")
	put(t, sentinel, "keep me")
	if _, err := execute("build", repo, "--platform-dir", custom, "-o", out, "--force"); err == nil {
		t.Fatal("expected rendering failure")
	}
	if b, err := os.ReadFile(sentinel); err != nil || string(b) != "keep me" {
		t.Fatalf("previous output lost: %q, %v", b, err)
	}
	stages, err := filepath.Glob(filepath.Join(out, ".fanout-*"))
	if err != nil || len(stages) != 0 {
		t.Fatalf("staging directories left behind: %v, %v", stages, err)
	}
}
