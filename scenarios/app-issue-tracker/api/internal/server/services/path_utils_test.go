package services

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveScenarioPathAllowsWhitelistedDirectories(t *testing.T) {
	root := t.TempDir()
	relative := filepath.Join("tmp", "agent", "output.log")

	resolved, err := ResolveScenarioPath(root, relative, "tmp")
	if err != nil {
		t.Fatalf("expected success resolving path: %v", err)
	}

	if !filepath.IsAbs(resolved) {
		t.Fatalf("expected absolute path, got %s", resolved)
	}
	if got := filepath.Clean(resolved); got != filepath.Join(root, relative) {
		t.Fatalf("unexpected resolved path: %s", got)
	}
}

func TestResolveScenarioPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	var up string
	if runtime.GOOS == "windows" {
		up = "..\\secret.txt"
	} else {
		up = "../secret.txt"
	}

	if _, err := ResolveScenarioPath(root, up); err == nil {
		t.Fatalf("expected traversal to be rejected")
	}
}
