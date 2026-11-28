package components

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureComponentsDirectoryCreatesUnderRootedParent(t *testing.T) {
	parent := t.TempDir()
	sourceRoot := filepath.Join(parent, "library")
	if err := ensureComponentsDirectory(sourceRoot); err != nil {
		t.Fatalf("ensure components directory: %v", err)
	}
	info, err := os.Stat(filepath.Join(sourceRoot, "components"))
	if err != nil {
		t.Fatalf("stat components directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("components path is not a directory: %s", info.Mode())
	}
}

func TestEnsureComponentsDirectoryRejectsSourceRootSymlinkEscape(t *testing.T) {
	parent := t.TempDir()
	outside := t.TempDir()
	sourceRoot := filepath.Join(parent, "library")
	if err := os.Symlink(outside, sourceRoot); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := ensureComponentsDirectory(sourceRoot); err == nil {
		t.Fatal("created components through a source-root symlink outside the parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "components")); !os.IsNotExist(err) {
		t.Fatalf("outside components directory unexpectedly exists: %v", err)
	}
}
