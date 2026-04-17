package reporoot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolvePrefersExplicitRootOverride(t *testing.T) {
	explicitRoot := newFixtureRepo(t)
	inheritedSourceRoot := repoRootFromCaller(t)

	env := map[string]string{
		"VROOLI_ROOT":        filepath.Join(explicitRoot, "scenarios", "vrooli-autoheal", "api"),
		"VROOLI_SOURCE_ROOT": inheritedSourceRoot,
	}

	if got := Resolve(func(key string) string { return env[key] }); got != explicitRoot {
		t.Fatalf("Resolve() = %q, want %q", got, explicitRoot)
	}
}

func TestCanonicalizeOverrideReturnsFixtureRoot(t *testing.T) {
	root := newFixtureRepo(t)
	nested := filepath.Join(root, "packages", "demo", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, ok := CanonicalizeOverride(nested)
	if !ok {
		t.Fatal("CanonicalizeOverride() did not resolve fixture root")
	}
	if got != root {
		t.Fatalf("CanonicalizeOverride() = %q, want %q", got, root)
	}
}

func newFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	contractData, err := os.ReadFile(filepath.Join(repoRootFromCaller(t), ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/reporoot-fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}
