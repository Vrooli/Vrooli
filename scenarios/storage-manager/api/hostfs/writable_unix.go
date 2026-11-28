//go:build !windows

package hostfs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// restoreOwnerWritableDirs gives the owner rwx on every directory in the tree
// at path that the current user owns but cannot modify, and reports how many
// it changed.
//
// Removing a file needs write and search permission on its parent directory,
// not on the file. Go's module cache marks every extracted directory 0555, and
// tests that chmod a t.TempDir fixture read-only leave the same shape behind,
// so a tree the user owns outright still fails with "unlinkat: permission
// denied". `go clean -modcache` handles this by making the tree writable
// first; this is the same repair, confined to the candidate being removed.
//
// The repair never widens what can be deleted. All resolution goes through an
// os.Root opened at path, so no symlink can lead it outside the candidate.
// Symlinks are never descended. A directory owned by another user, or on
// another device, is neither changed nor descended: the kernel would refuse
// the chmod for a non-root process anyway, and its contents could not be
// removed. Only directory modes change; file modes are irrelevant to unlink.
func restoreOwnerWritableDirs(ctx context.Context, path string) (int, error) {
	top, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !top.IsDir() || !ownedByCurrentUser(top) {
		return 0, nil
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return 0, err
	}
	defer root.Close()

	repaired := 0
	var firstErr error
	stack := []string{"."}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return repaired, err
		}
		rel := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		info, err := root.Lstat(rel)
		if err != nil {
			if firstErr == nil && !errors.Is(err, fs.ErrNotExist) {
				firstErr = err
			}
			continue
		}
		if !info.IsDir() || !ownedByCurrentUser(info) || !sameDevice(top, info) {
			continue
		}
		if info.Mode().Perm()&0o700 != 0o700 {
			if err := root.Chmod(rel, info.Mode().Perm()|0o700); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			repaired++
		}
		dir, err := root.Open(rel)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		entries, err := dir.ReadDir(-1)
		_ = dir.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		for _, entry := range entries {
			// The directory-entry type comes from the listing itself, so a
			// symlink is never mistaken for a directory here.
			if entry.IsDir() {
				stack = append(stack, filepath.Join(rel, entry.Name()))
			}
		}
	}
	return repaired, firstErr
}
