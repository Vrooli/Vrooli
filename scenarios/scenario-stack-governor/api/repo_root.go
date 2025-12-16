package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func FindRepoRoot(start string) (string, error) {
	dir := filepath.Clean(start)
	for i := 0; i < 25; i++ {
		// Repo root marker: Vrooli structure (no committed go.work required).
		if dirExists(filepath.Join(dir, ".vrooli")) && dirExists(filepath.Join(dir, "scenarios")) && dirExists(filepath.Join(dir, "resources")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found from %q", start)
}

