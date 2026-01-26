package docschema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateScenarioDocumentation(t *testing.T) {
	scenario := t.TempDir()

	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "PRD.md"), "# PRD")

	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), "{}")
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")
	writeFile(t, filepath.Join(scenario, "docs", "PROGRESS.md"), "# Progress")
	writeFile(t, filepath.Join(scenario, "docs", "QUICKSTART.md"), "# Quickstart")
	writeFile(t, filepath.Join(scenario, "docs", "concepts", "ARCHITECTURE.md"), "# Architecture")
	writeFile(t, filepath.Join(scenario, "docs", "misc", "NOTE.md"), "# Note")

	result, err := ValidateScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("ValidateScenarioDocumentation returned error: %v", err)
	}

	if !hasMisplacedDoc(result.MisplacedDocs, DocTypeProgress, "docs/PROGRESS.md", "docs/internal/PROGRESS.md") {
		t.Fatalf("expected misplaced progress doc, got: %#v", result.MisplacedDocs)
	}

	if !containsDocType(result.MissingDocs, DocTypeProgress) {
		t.Fatalf("expected missing progress doc")
	}

	if !containsString(result.ExtraDocs, "docs/misc/NOTE.md") {
		t.Fatalf("expected extra docs to include note, got: %#v", result.ExtraDocs)
	}

	if result.HealthScore < 0 || result.HealthScore > 1 {
		t.Fatalf("expected health score between 0 and 1, got %f", result.HealthScore)
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

func containsDocType(values []DocType, target DocType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasMisplacedDoc(docs []MisplacedDoc, docType DocType, actual string, expected string) bool {
	for _, doc := range docs {
		if doc.DocType == docType && doc.ActualPath == actual && doc.ExpectedPath == expected {
			return true
		}
	}
	return false
}
