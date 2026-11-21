// Package fixtures contains filesystem fixtures for API tests.
package fixtures

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteScenarioServiceJSON writes a service.json fixture for a scenario slug.
func WriteScenarioServiceJSON(t *testing.T, repoRoot, slug, content string) {
	t.Helper()

	dir := filepath.Join(repoRoot, "scenarios", slug, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create service fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write service fixture: %v", err)
	}
}
