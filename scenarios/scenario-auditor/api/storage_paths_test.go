package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
)

func TestResolveScenarioAuditorStoragePathUsesRequestedClassRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_CONFIG_ROOT", filepath.Join(root, "config-root"))
	t.Setenv("VROOLI_DATA_ROOT", filepath.Join(root, "data-root"))
	t.Setenv("VROOLI_CACHE_ROOT", filepath.Join(root, "cache-root"))
	t.Setenv("VROOLI_LOGS_ROOT", filepath.Join(root, "logs-root"))
	t.Setenv("VROOLI_STATE_ROOT", filepath.Join(root, "state-root"))

	tests := []struct {
		class storage.Class
		rel   string
		root  string
	}{
		{class: storage.ClassConfig, rel: "rule-preferences.json", root: filepath.Join(root, "config-root")},
		{class: storage.ClassData, rel: "automated-fix-history.json", root: filepath.Join(root, "data-root")},
		{class: storage.ClassCache, rel: "standards-violations.json", root: filepath.Join(root, "cache-root")},
		{class: storage.ClassLogs, rel: "audit.log", root: filepath.Join(root, "logs-root")},
		{class: storage.ClassState, rel: "runtime.json", root: filepath.Join(root, "state-root")},
	}

	for _, tc := range tests {
		path, err := resolveScenarioAuditorStoragePath(tc.class, tc.rel)
		if err != nil {
			t.Fatalf("resolveScenarioAuditorStoragePath(%s): %v", tc.class, err)
		}

		want := filepath.Join(tc.root, "vrooli", scenarioAuditorScenarioID, tc.rel)
		if path != want {
			t.Fatalf("path for %s = %q, want %q", tc.class, path, want)
		}

		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			t.Fatalf("expected parent directory for %s to exist: %v", tc.class, err)
		}
	}
}
