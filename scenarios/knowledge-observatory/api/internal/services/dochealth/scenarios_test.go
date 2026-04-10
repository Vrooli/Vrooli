package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListScenarios(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "alpha", "docs"), 0o755); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "beta", "docs"), 0o755); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "alpha", "README.md"), []byte("Alpha"), 0o644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "beta", "docs", "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	service, err := NewService(root)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	scenarios, err := service.ListScenarios(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(scenarios))
	}
	if scenarios[0].Name == "" || scenarios[1].Name == "" {
		t.Fatalf("expected scenario names")
	}
}
