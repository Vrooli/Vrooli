package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
)

// setupTestServer creates a Server backed by a temp directory that satisfies
// FindRepoRoot (has .vrooli/, scenarios/, resources/ markers) and a fresh
// ConfigStore with expensive/external command rules disabled.
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	root := setupFixTestDir(t)

	// ConfigStore in a temp config dir.
	configDir := filepath.Join(root, "config")
	mkdirAll(t, configDir)
	configPath := filepath.Join(configDir, "rules.json")
	cs := NewConfigStore(configPath)

	// Seed config so command-running rules don't fire on bare test scenarios.
	cfg := RulesConfig{
		Version: "1.0.0",
		EnabledRules: map[string]bool{
			"PACKAGE_GOVERNANCE_SCENARIO_ADOPTION": false,
			"REACT_VITE_UI_INSTALLS_DEPENDENCIES":  false,
		},
	}
	if err := cs.Save(t.Context(), cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	srv := &Server{
		config:       &Config{Port: "0"},
		router:       mux.NewRouter(),
		scenarioRoot: root, // FindRepoRoot(root) will resolve to root itself
		configStore:  cs,
	}
	srv.setupRoutes()

	return srv, root
}

func setupFixTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Create repo-root markers that the shared repo contract requires.
	for _, dir := range []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		mkdirAll(t, filepath.Join(root, dir))
	}
	writeRepoContractFixture(t, root)
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/test\n\ngo 1.25.0\n")
	return root
}

// mkdirAll is a test helper that creates a directory and fails the test on error.
func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo-contract fixture: %v", err)
	}
	writeTestFile(t, filepath.Join(root, ".vrooli", "repo-contract.json"), string(data))
}
