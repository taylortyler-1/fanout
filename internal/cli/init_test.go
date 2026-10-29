package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taylortyler-1/fanout/internal/config"
)

func TestDefaultPluginVersion(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "skills")
	if _, err := execute("init", repo); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != defaultVersion {
		t.Fatalf("init version = %q, want %q", cfg.Version, defaultVersion)
	}
	if err := os.Remove(filepath.Join(repo, config.FileName)); err != nil {
		t.Fatal(err)
	}
	out, err := execute("build", repo, "--out", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "skills@"+defaultVersion+" with") {
		t.Fatalf("build did not use default plugin version: %s", out)
	}
}

func TestInitQuotesGitAuthor(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for _, name := range []string{"[Alice]", "{name: Alice}", "*Alice", "Alice: Smith", "Alice # Smith", "Alice O'Connor", "Alice \"Smith\"", "Alice\nSmith", "Alice\tSmith", "true"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GIT_CONFIG_COUNT", "1")
			t.Setenv("GIT_CONFIG_KEY_0", "user.name")
			t.Setenv("GIT_CONFIG_VALUE_0", name)
			repo := filepath.Join(t.TempDir(), "skills")
			if _, err := execute("init", repo); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(repo, "")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Author.Name != name {
				t.Fatalf("author = %q, want %q", cfg.Author.Name, name)
			}
			if _, err := execute("validate", repo); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestShellQuoteRoundTrip(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell is not installed")
	}
	for _, arg := range []string{"", "plain", "my skills", "skills; false", "skills&", "(skills)", "a\tb", "a\nb", "a'b", "a\"b", "$HOME", "`false`", "a\\b", "*?[abc]", "#comment", "~"} {
		t.Run(arg, func(t *testing.T) {
			out, err := exec.Command(sh, "-c", "printf '%s' "+shellQuote(arg)).Output()
			if err != nil || string(out) != arg {
				t.Fatalf("round trip %q: got %q, %v", arg, out, err)
			}
		})
	}
}

func TestInitRefusesDanglingSymlink(t *testing.T) {
	for _, rel := range []string{config.FileName, filepath.Join(exampleSkill, "SKILL.md")} {
		t.Run(rel, func(t *testing.T) {
			repo := t.TempDir()
			path := filepath.Join(repo, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "missing")
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			if out, err := execute("init", repo); err == nil || !strings.Contains(out, "already exists") {
				t.Fatalf("expected refusal: %q, %v", out, err)
			}
			if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("symlink target was created: %v", err)
			}
			other := config.FileName
			if rel == other {
				other = filepath.Join(exampleSkill, "SKILL.md")
			}
			if _, err := os.Lstat(filepath.Join(repo, other)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("init wrote files after refusal: %v", err)
			}
		})
	}
}

func TestCreateFileExclusive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := createFileExclusive(path, "original"); err != nil {
		t.Fatal(err)
	}
	if err := createFileExclusive(path, "replacement"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected existing-file error: %v", err)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "original" {
		t.Fatalf("existing file changed: %q, %v", b, err)
	}
	for _, target := range []string{path, filepath.Join(dir, "missing")} {
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if err := createFileExclusive(link, "replacement"); !errors.Is(err, os.ErrExist) {
			t.Fatalf("expected existing-symlink error: %v", err)
		}
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "original" {
		t.Fatalf("symlink target changed: %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dangling symlink target created: %v", err)
	}
}
