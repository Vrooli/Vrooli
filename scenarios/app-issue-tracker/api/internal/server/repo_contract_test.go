package server

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverComponentsUsesConfiguredContractRoot(t *testing.T) {
	root := newServerContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "app-issue-tracker", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	for _, scenario := range []string{"alpha-scenario", "beta-scenario"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario), 0o755); err != nil {
			t.Fatalf("mkdir scenario %s: %v", scenario, err)
		}
	}

	server := &Server{config: &Config{VrooliRoot: root}}

	components, err := server.discoverComponents()
	if err != nil {
		t.Fatalf("discoverComponents() error = %v", err)
	}

	var found []string
	for _, component := range components {
		if component.Type == "scenario" {
			found = append(found, component.ID)
		}
	}

	if len(found) < 3 {
		t.Fatalf("expected discovered scenarios, got %v", found)
	}
	if !containsComponentID(found, "alpha-scenario") || !containsComponentID(found, "beta-scenario") {
		t.Fatalf("expected alpha-scenario and beta-scenario in %v", found)
	}
}

func containsComponentID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func newServerContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := appIssueTrackerServerRepoRoot(t)
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app-issue-tracker-server-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func appIssueTrackerServerRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}
