package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
	"test-genie/internal/unit"
)

func TestRunUnitPhaseSuccess(t *testing.T) {
	// Create a minimal scenario with Go project
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	apiDir := filepath.Join(scenarioDir, "api")
	cliDir := filepath.Join(scenarioDir, "cli")

	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create go.mod to trigger Go runner detection
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module demo\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}

	// Note: This test will attempt to run actual go test, which may fail
	// The important thing is the orchestrator runs without panic
	report := runUnitPhase(context.Background(), env, io.Discard)

	// Should have observations regardless of success/failure
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunUnitPhaseContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  t.TempDir(),
	}

	report := runUnitPhase(ctx, env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected error when context cancelled")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("expected system classification, got %s", report.FailureClassification)
	}
}

func TestRunUnitPhaseNoLanguagesDetected(t *testing.T) {
	// Create empty scenario - no languages detected
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}

	report := runUnitPhase(context.Background(), env, io.Discard)

	// Should succeed with skip observations
	if report.Err != nil {
		t.Fatalf("expected success with no languages, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected skip observations")
	}
}

func TestRunUnitPhaseObservationsConverted(t *testing.T) {
	// Create empty scenario
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
	}

	report := runUnitPhase(context.Background(), env, io.Discard)

	// Verify observations are properly converted
	for _, obs := range report.Observations {
		// Check that observations have valid text (not empty or with null types)
		if obs.Text == "" && obs.Icon == "" {
			t.Errorf("observation has no text or icon")
		}
	}
}

func TestConvertUnitObservation_AllTypes(t *testing.T) {
	tests := []struct {
		name        string
		input       unit.Observation
		wantText    string
		wantSection string
		wantIcon    string
		wantPrefix  string
	}{
		{
			name:        "section",
			input:       unit.NewSectionObservation("🧪", "Testing"),
			wantSection: "Testing",
			wantIcon:    "🧪",
		},
		{
			name:       "success",
			input:      unit.NewSuccessObservation("passed"),
			wantText:   "passed",
			wantPrefix: "SUCCESS",
		},
		{
			name:       "warning",
			input:      unit.NewWarningObservation("caution"),
			wantText:   "caution",
			wantPrefix: "WARNING",
		},
		{
			name:       "error",
			input:      unit.NewErrorObservation("failed"),
			wantText:   "failed",
			wantPrefix: "ERROR",
		},
		{
			name:       "info",
			input:      unit.NewInfoObservation("note"),
			wantText:   "note",
			wantPrefix: "INFO",
		},
		{
			name:       "skip",
			input:      unit.NewSkipObservation("skipped"),
			wantText:   "skipped",
			wantPrefix: "SKIP",
		},
		{
			name:     "unknown type defaults to plain",
			input:    unit.Observation{Type: 999, Message: "unknown"},
			wantText: "unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the generic converter with the standard extractor
			results := ConvertObservationsGeneric([]unit.Observation{tc.input}, ExtractStandardObservation[unit.Observation])
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			result := results[0]

			// For section observations, check Section and Icon fields
			if tc.wantSection != "" {
				if result.Section != tc.wantSection {
					t.Errorf("Section = %q, want %q", result.Section, tc.wantSection)
				}
				if result.Icon != tc.wantIcon {
					t.Errorf("Icon = %q, want %q", result.Icon, tc.wantIcon)
				}
			} else {
				// For non-section observations, check Text and Prefix fields
				if result.Text != tc.wantText {
					t.Errorf("Text = %q, want %q", result.Text, tc.wantText)
				}
				if tc.wantPrefix != "" && result.Prefix != tc.wantPrefix {
					t.Errorf("Prefix = %q, want %q", result.Prefix, tc.wantPrefix)
				}
			}
		})
	}
}

func TestConvertUnitObservations_Empty(t *testing.T) {
	result := ConvertObservationsGeneric([]unit.Observation(nil), ExtractStandardObservation[unit.Observation])
	if len(result) != 0 {
		t.Errorf("ConvertObservationsGeneric(nil) = %d observations, want 0", len(result))
	}

	result = ConvertObservationsGeneric([]unit.Observation{}, ExtractStandardObservation[unit.Observation])
	if len(result) != 0 {
		t.Errorf("ConvertObservationsGeneric([]) = %d observations, want 0", len(result))
	}
}

func TestConvertUnitObservations_Multiple(t *testing.T) {
	input := []unit.Observation{
		unit.NewSectionObservation("🧪", "Section 1"),
		unit.NewSuccessObservation("Success 1"),
		unit.NewWarningObservation("Warning 1"),
	}

	result := ConvertObservationsGeneric(input, ExtractStandardObservation[unit.Observation])

	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}

	// Section observation uses Section field
	if result[0].Section != "Section 1" {
		t.Errorf("result[0].Section = %q, want %q", result[0].Section, "Section 1")
	}
	if result[0].Icon != "🧪" {
		t.Errorf("result[0].Icon = %q, want %q", result[0].Icon, "🧪")
	}
	// Success and Warning use Text field
	if result[1].Text != "Success 1" {
		t.Errorf("result[1].Text = %q, want %q", result[1].Text, "Success 1")
	}
	if result[2].Text != "Warning 1" {
		t.Errorf("result[2].Text = %q, want %q", result[2].Text, "Warning 1")
	}
}

