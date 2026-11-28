package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoRootFindsVrooliRoot(t *testing.T) {
	root := RepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "VISION.md")); err != nil {
		t.Fatalf("RepoRoot() = %q without VISION.md: %v", root, err)
	}
}

func TestWriteFileCreatesPrivateFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "fixture.txt")
	WriteFile(t, path, "expected")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "expected" {
		t.Fatalf("fixture = %q, want %q", got, "expected")
	}
}
