package buildinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFingerprint_DetectsChanges(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	writeFile(t, root, "main.go", []byte("package main\n"))
	writeFile(t, root, "format.go", []byte("package main\n"))

	original, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	writeFile(t, root, "format.go", []byte("package main\n// updated\n"))
	updated, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint after update: %v", err)
	}

	if original == updated {
		t.Fatalf("expected fingerprint to change after modifying files")
	}

	// Ensure deterministic ordering by rerunning without modifications
	secondRun, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint second run: %v", err)
	}
	if updated != secondRun {
		t.Fatalf("expected fingerprint to be deterministic; got %s and %s", updated, secondRun)
	}
}

func writeFile(t *testing.T, root, name string, contents []byte) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
