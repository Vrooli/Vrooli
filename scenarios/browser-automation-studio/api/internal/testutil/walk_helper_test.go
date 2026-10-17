package testutil_test

import (
	"os"
	"testing"
)

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
	for _, entry := range entries {
		out = append(out, dirEntry{name: entry.Name(), isDir: entry.IsDir()})
	}
	return out
}
