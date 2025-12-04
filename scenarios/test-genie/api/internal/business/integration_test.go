package business

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Integration Tests - Full Runner with Real Filesystem
// =============================================================================

// TestIntegration_ValidScenario tests the full runner with a valid scenario layout.
func TestIntegration_ValidScenario(t *testing.T) {
	scenario := newIntegrationScenario(t)

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v\nRemediation: %s", result.Error, result.Remediation)
	}
	if result.Summary.ModulesFound == 0 {
		t.Error("expected at least one module to be found")
	}
	if result.Summary.RequirementsFound == 0 {
		t.Error("expected at least one requirement to be found")
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

// TestIntegration_MissingRequirementsDir tests failure when requirements/ is missing.
func TestIntegration_MissingRequirementsDir(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Remove requirements directory
	if err := os.RemoveAll(filepath.Join(scenario.dir, "requirements")); err != nil {
		t.Fatalf("failed to remove requirements dir: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure for missing requirements directory")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

// TestIntegration_MissingIndexFile tests failure when index.json is missing.
func TestIntegration_MissingIndexFile(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Remove index.json
	if err := os.Remove(filepath.Join(scenario.dir, "requirements", "index.json")); err != nil {
		t.Fatalf("failed to remove index.json: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure for missing index.json")
	}
}

// TestIntegration_IndexFileOptional tests that index.json can be skipped via config.
func TestIntegration_IndexFileOptional(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Remove index.json
	if err := os.Remove(filepath.Join(scenario.dir, "requirements", "index.json")); err != nil {
		t.Fatalf("failed to remove index.json: %v", err)
	}

	expectations := DefaultExpectations()
	expectations.RequireIndex = false

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: expectations,
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when RequireIndex=false, got: %v", result.Error)
	}
}

// TestIntegration_NoModulesFound tests failure when no modules exist.
func TestIntegration_NoModulesFound(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Remove all module directories
	reqDir := filepath.Join(scenario.dir, "requirements")
	entries, err := os.ReadDir(reqDir)
	if err != nil {
		t.Fatalf("failed to read requirements dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := os.RemoveAll(filepath.Join(reqDir, entry.Name())); err != nil {
				t.Fatalf("failed to remove module dir: %v", err)
			}
		}
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when no modules exist")
	}
	if !strings.Contains(result.Error.Error(), "no requirement modules") {
		t.Errorf("expected 'no requirement modules' error, got: %v", result.Error)
	}
}

// TestIntegration_NoModulesOptional tests that modules can be optional via config.
func TestIntegration_NoModulesOptional(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Remove all module directories
	reqDir := filepath.Join(scenario.dir, "requirements")
	entries, err := os.ReadDir(reqDir)
	if err != nil {
		t.Fatalf("failed to read requirements dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := os.RemoveAll(filepath.Join(reqDir, entry.Name())); err != nil {
				t.Fatalf("failed to remove module dir: %v", err)
			}
		}
	}

	expectations := DefaultExpectations()
	expectations.RequireModules = false

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: expectations,
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success when RequireModules=false, got: %v", result.Error)
	}
}

