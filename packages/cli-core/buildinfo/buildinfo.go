package buildinfo

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileEntry struct {
	rel  string
	size int64
	hash [32]byte
}

var skipDirs = []string{
	".git",
	".vscode",
	".idea",
	"coverage",
	"dist",
	"build",
	"tmp",
	"data",
	"node_modules",
}

var skipFiles = []string{
	"build.meta",
}

// ComputeFingerprint walks the provided root directory and returns a deterministic
// fingerprint derived from each file's relative path, size, and contents.
func ComputeFingerprint(root string, extraSkipFiles ...string) (string, error) {
	var entries []fileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		if rel == "." {
			return nil
		}

		if skipDir(rel) && d.IsDir() {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if skipFile(rel, extraSkipFiles) {
			return nil
		}

		content, readErr := fs.ReadFile(fs.FS(os.DirFS(root)), rel)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rel, readErr)
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}

		entries = append(entries, fileEntry{
			rel:  filepath.ToSlash(rel),
			size: info.Size(),
			hash: sha256.Sum256(content),
		})

		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	hasher := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func skipDir(path string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range skipDirs {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

func skipFile(path string, extra []string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range skipFiles {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	for _, skip := range extra {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}
