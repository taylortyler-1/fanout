package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkFollowsSymlinks(t *testing.T) {
	root := t.TempDir()
	shared := filepath.Join(root, "shared")
	os.MkdirAll(filepath.Join(shared, "sub"), 0o755)
	os.WriteFile(filepath.Join(shared, "sub", "f.txt"), nil, 0o644)
	tree := filepath.Join(root, "tree")
	os.MkdirAll(tree, 0o755)
	os.Symlink(shared, filepath.Join(tree, "linkdir"))
	os.Symlink(filepath.Join(shared, "sub", "f.txt"), filepath.Join(tree, "linkfile"))
	os.Symlink(filepath.Join(root, "missing"), filepath.Join(tree, "dangling"))
	os.Symlink(tree, filepath.Join(tree, "loop"))

	var visited []string
	var errs []string
	err := Walk(tree, func(path string, info fs.FileInfo, err error) error {
		rel, _ := filepath.Rel(tree, path)
		if err != nil {
			errs = append(errs, rel)
			switch {
			case errors.Is(err, fs.ErrNotExist) && rel == "dangling":
			case errors.Is(err, ErrCycle) && rel == "loop":
			default:
				t.Errorf("%s: unexpected error %v", rel, err)
			}
			return nil
		}
		visited = append(visited, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := ". linkdir linkdir/sub linkdir/sub/f.txt linkfile"
	if got := strings.Join(visited, " "); got != want {
		t.Errorf("visited %q, want %q", got, want)
	}
	if got := strings.Join(errs, " "); got != "dangling loop" {
		t.Errorf("errors at %q, want dangling and loop", got)
	}
}