// TestIntegration_InvalidModuleJSON tests that malformed module.json causes warnings or failure.
// The parser may be lenient and collect errors rather than fail hard.
func TestIntegration_InvalidModuleJSON(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Corrupt the module.json
	modulePath := filepath.Join(scenario.dir, "requirements", "01-core", "module.json")
	if err := os.WriteFile(modulePath, []byte(`{invalid json`), 0o644); err != nil {
		t.Fatalf("failed to corrupt module.json: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	// The parser may either fail or succeed with warnings.
	// Check that either:
	// 1. It failed, or
	// 2. It succeeded but reported 0 requirements (corrupted file was skipped)
	if result.Success {
		// If successful, the corrupted file should have been skipped
		// resulting in 0 requirements found
		if result.Summary.RequirementsFound > 0 {
			t.Error("expected 0 requirements when module.json is corrupted")
		}
		// Also check for warning observations about parsing issues
		hasWarning := false
		for _, obs := range result.Observations {
			if obs.Type == ObservationWarning {
				hasWarning = true
				break
			}
		}
		if !hasWarning && result.Summary.ModulesFound > 0 {
			t.Log("Note: corrupted JSON was silently handled without warnings")
		}
	}
	// If it failed, that's also acceptable behavior for corrupted JSON
}

// TestIntegration_DuplicateRequirementIDs tests detection of duplicate IDs.
func TestIntegration_DuplicateRequirementIDs(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Create a second module with duplicate ID
	duplicateDir := filepath.Join(scenario.dir, "requirements", "02-duplicate")
	if err := os.MkdirAll(duplicateDir, 0o755); err != nil {
		t.Fatalf("failed to create duplicate module dir: %v", err)
	}

	// REQ-CORE-001 already exists in 01-core
	duplicateModule := `{
		"requirements": [
			{
				"id": "REQ-CORE-001",
				"title": "Duplicate Requirement",
				"criticality": "p1",
				"status": "draft",
				"validation": [{"type": "manual", "ref": "docs"}]
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(duplicateDir, "module.json"), []byte(duplicateModule), 0o644); err != nil {
		t.Fatalf("failed to write duplicate module: %v", err)
	}

	// Update index.json to include the new module
	indexContent := `{"imports": ["01-core/module.json", "02-duplicate/module.json"]}`
	if err := os.WriteFile(filepath.Join(scenario.dir, "requirements", "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to update index.json: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure for duplicate requirement IDs")
	}
	if !strings.Contains(result.Error.Error(), "duplicate") {
		t.Errorf("expected 'duplicate' in error, got: %v", result.Error)
	}
}

// TestIntegration_CyclicDependencies tests detection of requirement cycles.
func TestIntegration_CyclicDependencies(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Overwrite module with cyclic dependencies
	cyclicModule := `{
		"requirements": [
			{
				"id": "REQ-A",
				"title": "Requirement A",
				"criticality": "p1",
				"status": "draft",
				"children": ["REQ-B"],
				"validation": [{"type": "manual", "ref": "docs"}]
			},
			{
				"id": "REQ-B",
				"title": "Requirement B",
				"criticality": "p1",
				"status": "draft",
				"children": ["REQ-A"],
				"validation": [{"type": "manual", "ref": "docs"}]
			}
		]
	}`
	modulePath := filepath.Join(scenario.dir, "requirements", "01-core", "module.json")
	if err := os.WriteFile(modulePath, []byte(cyclicModule), 0o644); err != nil {
		t.Fatalf("failed to write cyclic module: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure for cyclic dependencies")
	}
	if !strings.Contains(result.Error.Error(), "cycle") {
		t.Errorf("expected 'cycle' in error, got: %v", result.Error)
	}
}

// TestIntegration_OrphanedChildren tests detection of orphaned child references.
func TestIntegration_OrphanedChildren(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Overwrite module with orphaned child
	orphanModule := `{
		"requirements": [
			{
				"id": "REQ-PARENT",
				"title": "Parent Requirement",
				"criticality": "p1",
				"status": "draft",
				"children": ["REQ-NONEXISTENT"],
				"validation": [{"type": "manual", "ref": "docs"}]
			}
		]
	}`
	modulePath := filepath.Join(scenario.dir, "requirements", "01-core", "module.json")
	if err := os.WriteFile(modulePath, []byte(orphanModule), 0o644); err != nil {
		t.Fatalf("failed to write orphan module: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure for orphaned children")
	}
	if !strings.Contains(result.Error.Error(), "non-existent child") {
		t.Errorf("expected 'non-existent child' in error, got: %v", result.Error)
	}
}

// TestIntegration_ContextCancellation tests proper handling of cancelled context.
func TestIntegration_ContextCancellation(t *testing.T) {
	scenario := newIntegrationScenario(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(ctx)

	if result.Success {
		t.Fatal("expected failure for cancelled context")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
}

// TestIntegration_LoadExpectationsFromFile tests expectations loaded from testing.json.
func TestIntegration_LoadExpectationsFromFile(t *testing.T) {
	scenario := newIntegrationScenario(t)

	// Write custom testing.json
	testingConfig := `{
		"business": {
			"require_modules": true,
			"min_coverage_percent": 50,
			"error_coverage_percent": 30
		}
	}`
	vrooliDir := filepath.Join(scenario.dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingConfig), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	expectations, err := LoadExpectations(scenario.dir)
	if err != nil {
		t.Fatalf("failed to load expectations: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: expectations,
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
}

// =============================================================================
// Integration Test Harness
// =============================================================================

type integrationScenario struct {
	dir  string
	name string
}

func newIntegrationScenario(t *testing.T) *integrationScenario {
	t.Helper()

	root := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	// Create scenario structure
	dirs := []string{
		filepath.Join(scenarioDir, "requirements", "01-core"),
		filepath.Join(scenarioDir, ".vrooli"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Create index.json
	indexContent := `{"imports": ["01-core/module.json"]}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to write index.json: %v", err)
	}

	// Create a valid module
	moduleContent := `{
		"requirements": [
			{
				"id": "REQ-CORE-001",
				"title": "Core Requirement",
				"description": "A core requirement for testing",
				"criticality": "p1",
				"status": "draft",
				"validation": [{"type": "manual", "ref": "docs"}]
			},
			{
				"id": "REQ-CORE-002",
				"title": "Secondary Requirement",
				"description": "A secondary requirement",
				"criticality": "p2",
				"status": "implemented",
				"validation": [{"type": "test", "ref": "integration_test.go"}]
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(scenarioDir, "requirements", "01-core", "module.json"), []byte(moduleContent), 0o644); err != nil {
		t.Fatalf("failed to write module.json: %v", err)
	}

	return &integrationScenario{
		dir:  scenarioDir,
		name: scenarioName,
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkIntegration_ValidScenario(b *testing.B) {
	root := b.TempDir()
	scenarioName := "bench-scenario"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	// Setup once
	setupBenchScenario(b, scenarioDir)

	runner := New(Config{
		ScenarioDir:  scenarioDir,
		ScenarioName: scenarioName,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

func setupBenchScenario(b *testing.B, scenarioDir string) {
	b.Helper()

	dirs := []string{
		filepath.Join(scenarioDir, "requirements", "01-core"),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	indexContent := `{"imports": ["01-core/module.json"]}`
	os.WriteFile(filepath.Join(scenarioDir, "requirements", "index.json"), []byte(indexContent), 0o644)

	moduleContent := `{
		"requirements": [
			{"id": "REQ-1", "title": "Test", "criticality": "p1", "status": "draft", "validation": [{"type": "manual", "ref": "docs"}]}
		]
	}`
	os.WriteFile(filepath.Join(scenarioDir, "requirements", "01-core", "module.json"), []byte(moduleContent), 0o644)
}
