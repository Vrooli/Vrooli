package docvalidation

import (
	"path/filepath"
	"testing"
)

func TestKnowledgeObservatoryDocumentationContract(t *testing.T) {
	root := repoRootForTest(t)
	result, err := ValidateScenarioDocumentation(filepath.Join(root, "scenarios", "knowledge-observatory"))
	if err != nil {
		t.Fatalf("ValidateScenarioDocumentation: %v", err)
	}
	if result.ManifestStatus != "present" {
		t.Fatalf("expected scenario manifest, got %s", result.ManifestStatus)
	}
	if len(result.ContractFindings) != 0 {
		t.Fatalf("contract findings: %#v", result.ContractFindings)
	}
	if len(result.MissingDocs) != 0 {
		t.Fatalf("missing docs: %#v", result.MissingDocs)
	}
	if len(result.ExtraDocs) != 0 {
		t.Fatalf("extra docs: %#v", result.ExtraDocs)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "scenarios", "knowledge-observatory", "docs", "manifest.json")
		if _, err := filepath.Abs(candidate); err == nil {
			if result, validateErr := ValidateScenarioDocumentation(filepath.Join(dir, "scenarios", "knowledge-observatory")); validateErr == nil && result.ScenarioName == "knowledge-observatory" {
				return dir
			}
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatal("repo root not found")
		}
		dir = next
	}
}
