package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// setupTestServer creates a Server backed by a temp directory that satisfies
// FindRepoRoot (has .vrooli/, scenarios/, resources/ markers) and a fresh
// ConfigStore with only MAKEFILE_* rules enabled.
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	root := t.TempDir()

	// Create repo-root markers that FindRepoRoot requires.
	for _, dir := range []string{".vrooli", "scenarios", "resources"} {
		mkdirAll(t, filepath.Join(root, dir))
	}

	// ConfigStore in a temp config dir.
	configDir := filepath.Join(root, "config")
	mkdirAll(t, configDir)
	configPath := filepath.Join(configDir, "rules.json")
	cs := NewConfigStore(configPath)

	// Seed config with only MAKEFILE_* rules enabled so GO_CLI and
	// REACT_VITE rules don't fire on bare test scenarios.
	cfg := RulesConfig{
		Version: "1.0.0",
		EnabledRules: map[string]bool{
			"GO_CLI_WORKSPACE_INDEPENDENCE":       false,
			"REACT_VITE_UI_INSTALLS_DEPENDENCIES": false,
			"MAKEFILE_STRUCTURE":                  true,
			"MAKEFILE_LIFECYCLE":                  true,
			"MAKEFILE_QUALITY":                    true,
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

// mkdirAll is a test helper that creates a directory and fails the test on error.
func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
