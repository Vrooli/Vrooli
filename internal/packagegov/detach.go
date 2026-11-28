package packagegov

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DetachShippedFiles gives every shipped package file that shares its inode
// with another path a private inode of its own, before a lifecycle command can
// rewrite it.
//
// A `file:` dependency install hard-links the package's shipped files (its
// package.json `files`) into each consumer's node_modules. Build tools such as
// tsc write outputs in place, so without this step a rebuild rewrites every
// consumer's installed copy at once, while files the rebuild adds reach only
// consumers whose setup reinstalls: a consumer sees a new index.js importing a
// module it does not have. Detaching first keeps each consumer on a complete,
// consistent old snapshot; the installed-copy completeness check then reports
// it stale and setup reinstalls it.
//
// Each replacement is a same-directory copy renamed over the original, so the
// package path is never missing or partial. Files with a single link are left
// alone, which makes repeated calls cheap.
func DetachShippedFiles(packageRoot string) error {
	roots, err := shippedRoots(packageRoot)
	if err != nil || len(roots) == 0 {
		return err
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, fs.ErrNotExist) {
					return nil
				}
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if linkCount(info) <= 1 {
				return nil
			}
			return detachFile(path, info.Mode().Perm())
		})
		if err != nil {
			return fmt.Errorf("detach shipped files under %s: %w", root, err)
		}
	}
	return nil
}

// shippedRoots resolves package.json `files` entries to existing paths under
// packageRoot. A glob entry such as "dist/**" contributes its literal prefix.
// A package without a `files` list ships nothing through this path.
func shippedRoots(packageRoot string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(packageRoot, "package.json"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var manifest struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("decode %s/package.json: %w", packageRoot, err)
	}
	seen := map[string]bool{}
	var roots []string
	for _, entry := range manifest.Files {
		entry = strings.TrimSpace(entry)
		if index := strings.IndexAny(entry, "*?[{"); index >= 0 {
			entry = entry[:index]
		}
		entry = strings.Trim(filepath.ToSlash(entry), "/")
		if entry == "" || entry == "." || strings.HasPrefix(entry, "..") || strings.HasPrefix(entry, "!") {
			continue
		}
		path := filepath.Join(packageRoot, filepath.FromSlash(entry))
		if seen[path] {
			continue
		}
		seen[path] = true
		roots = append(roots, path)
	}
	return roots, nil
}

func detachFile(path string, perm fs.FileMode) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".detach-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if _, err := io.Copy(temp, source); err != nil {
		temp.Close()
		os.Remove(tempPath)
		return err
	}
	if err := temp.Chmod(perm); err != nil {
		temp.Close()
		os.Remove(tempPath)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}
