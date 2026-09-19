package docvalidation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knowledge-observatory/internal/doccontract"
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

func TestValidateContent_TableContracts(t *testing.T) {
	doc := doccontract.Document{
		ScenarioPath: "docs/concepts/DOMAINS.md",
		DocType:      "domains",
		Validation: doccontract.Validation{
			TableContracts: []doccontract.TableContract{{
				AnchorHeading: "Domain Inventory",
				Columns: []doccontract.TableColumnContract{
					{Name: "Domain", Required: true, Type: "text"},
					{Name: "Responsibility", Required: true, Type: "text", Aliases: []string{"Purpose"}},
					{Name: "Primary Archetype", Required: true, Type: "enum", EnumValues: []string{"service", "reporting"}},
					{Name: "Source Paths", Required: true, Type: "comma-list", Aliases: []string{"Primary Paths"}},
				},
			}},
		},
	}
	content := `# Domains

## Domain Inventory

| Domain | Purpose | Primary Archetype | Primary Paths |
|---|---|---|---|
| graph | Build the graph. | service | ` + "`api/internal/graph/`" + ` |
| drift | Detect drift. | mystery | ` + "`api/internal/drift/`" + ` |
`
	scenarioPath := t.TempDir()
	abs := filepath.Join(scenarioPath, filepath.FromSlash(doc.ScenarioPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir doc dir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	issues := validateContent(scenarioPath, doc)
	if !hasIssueContaining(issues, `uses alias header "Purpose"`) {
		t.Fatalf("expected Purpose alias issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, `uses alias header "Primary Paths"`) {
		t.Fatalf("expected Primary Paths alias issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, `value "mystery" outside enum`) {
		t.Fatalf("expected enum issue, got %#v", issues)
	}
}

func TestValidateScenarioDocumentationIgnoresRootArchitectureManifest(t *testing.T) {
	scenarioPath := t.TempDir()
	writeFile(t, filepath.Join(scenarioPath, "manifest.json"), `{
		"contract": {
			"kind": "scenario-architecture",
			"schema": "scenario-architecture-manifest/v1"
		}
	}`)
	writeFile(t, filepath.Join(scenarioPath, "docs", "manifest.json"), `{
		"version": "2.0.0",
		"contract": {
			"kind": "scenario-docs",
			"schema": "scenario-docs-manifest/v2",
			"maturityValues": ["active"],
			"stages": ["generated"]
		},
		"sections": [{
			"id": "reference",
			"title": "Reference",
			"documents": [{
				"path": "manifest.json",
				"docType": "manifest",
				"title": "Documentation Manifest",
				"maturity": "active",
				"requiredBy": ["generated"],
				"completion": "required"
			}]
		}]
	}`)

	result, err := ValidateScenarioDocumentation(scenarioPath)
	if err != nil {
		t.Fatalf("ValidateScenarioDocumentation: %v", err)
	}
	if len(result.MisplacedDocs) != 0 {
		t.Fatalf("expected architecture manifest to be ignored, got misplaced docs: %#v", result.MisplacedDocs)
	}
	if len(result.ExtraDocs) != 0 {
		t.Fatalf("expected architecture manifest not to be treated as an extra doc, got %#v", result.ExtraDocs)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func hasIssueContaining(issues []DocContentIssue, needle string) bool {
	for _, issue := range issues {
		if strings.Contains(issue.Message, needle) {
			return true
		}
	}
	return false
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
