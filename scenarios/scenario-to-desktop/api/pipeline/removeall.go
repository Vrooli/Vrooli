package pipeline

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// removeAllRobust is a defensive os.RemoveAll wrapper.
//
// We've observed rare "directory not empty" failures from os.RemoveAll during
// concurrent filesystem activity. This helper retries and, as a last resort,
// performs a bottom-up delete pass before retrying.
func removeAllRobust(path string) error {
	p := strings.TrimSpace(path)
	if p == "" || p == "." || p == string(filepath.Separator) {
		return errors.New("refusing to remove empty or root path")
	}

	// Fast path: the common case.
	if err := os.RemoveAll(p); err == nil || os.IsNotExist(err) {
		return nil
	} else if !isDirNotEmpty(err) {
		return err
	}

	// Retry a few times for transient ENOTEMPTY races.
	for i := 0; i < 3; i++ {
		time.Sleep(time.Duration(75*(i+1)) * time.Millisecond)
		if err := os.RemoveAll(p); err == nil || os.IsNotExist(err) {
			return nil
		} else if !isDirNotEmpty(err) {
			return err
		}
	}

	// Last resort: do an explicit bottom-up removal pass, then retry.
	_ = forceRemoveTree(p)
	if err := os.RemoveAll(p); err == nil || os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(p)
}

func isDirNotEmpty(err error) bool {
	if err == nil {
		return false
	}
	// Best-effort: not all platforms wrap ENOTEMPTY consistently through os.PathError.
	if errors.Is(err, syscall.ENOTEMPTY) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "directory not empty") || strings.Contains(msg, "not empty")
}

func forceRemoveTree(root string) error {
	type entry struct {
		path  string
		isDir bool
	}

	var entries []entry
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// WalkDir may fail mid-tree when things change; keep going where possible.
			return nil
		}
		entries = append(entries, entry{path: path, isDir: d.IsDir()})
		return nil
	})
	if walkErr != nil {
		// We'll still attempt removal below even if walk had issues.
	}

	// Remove children first, then directories.
	for i := len(entries) - 1; i >= 0; i-- {
		p := entries[i].path
		if p == root {
			continue
		}
		if entries[i].isDir {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				// Ignore transient ENOTEMPTY; caller will retry RemoveAll.
				_ = err
			}
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			_ = err
		}
	}
	return nil
}
