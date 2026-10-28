// Package fsutil provides filesystem helpers shared by discovery and copying.
package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrCycle reports a symlink that points at one of its own ancestors.
var ErrCycle = errors.New("symlink cycle")

// WalkFunc is called for each file and directory visited by Walk. path is
// the logical path (through any symlinks) and info describes the link target.
// If a path cannot be resolved (a dangling symlink or a cycle), info is nil
// and err is non-nil; returning nil skips that path. Returning fs.SkipDir
// for a directory skips its contents.
type WalkFunc func(path string, info fs.FileInfo, err error) error

// Walk walks the tree rooted at root in lexical order, like filepath.Walk,
// but follows symbolic links to files and directories.
func Walk(root string, fn WalkFunc) error {
	info, err := os.Stat(root)
	if err != nil {
		return fn(root, nil, err)
	}
	return walk(root, info, fn, map[string]bool{})
}

func walk(path string, info fs.FileInfo, fn WalkFunc, ancestors map[string]bool) error {
	var real string
	if info.IsDir() {
		var err error
		if real, err = filepath.EvalSymlinks(path); err != nil {
			return fn(path, nil, err)
		}
		if ancestors[real] {
			return fn(path, nil, fmt.Errorf("%s: %w", path, ErrCycle))
		}
	}
	if err := fn(path, info, nil); err != nil {
		if errors.Is(err, fs.SkipDir) && info.IsDir() {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	ancestors[real] = true
	defer delete(ancestors, real)
	entries, err := os.ReadDir(path)
	if err != nil {
		return fn(path, nil, err)
	}
	for _, e := range entries {
		p := filepath.Join(path, e.Name())
		fi, err := os.Stat(p)
		if err != nil {
			if err := fn(p, nil, err); err != nil {
				return err
			}
			continue
		}
		if err := walk(p, fi, fn, ancestors); err != nil {
			return err
		}
	}
	return nil
}
