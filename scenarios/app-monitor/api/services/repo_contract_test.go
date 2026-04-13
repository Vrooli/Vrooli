package services

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindRepoRootUsesRepoContract(t *testing.T) {
	root := repoRootForAppMonitorTest(t)
	t.Setenv("VROOLI_ROOT", root)

	got, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("findRepoRoot() = %q, want %q", got, root)
	}
}

func repoRootForAppMonitorTest(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}
