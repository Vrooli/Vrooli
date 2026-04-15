package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetVrooliRoot_UsesCanonicalEnvRoot(t *testing.T) {
	root := newLandingManagerContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "landing-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	got := GetVrooliRoot()
	if got != root {
		t.Fatalf("expected %s, got %s", root, got)
	}
}

func TestGetVrooliRootCanonicalizesContractDescendantOverride(t *testing.T) {
	root := newLandingManagerContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "landing-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	if got := GetVrooliRoot(); got != root {
		t.Fatalf("expected %s, got %s", root, got)
	}
}

func TestResolveScenarioPathUsesCanonicalRepoRoot(t *testing.T) {
	root := newLandingManagerContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "landing-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	target := filepath.Join(root, "scenarios", "alpha")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	loc := ResolveScenarioPath("alpha")
	if !loc.Found {
		t.Fatalf("expected scenario to be found")
	}
	if loc.Path != target {
		t.Fatalf("expected %s, got %s", target, loc.Path)
	}
	if !loc.IsProduction() {
		t.Fatalf("expected production location, got %s", loc.Location)
	}
}

func TestIsScenarioNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "scenario not found message",
			output:   "[ERROR] Scenario not found: nonexistent (path: /tmp/nope)",
			expected: true,
		},
		{
			name:     "psql not found should be ignored",
			output:   "psql not found; runtime will auto-init schema",
			expected: false,
		},
		{
			name:     "lifecycle log missing",
			output:   "No lifecycle log found for scenario test-scenario",
			expected: true,
		},
		{
			name:     "no such scenario message",
			output:   "Error: No such scenario 'foo'",
			expected: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsScenarioNotFound(tt.output); got != tt.expected {
				t.Fatalf("expected %v for output %q, got %v", tt.expected, tt.output, got)
			}
		})
	}
}

func newLandingManagerContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := landingManagerRepoRoot(t)
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/landing-manager-util-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func landingManagerRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}
