// Package buildidentity exposes the immutable identity embedded in the API
// binary and the timestamp of the executable that is currently running.
package buildidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SourceIdentity is populated by the scenario build driver. Keeping the
// default explicit makes test binaries and manually-built binaries honest.
var SourceIdentity = "unknown"

// RuntimeBuildTime returns the executable's filesystem timestamp.
func RuntimeBuildTime() string {
	executable, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	info, err := os.Stat(executable)
	if err != nil || info.ModTime().IsZero() {
		return "unknown"
	}
	return info.ModTime().UTC().Format(time.RFC3339Nano)
}

// SourceFingerprint hashes the API source inputs that determine the binary.
// Paths are sorted and included in the digest so identical bytes at different
// locations cannot produce the same identity accidentally.
func SourceFingerprint(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve source root: %w", err)
	}

	var paths []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if name == "audio-tools-api" || name == "audio-tools-api.exe" {
			return nil
		}
		if strings.HasSuffix(name, ".go") || name == "go.mod" || name == "go.sum" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk source root: %w", err)
	}
	sort.Strings(paths)

	hash := sha256.New()
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("read %s: %w", path, readErr)
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return "", fmt.Errorf("relativize %s: %w", path, relErr)
		}
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", filepath.ToSlash(relative), len(data))
		_, _ = hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
