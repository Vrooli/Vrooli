package buildinfo

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type fileEntry struct {
	rel     string
	size    int64
	modTime int64
}

var skipDirs = []string{
	".git",
	".vscode",
	".idea",
}

var skipFiles = []string{
	"scenario-completeness-scoring",
	"build.meta",
}

// ComputeFingerprint walks the provided root directory and returns a deterministic
// fingerprint derived from each file's relative path, size, and modification time.
func ComputeFingerprint(root string) (string, error) {
	var entries []fileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		if rel == "." {
			return nil
		}

		if skipDir(rel) && d.IsDir() {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if skipFile(rel) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}

		entries = append(entries, fileEntry{
			rel:     filepath.ToSlash(rel),
			size:    info.Size(),
			modTime: info.ModTime().UnixNano(),
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
		fmt.Fprintf(hasher, "%s|%d|%d\n", entry.rel, entry.size, entry.modTime)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func skipDir(path string) bool {
	for _, skip := range skipDirs {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

func skipFile(path string) bool {
	for _, skip := range skipFiles {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}
