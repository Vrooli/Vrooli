package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverScenarioNames(t *testing.T) {
	// Create a temporary directory structure mimicking scenarios/<name>/store
	tmpDir := t.TempDir()

	// scenarios/
	scenariosDir := filepath.Join(tmpDir, "scenarios")
	if err := os.Mkdir(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create scenario directories
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := os.Mkdir(filepath.Join(scenariosDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a hidden directory (should be skipped)
	if err := os.Mkdir(filepath.Join(scenariosDir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file (should be skipped)
	if err := os.WriteFile(filepath.Join(scenariosDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	// storeDir mimics scenarios/alpha/store
	storeDir := filepath.Join(scenariosDir, "alpha", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	names := discoverScenarioNames(storeDir)

	expected := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	if len(names) != len(expected) {
		t.Fatalf("expected %d scenario names, got %d: %v", len(expected), len(names), names)
	}
	for _, n := range names {
		if !expected[n] {
			t.Errorf("unexpected scenario name: %s", n)
		}
	}
}

func TestDiscoverScenarioNames_NonexistentDir(t *testing.T) {
	names := discoverScenarioNames("/nonexistent/path/store")
	if len(names) != 0 {
		t.Fatalf("expected 0 names for nonexistent dir, got %d", len(names))
	}
}

func TestResolveQdrantURL(t *testing.T) {
	// Save original env and restore after test
	origURL := os.Getenv("QDRANT_URL")
	origBase := os.Getenv("QDRANT_BASE_URL")
	origPort := os.Getenv("QDRANT_PORT")
	t.Cleanup(func() {
		os.Setenv("QDRANT_URL", origURL)
		os.Setenv("QDRANT_BASE_URL", origBase)
		os.Setenv("QDRANT_PORT", origPort)
	})

	tests := []struct {
		name     string
		envURL   string
		envBase  string
		envPort  string
		expected string
	}{
		{
			name:     "QDRANT_URL takes priority",
			envURL:   "http://qdrant:6333",
			envBase:  "http://localhost:6333",
			envPort:  "6333",
			expected: "http://qdrant:6333",
		},
		{
			name:     "falls back to QDRANT_BASE_URL",
			envURL:   "",
			envBase:  "http://localhost:6333",
			envPort:  "6333",
			expected: "http://localhost:6333",
		},
		{
			name:     "constructs from QDRANT_PORT",
			envURL:   "",
			envBase:  "",
			envPort:  "6333",
			expected: "http://localhost:6333",
		},
		{
			name:     "custom port from QDRANT_PORT",
			envURL:   "",
			envBase:  "",
			envPort:  "16333",
			expected: "http://localhost:16333",
		},
		{
			name:     "returns empty when nothing set",
			envURL:   "",
			envBase:  "",
			envPort:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("QDRANT_URL", tt.envURL)
			os.Setenv("QDRANT_BASE_URL", tt.envBase)
			os.Setenv("QDRANT_PORT", tt.envPort)

			got := resolveQdrantURL()
			if got != tt.expected {
				t.Errorf("resolveQdrantURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}
