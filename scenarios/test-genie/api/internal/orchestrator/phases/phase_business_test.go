package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestRunBusinessPhaseValidatesModules(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}

	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected business phase to pass, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunBusinessPhaseFailsWhenModuleIncomplete(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	brokenDir := filepath.Join(scenarioDir, "requirements", "02-broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("failed to create broken module dir: %v", err)
	}
	payload := `{"requirements":[{"id":"","title":"","criticality":"","status":"","validation":[]}]}`
	if err := os.WriteFile(filepath.Join(brokenDir, "module.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write broken module: %v", err)
	}
	// Update index.json to include the broken module
	indexContent := `{"imports":["01-internal-orchestrator/module.json","02-broken/module.json"]}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to update requirements index: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected validation failure for malformed module")
	}
	// The new implementation detects missing ID as an error
	if !strings.Contains(report.Err.Error(), "missing") && !strings.Contains(report.Err.Error(), "02-broken") {
		t.Fatalf("expected error about missing fields or module reference, got %v", report.Err)
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", report.FailureClassification)
	}
	if report.Remediation == "" {
		t.Fatalf("expected remediation guidance")
	}
}

func TestRunBusinessPhaseFailsWhenNoModulesFound(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")
	if err := os.RemoveAll(filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")); err != nil {
		t.Fatalf("failed to remove seeded module: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected failure when no modules exist")
	}
	if !strings.Contains(report.Err.Error(), "no requirement modules") {
		t.Fatalf("unexpected error: %v", report.Err)
	}
}

func TestRunBusinessPhaseDetectsDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create a second module with a duplicate ID
	duplicateDir := filepath.Join(scenarioDir, "requirements", "02-duplicate")
	if err := os.MkdirAll(duplicateDir, 0o755); err != nil {
		t.Fatalf("failed to create duplicate module dir: %v", err)
	}
	// REQ-1 already exists in 01-internal-orchestrator
	payload := `{"requirements":[{"id":"REQ-1","title":"Duplicate","criticality":"p1","status":"draft","validation":[{"type":"manual","ref":"docs"}]}]}`
	if err := os.WriteFile(filepath.Join(duplicateDir, "module.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write duplicate module: %v", err)
	}
	// Update index.json to include the new module
	indexContent := `{"imports":["01-internal-orchestrator/module.json","02-duplicate/module.json"]}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to update requirements index: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected validation failure for duplicate IDs")
	}
	if !strings.Contains(report.Err.Error(), "duplicate") {
		t.Fatalf("expected duplicate ID error, got: %v", report.Err)
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", report.FailureClassification)
	}
}

func TestRunBusinessPhaseDetectsCycles(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Overwrite the module with a cycle (REQ-A -> REQ-B -> REQ-A)
	moduleDir := filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")
	payload := `{"requirements":[
		{"id":"REQ-A","title":"A","criticality":"p1","status":"draft","children":["REQ-B"],"validation":[{"type":"manual","ref":"docs"}]},
		{"id":"REQ-B","title":"B","criticality":"p1","status":"draft","children":["REQ-A"],"validation":[{"type":"manual","ref":"docs"}]}
	]}`
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write cyclic module: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected validation failure for cycle")
	}
	if !strings.Contains(report.Err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got: %v", report.Err)
	}
}

func TestRunBusinessPhaseDetectsOrphanedChildren(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Overwrite the module with an orphaned child reference
	moduleDir := filepath.Join(scenarioDir, "requirements", "01-internal-orchestrator")
	payload := `{"requirements":[
		{"id":"REQ-PARENT","title":"Parent","criticality":"p1","status":"draft","children":["REQ-NONEXISTENT"],"validation":[{"type":"manual","ref":"docs"}]}
	]}`
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write module with orphan: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}
	report := runBusinessPhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected validation failure for orphaned child")
	}
	if !strings.Contains(report.Err.Error(), "non-existent child") {
		t.Fatalf("expected orphaned child error, got: %v", report.Err)
	}
}
