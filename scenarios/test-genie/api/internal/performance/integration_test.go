package performance

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Integration Tests - Full Runner with Real Filesystem
// =============================================================================

// TestIntegration_ValidScenarioWithGoAndUI tests the full runner with both Go and UI.
func TestIntegration_ValidScenarioWithGoAndUI(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI, withUI)

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v\nRemediation: %s", result.Error, result.Remediation)
	}
	if !result.Summary.GoBuildPassed {
		t.Error("expected Go build to pass")
	}
	if !result.Summary.UIBuildPassed {
		t.Error("expected UI build to pass")
	}
	if result.Summary.GoBuildDuration == 0 {
		t.Error("expected Go build duration to be recorded")
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

// TestIntegration_GoOnlyScenario tests the runner with just Go API.
func TestIntegration_GoOnlyScenario(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !result.Summary.GoBuildPassed {
		t.Error("expected Go build to pass")
	}
	if !result.Summary.UIBuildSkipped {
		t.Error("expected UI build to be skipped when no ui/ directory")
	}
}

// TestIntegration_MissingAPIDirectory tests failure when api/ is missing.
func TestIntegration_MissingAPIDirectory(t *testing.T) {
	scenario := newIntegrationScenario(t) // No api/ or ui/

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when api directory missing")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

// TestIntegration_InvalidGoCode tests failure when Go code has syntax errors.
func TestIntegration_InvalidGoCode(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	// Corrupt the main.go file
	mainPath := filepath.Join(scenario.dir, "api", "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n\nfunc main( { }"), 0o644); err != nil {
		t.Fatalf("failed to corrupt main.go: %v", err)
	}

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure for invalid Go code")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
	if !result.Summary.GoBuildPassed {
		// Expected - build failed
	}
}

// TestIntegration_UIBuildSkippedWhenNoBuildScript tests that UI is skipped without build script.
func TestIntegration_UIBuildSkippedWhenNoBuildScript(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI, withUIWithoutBuild)

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !result.Summary.UIBuildSkipped {
		t.Error("expected UI build to be skipped when no build script")
	}
}

// TestIntegration_ContextCancellation tests proper handling of cancelled context.
func TestIntegration_ContextCancellation(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

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
	scenario := newIntegrationScenario(t, withGoAPI)

	// Write custom testing.json
	testingConfig := `{
		"performance": {
			"go_build_max_seconds": 300,
			"ui_build_max_seconds": 600,
			"require_go_build": true,
			"require_ui_build": false
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

	if expectations.GoBuildMaxDuration != 300*time.Second {
		t.Errorf("expected GoBuildMaxDuration=300s, got %s", expectations.GoBuildMaxDuration)
	}
	if expectations.UIBuildMaxDuration != 600*time.Second {
		t.Errorf("expected UIBuildMaxDuration=600s, got %s", expectations.UIBuildMaxDuration)
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

// TestIntegration_LoadExpectationsDefaultsWhenMissing tests default expectations.
func TestIntegration_LoadExpectationsDefaultsWhenMissing(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	expectations, err := LoadExpectations(scenario.dir)
	if err != nil {
		t.Fatalf("failed to load expectations: %v", err)
	}

	defaults := DefaultExpectations()
	if expectations.GoBuildMaxDuration != defaults.GoBuildMaxDuration {
		t.Errorf("expected default GoBuildMaxDuration, got %s", expectations.GoBuildMaxDuration)
	}
	if expectations.UIBuildMaxDuration != defaults.UIBuildMaxDuration {
		t.Errorf("expected default UIBuildMaxDuration, got %s", expectations.UIBuildMaxDuration)
	}
}

// TestIntegration_LoadExpectationsInvalidJSON tests error handling for invalid JSON.
func TestIntegration_LoadExpectationsInvalidJSON(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	// Write invalid testing.json
	vrooliDir := filepath.Join(scenario.dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}

	_, err := LoadExpectations(scenario.dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' error, got: %v", err)
	}
}

// TestIntegration_NilExpectationsUsesDefaults tests that nil expectations use defaults.
func TestIntegration_NilExpectationsUsesDefaults(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: nil, // Explicitly nil
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success with nil expectations, got: %v", result.Error)
	}
}

// TestIntegration_LoggingWorks tests that logging produces output.
func TestIntegration_LoggingWorks(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	var buf strings.Builder
	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(&buf))

	runner.Run(context.Background())

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected logging output")
	}
	if !strings.Contains(output, "Building") && !strings.Contains(output, "SUCCESS") {
		t.Errorf("expected build-related log messages, got: %s", output)
	}
}

// TestIntegration_SummaryString tests that summary produces readable output.
func TestIntegration_SummaryString(t *testing.T) {
	scenario := newIntegrationScenario(t, withGoAPI)

	runner := New(Config{
		ScenarioDir:  scenario.dir,
		ScenarioName: scenario.name,
		Expectations: DefaultExpectations(),
	}, WithLogger(io.Discard))

	result := runner.Run(context.Background())

	summary := result.Summary.String()
	if summary == "" {
		t.Error("expected non-empty summary string")
	}
	if !strings.Contains(summary, "Go build") {
		t.Errorf("expected 'Go build' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "UI build") {
		t.Errorf("expected 'UI build' in summary, got: %s", summary)
	}
}

// =============================================================================
// Integration Test Harness
// =============================================================================

type integrationScenario struct {
	dir  string
	name string
}

type scenarioOption func(t *testing.T, dir string)

func withGoAPI(t *testing.T, dir string) {
	t.Helper()

	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	// Create minimal go.mod
	goMod := `module test-scenario

go 1.21
`
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create minimal main.go
	mainGo := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
}

func withUI(t *testing.T, dir string) {
	t.Helper()

	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	// Create package.json with build script that just echoes (fast for testing)
	pkgJSON := `{
	"name": "test-ui",
	"version": "1.0.0",
	"scripts": {
		"build": "echo 'build complete'"
	}
}
`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	// Create node_modules to skip install step
	nodeModules := filepath.Join(uiDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
}

func withUIWithoutBuild(t *testing.T, dir string) {
	t.Helper()

	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	// Create package.json WITHOUT build script
	pkgJSON := `{
	"name": "test-ui",
	"version": "1.0.0",
	"scripts": {}
}
`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
}

func newIntegrationScenario(t *testing.T, opts ...scenarioOption) *integrationScenario {
	t.Helper()

	root := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)

	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	for _, opt := range opts {
		opt(t, scenarioDir)
	}

	return &integrationScenario{
		dir:  scenarioDir,
		name: scenarioName,
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkIntegration_GoOnlyScenario(b *testing.B) {
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

	apiDir := filepath.Join(scenarioDir, "api")
	os.MkdirAll(apiDir, 0o755)

	goMod := `module bench-scenario

go 1.21
`
	os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte(goMod), 0o644)

	mainGo := `package main

func main() {}
`
	os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(mainGo), 0o644)
}
