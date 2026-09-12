package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMissingAncestorsComputesCreatedSet(t *testing.T) {
	base := t.TempDir()
	// base exists; base/a/b/c do not.
	target := filepath.Join(base, "a", "b", "c")

	got := missingAncestors(target)
	want := []string{
		filepath.Join(base, "a", "b", "c"),
		filepath.Join(base, "a", "b"),
		filepath.Join(base, "a"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("missingAncestors = %v, want %v", got, want)
	}

	// Once b exists, only c is "to be created".
	if err := os.MkdirAll(filepath.Join(base, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	got = missingAncestors(target)
	want = []string{filepath.Join(base, "a", "b", "c")}
	if !slices.Equal(got, want) {
		t.Fatalf("missingAncestors after partial create = %v, want %v", got, want)
	}

	// Fully existing path: nothing to create.
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := missingAncestors(target); len(got) != 0 {
		t.Fatalf("missingAncestors for existing path = %v, want empty", got)
	}
}

func TestEnsureOwnedDirCreatesWithoutRoot(t *testing.T) {
	// Not root-via-sudo in the test environment, so EnsureOwnedDir is a plain
	// MkdirAll with no chown — it must still create the directory.
	dir := filepath.Join(t.TempDir(), "x", "y")
	got, err := EnsureOwnedDir(dir)
	if err != nil {
		t.Fatalf("EnsureOwnedDir: %v", err)
	}
	if got != dir {
		t.Fatalf("EnsureOwnedDir returned %q, want %q", got, dir)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("directory not created: %v", err)
	}
}

func TestWriteOwnedFileWritesWithoutRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "file.txt")
	if err := WriteOwnedFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteOwnedFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("contents = %q, want hello", string(data))
	}
}

func TestVrooliPathResolvesThroughContract(t *testing.T) {
	// Resolves via the live repo contract (cwd is inside the repo during tests).
	home, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir: %v", err)
	}

	got, err := VrooliPath("processes", "scenarios", "demo")
	if err != nil {
		t.Fatalf("VrooliPath: %v", err)
	}
	want := filepath.Join(home, ".vrooli", "processes", "scenarios", "demo")
	if got != want {
		t.Fatalf("VrooliPath = %q, want %q", got, want)
	}

	if _, err := VrooliPath("not-a-real-entry"); err == nil {
		t.Fatal("VrooliPath(unknown key) expected error")
	}
}

func TestReconcileVrooliOwnershipNoopWithoutSudo(t *testing.T) {
	// In the test environment the process is not root-via-sudo, so reconcile is
	// a pure no-op that never touches the filesystem.
	changed, err := ReconcileVrooliOwnership()
	if err != nil {
		t.Fatalf("ReconcileVrooliOwnership: %v", err)
	}
	if changed != 0 {
		t.Fatalf("expected 0 reowned entries when not root-via-sudo, got %d", changed)
	}
}

func TestVrooliScopedPathResolvesThroughContract(t *testing.T) {
	home, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir: %v", err)
	}
	got, err := VrooliScopedPath("project_state", map[string]string{"project_key": "abc123"})
	if err != nil {
		t.Fatalf("VrooliScopedPath: %v", err)
	}
	want := filepath.Join(home, ".vrooli", "state", "projects", "abc123")
	if got != want {
		t.Fatalf("VrooliScopedPath = %q, want %q", got, want)
	}
}
