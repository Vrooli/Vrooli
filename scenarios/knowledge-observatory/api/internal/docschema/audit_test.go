package docschema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditScenarioDocumentation_Basic(t *testing.T) {
	scenario := t.TempDir()

	writeFile(t, filepath.Join(scenario, "README.md"), "# Test Scenario")
	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), `{
		"sections": [
			{"documents": [{"path": "internal/PROBLEMS.md"}]}
		]
	}`)
	writeFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Known Issues")

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("AuditScenarioDocumentation error: %v", err)
	}
	if result.ScenarioName == "" {
		t.Fatal("expected scenario name")
	}
	if result.Infrastructure == nil {
		t.Fatal("expected infrastructure result")
	}
	if result.TotalDocs < 2 {
		t.Fatalf("expected at least 2 docs, got %d", result.TotalDocs)
	}
}

func TestAuditScenarioDocumentation_EmptyPath(t *testing.T) {
	_, err := AuditScenarioDocumentation("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestAuditScenarioDocumentation_CodeWithoutDocRefs(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "api", "handler.go"), `package main

func HandleRequest() {}
func ProcessData() {}
`)
	writeFile(t, filepath.Join(scenario, "api", "documented.go"), `package main

// DOC: docs/reference/api.md
func DocumentedFunc() {}
`)

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.CodeWithoutDocRefs) != 1 {
		t.Fatalf("expected 1 undocumented file, got %d: %v", len(result.CodeWithoutDocRefs), result.CodeWithoutDocRefs)
	}
	if result.CodeWithoutDocRefs[0].Path != "api/handler.go" {
		t.Fatalf("expected api/handler.go, got %s", result.CodeWithoutDocRefs[0].Path)
	}
	if result.CodeWithoutDocRefs[0].ExportedSymbols != 2 {
		t.Fatalf("expected 2 exported symbols, got %d", result.CodeWithoutDocRefs[0].ExportedSymbols)
	}
}

func TestAuditScenarioDocumentation_SkipsTestFiles(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "api", "handler_test.go"), `package main

func TestSomething() {}
`)
	writeFile(t, filepath.Join(scenario, "ui", "src", "app.test.ts"), `export function testHelper() {}
`)

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.CodeWithoutDocRefs) != 0 {
		t.Fatalf("expected no undocumented files (test files should be skipped), got %d", len(result.CodeWithoutDocRefs))
	}
}

func TestAuditScenarioDocumentation_BrokenCodeRefs(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "docs", "guide.md"), `# Guide

See [CODE: src/missing.ts#func] for details.
Also [CODE: api/handler.go] works fine.
`)
	writeFile(t, filepath.Join(scenario, "api", "handler.go"), `package main

func Handler() {}
`)

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.BrokenCodeRefs) != 1 {
		t.Fatalf("expected 1 broken ref, got %d: %v", len(result.BrokenCodeRefs), result.BrokenCodeRefs)
	}
	if result.BrokenCodeRefs[0].Target != "src/missing.ts#func" {
		t.Fatalf("unexpected broken ref target: %s", result.BrokenCodeRefs[0].Target)
	}
}

func TestAuditScenarioDocumentation_OrphanedDocs(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "docs", "manifest.json"), `{
		"sections": [
			{"documents": [{"path": "guide.md"}]}
		]
	}`)
	writeFile(t, filepath.Join(scenario, "docs", "guide.md"), "# Guide")
	writeFile(t, filepath.Join(scenario, "docs", "orphan.md"), "# Orphan")

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.OrphanedDocs) != 1 {
		t.Fatalf("expected 1 orphaned doc, got %d: %v", len(result.OrphanedDocs), result.OrphanedDocs)
	}
	if result.OrphanedDocs[0] != "docs/orphan.md" {
		t.Fatalf("expected docs/orphan.md, got %s", result.OrphanedDocs[0])
	}
}

func TestAuditScenarioDocumentation_DuplicateTitles(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "docs", "one.md"), "# Getting Started\n\nSome content")
	writeFile(t, filepath.Join(scenario, "docs", "two.md"), "# Getting Started\n\nDifferent content")

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.DuplicateTitles) != 1 {
		t.Fatalf("expected 1 duplicate title, got %d: %v", len(result.DuplicateTitles), result.DuplicateTitles)
	}
	if result.DuplicateTitles[0].Title != "Getting Started" {
		t.Fatalf("unexpected title: %s", result.DuplicateTitles[0].Title)
	}
}

func TestAuditScenarioDocumentation_UndocumentedTargets(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	writeFile(t, filepath.Join(scenario, "PRD.md"), `# PRD
## Targets
- OT-P0-001: Auth
- OT-P0-002: Data
`)
	writeFile(t, filepath.Join(scenario, "docs", "auth.md"), "# Auth\nImplements OT-P0-001")

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.UndocumentedTargets) != 1 {
		t.Fatalf("expected 1 undocumented target, got %d: %v", len(result.UndocumentedTargets), result.UndocumentedTargets)
	}
	if result.UndocumentedTargets[0] != "OT-P0-002" {
		t.Fatalf("expected OT-P0-002, got %s", result.UndocumentedTargets[0])
	}
}

func TestAuditScenarioDocumentation_NoPRD(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.UndocumentedTargets) != 0 {
		t.Fatalf("expected no undocumented targets without PRD, got %d", len(result.UndocumentedTargets))
	}
}

func TestAuditScenarioDocumentation_SkipsDirs(t *testing.T) {
	scenario := t.TempDir()
	writeFile(t, filepath.Join(scenario, "README.md"), "# Scenario")
	// Files inside node_modules should be skipped.
	if err := os.MkdirAll(filepath.Join(scenario, "ui", "src", "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(scenario, "ui", "src", "node_modules", "pkg", "index.ts"),
		"export function LibFunc() {}")

	result, err := AuditScenarioDocumentation(scenario)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result.CodeWithoutDocRefs) != 0 {
		t.Fatalf("expected node_modules to be skipped, got %d files", len(result.CodeWithoutDocRefs))
	}
}