func TestConvertUnitFailureClass_AllTypes(t *testing.T) {
	tests := []struct {
		input    shared.FailureClass
		expected shared.FailureClass
	}{
		{shared.FailureClassMisconfiguration, shared.FailureClassMisconfiguration},
		{shared.FailureClassMissingDependency, shared.FailureClassMissingDependency},
		{shared.FailureClassTestFailure, shared.FailureClassSystem}, // Maps to system
		{shared.FailureClassSystem, shared.FailureClassSystem},
		{shared.FailureClass("unknown"), shared.FailureClassSystem}, // Default to system
	}

	for _, tc := range tests {
		t.Run(string(tc.input), func(t *testing.T) {
			result := shared.StandardizeFailureClass(tc.input)
			if result != tc.expected {
				t.Errorf("StandardizeFailureClass(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ============================================================================
// Integration-style tests with real filesystem harness
// ============================================================================

type unitTestHarness struct {
	env         workspace.Environment
	scenarioDir string
	apiDir      string
	uiDir       string
	cliDir      string
}

func newUnitTestHarness(t *testing.T) *unitTestHarness {
	t.Helper()
	projectRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(projectRoot, "scenarios", scenarioName)

	apiDir := filepath.Join(scenarioDir, "api")
	uiDir := filepath.Join(scenarioDir, "ui")
	cliDir := filepath.Join(scenarioDir, "cli")

	// Create directories
	for _, dir := range []string{apiDir, uiDir, cliDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	return &unitTestHarness{
		env: workspace.Environment{
			ScenarioName: scenarioName,
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
			AppRoot:      filepath.Dir(filepath.Dir(scenarioDir)),
		},
		scenarioDir: scenarioDir,
		apiDir:      apiDir,
		uiDir:       uiDir,
		cliDir:      cliDir,
	}
}

// setupGoProject creates a minimal Go project structure
func (h *unitTestHarness) setupGoProject(t *testing.T) {
	t.Helper()
	// Create go.mod
	writeTestFile(t, filepath.Join(h.apiDir, "go.mod"), "module test-scenario\n\ngo 1.21\n")
	// Create main.go
	writeTestFile(t, filepath.Join(h.apiDir, "main.go"), `package main

func main() {}
`)
}

// setupGoProjectWithPassingTest creates a Go project with a passing test
func (h *unitTestHarness) setupGoProjectWithPassingTest(t *testing.T) {
	t.Helper()
	h.setupGoProject(t)
	// Create a passing test
	writeTestFile(t, filepath.Join(h.apiDir, "main_test.go"), `package main

import "testing"

func TestExample(t *testing.T) {
	// This test passes
}
`)
}

// setupNodeProject creates a minimal Node.js project structure
func (h *unitTestHarness) setupNodeProject(t *testing.T, hasTestScript bool) {
	t.Helper()
	testScript := `"echo \"Error: no test specified\" && exit 1"`
	if hasTestScript {
		testScript = `"echo tests passed"`
	}
	writeTestFile(t, filepath.Join(h.uiDir, "package.json"), `{
  "name": "test-ui",
  "version": "1.0.0",
  "scripts": {
    "test": `+testScript+`
  }
}`)
	// Create node_modules to skip install
	if err := os.MkdirAll(filepath.Join(h.uiDir, "node_modules"), 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
}

// setupShellCLI creates a minimal shell CLI structure
func (h *unitTestHarness) setupShellCLI(t *testing.T) {
	t.Helper()
	cliScript := filepath.Join(h.cliDir, h.env.ScenarioName)
	writeTestFile(t, cliScript, `#!/usr/bin/env bash
echo "CLI working"
`)
	if err := os.Chmod(cliScript, 0o755); err != nil {
		t.Fatalf("failed to chmod CLI script: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestRunUnitPhase_Integration_GoProjectWithPassingTest(t *testing.T) {
	h := newUnitTestHarness(t)
	h.setupGoProjectWithPassingTest(t)

	report := runUnitPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatal("expected observations to be recorded")
	}

	// Should have at least a section observation for Go tests
	hasSectionObs := false
	for _, obs := range report.Observations {
		if obs.Section != "" {
			hasSectionObs = true
			break
		}
	}
	if !hasSectionObs {
		t.Error("expected section observation for language tests")
	}
}

func TestRunUnitPhase_Integration_EmptyScenario(t *testing.T) {
	h := newUnitTestHarness(t)
	// Don't set up any language projects

	report := runUnitPhase(context.Background(), h.env, io.Discard)

	// Should succeed but with skip observations
	if report.Err != nil {
		t.Fatalf("expected success for empty scenario, got error: %v", report.Err)
	}

	// Should have skip observations for each language
	hasSkipObs := false
	for _, obs := range report.Observations {
		if obs.Prefix == "SKIP" {
			hasSkipObs = true
			break
		}
	}
	if !hasSkipObs {
		t.Error("expected skip observations for undetected languages")
	}
}

func TestRunUnitPhase_Integration_MultipleLanguages(t *testing.T) {
	h := newUnitTestHarness(t)
	h.setupGoProjectWithPassingTest(t)
	h.setupShellCLI(t)

	report := runUnitPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success, got error: %v", report.Err)
	}

	// Should have observations for both Go and Shell
	sectionCount := 0
	for _, obs := range report.Observations {
		if obs.Section != "" {
			sectionCount++
		}
	}
	// At minimum, should have section observations for multiple languages
	if sectionCount < 2 {
		t.Errorf("expected at least 2 section observations, got %d", sectionCount)
	}
}

func TestRunUnitPhase_Integration_NodeWithDefaultTestScript(t *testing.T) {
	h := newUnitTestHarness(t)
	h.setupNodeProject(t, false) // Default npm test script (should be skipped)

	report := runUnitPhase(context.Background(), h.env, io.Discard)

	// Should succeed (Node tests should be skipped, not failed)
	if report.Err != nil {
		t.Fatalf("expected success when Node has default test script, got error: %v", report.Err)
	}
}
