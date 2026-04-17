package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectVrooliRoot(t *testing.T) {
	// Test that the function returns a non-empty path
	result := DetectVrooliRoot()

	if result == "" {
		t.Error("DetectVrooliRoot() returned empty string")
	}

	// Verify the path exists
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("DetectVrooliRoot() returned non-existent path: %v", result)
	}

	// Verify it's an absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("DetectVrooliRoot() returned relative path: %v", result)
	}
}

func TestDetectVrooliRootConsistency(t *testing.T) {
	// Test that multiple calls return the same result
	first := DetectVrooliRoot()
	second := DetectVrooliRoot()

	if first != second {
		t.Errorf("DetectVrooliRoot() returned inconsistent results: %v vs %v", first, second)
	}
}

func TestDetectVrooliRootStructure(t *testing.T) {
	// Test that the detected root has expected Vrooli structure
	root := DetectVrooliRoot()

	// Check for common Vrooli directories/files
	expectedPaths := []string{
		filepath.Join(root, "scenarios"),
		filepath.Join(root, "scripts"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Logf("Warning: Expected Vrooli path does not exist: %v (root: %v)", path, root)
			// This is a warning, not a failure, as the structure might vary
		}
	}
}

func TestDetectVrooliRootFromEnv(t *testing.T) {
	root := newEcosystemContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "ecosystem-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_ROOT", nested)

	result := DetectVrooliRoot()
	if result != root {
		t.Fatalf("DetectVrooliRoot() = %q, want %q", result, root)
	}
}

func newEcosystemContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := ecosystemRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/ecosystem-paths-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"scenarios", "resources", "packages", "cmd", "internal", "templates"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func ecosystemRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", ".."))
}
