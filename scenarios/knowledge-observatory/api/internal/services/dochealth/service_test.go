package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"knowledge-observatory/internal/doclogs"
)

func TestValidateScenario(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "alpha")
	if err := os.MkdirAll(filepath.Join(scenario, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(scenario, "README.md"), "# Readme")
	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), "{}")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")

	service, err := NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	result, err := service.ValidateScenario(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if result.TotalDocs == 0 {
		t.Fatalf("expected doc count to be positive")
	}
	if result.Validation == nil || result.Validation.ScenarioName != "alpha" {
		t.Fatalf("unexpected validation result: %#v", result.Validation)
	}
}

func TestResetScenarioDoc(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "alpha")
	if err := os.MkdirAll(filepath.Join(scenario, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "# Problems\n\n## Entries\n\n### 2025-01-01 - Old\n"
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), content)

	service, err := NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	config := doclogs.ResetConfig{MaxAgeDays: 1, PreviewMode: true}
	result, _, err := service.ResetScenarioDoc(context.Background(), "alpha", "problems", config)
	if err != nil {
		t.Fatalf("ResetScenarioDoc: %v", err)
	}
	if result == nil {
		t.Fatalf("expected reset result")
	}
}

func TestScenarioNameValidation(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	if _, err := service.ValidateScenario(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty scenario name")
	}
	if _, err := service.ValidateScenario(context.Background(), "../secrets"); err == nil {
		t.Fatalf("expected error for invalid scenario name")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
