package testutil_test

import (
	"os"
	"testing"
)

// dirEntry is a thin os.DirEntry alternative that walk() consumes.
type dirEntry struct {
	name  string
	isDir bool
}

func readDir(t *testing.T, path string) []dirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatalf("readDir %s: %v", path, err)
	}
	out := make([]dirEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, dirEntry{name: e.Name(), isDir: e.IsDir()})
	}
	return out
}
