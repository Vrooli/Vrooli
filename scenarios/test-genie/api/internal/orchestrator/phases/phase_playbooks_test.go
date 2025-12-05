package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
	"test-genie/internal/shared"
)

// playbooksTestHarness provides a consistent test setup for playbooks phase tests.
type playbooksTestHarness struct {
	env         workspace.Environment
	scenarioDir string
	testDir     string
	appRoot     string
}

func newPlaybooksTestHarness(t *testing.T) *playbooksTestHarness {
	t.Helper()
	appRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(appRoot, "scenarios", scenarioName)

	// Create required directory structure
	requiredDirs := []string{
		"ui",
		"test/playbooks",
		".vrooli",
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	testDir := filepath.Join(scenarioDir, "test")

	return &playbooksTestHarness{
		env: workspace.Environment{
			ScenarioName: scenarioName,
			ScenarioDir:  scenarioDir,
			TestDir:      testDir,
			AppRoot:      appRoot,
		},
		scenarioDir: scenarioDir,
		testDir:     testDir,
		appRoot:     appRoot,
	}
}

func (h *playbooksTestHarness) writeRegistry(t *testing.T, content string) {
	t.Helper()
	registryDir := filepath.Join(h.testDir, "playbooks")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "registry.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write registry.json: %v", err)
	}
}

func (h *playbooksTestHarness) writeWorkflow(t *testing.T, relativePath, content string) {
	t.Helper()
	fullPath := filepath.Join(h.scenarioDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}
}

func (h *playbooksTestHarness) removeUI(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(h.scenarioDir, "ui")); err != nil {
		t.Fatalf("failed to remove ui dir: %v", err)
	}
}

// Tests for runPlaybooksPhase

func TestRunPlaybooksPhaseNoUIDirectory(t *testing.T) {
	h := newPlaybooksTestHarness(t)
	h.removeUI(t)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success when no UI directory, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatal("expected observations for skipped phase")
	}
	// Should have an info observation about missing UI
	hasSkipObs := false
	for _, obs := range report.Observations {
		// Check both Text field and the observation's String() representation
		obsStr := obs.String()
		if strings.Contains(obs.Text, "ui/") || strings.Contains(obs.Text, "missing") ||
			strings.Contains(obsStr, "ui/") || strings.Contains(obsStr, "missing") {
			hasSkipObs = true
			break
		}
	}
	if !hasSkipObs {
		t.Errorf("expected observation about missing ui directory, got: %v", report.Observations)
	}
}

func TestRunPlaybooksPhaseSkipViaEnv(t *testing.T) {
	h := newPlaybooksTestHarness(t)

	os.Setenv("TEST_GENIE_SKIP_PLAYBOOKS", "1")
	defer os.Unsetenv("TEST_GENIE_SKIP_PLAYBOOKS")

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success when skipped via env, got error: %v", report.Err)
	}
}

func TestRunPlaybooksPhaseEmptyRegistry(t *testing.T) {
	h := newPlaybooksTestHarness(t)
	h.writeRegistry(t, `{"playbooks": []}`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success for empty registry, got error: %v", report.Err)
	}
}

func TestRunPlaybooksPhaseRegistryNotFound(t *testing.T) {
	h := newPlaybooksTestHarness(t)
	// Don't create registry.json

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected error when registry not found")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got: %s", report.FailureClassification)
	}
}

func TestRunPlaybooksPhaseInvalidRegistryJSON(t *testing.T) {
	h := newPlaybooksTestHarness(t)
	h.writeRegistry(t, `{"invalid json`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected error for invalid registry JSON")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got: %s", report.FailureClassification)
	}
}

func TestRunPlaybooksPhaseContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	h := newPlaybooksTestHarness(t)

	report := runPlaybooksPhase(ctx, h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected error when context cancelled")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Errorf("expected system failure, got: %s", report.FailureClassification)
	}
}

func TestConvertPlaybooksObservations(t *testing.T) {
	tests := []struct {
		name     string
		input    playbooks.Observation
		wantText string
	}{
		{
			name:     "success",
			input:    playbooks.NewSuccessObservation("test passed"),
			wantText: "test passed",
		},
		{
			name:     "warning",
			input:    playbooks.NewWarningObservation("test warning"),
			wantText: "test warning",
		},
		{
			name:     "error",
			input:    playbooks.NewErrorObservation("test error"),
			wantText: "test error",
		},
		{
			name:     "info",
			input:    playbooks.NewInfoObservation("test info"),
			wantText: "test info",
		},
		{
			name:     "skip",
			input:    playbooks.NewSkipObservation("test skip"),
			wantText: "test skip",
		},
		// Note: section observations store message in Section field, not Text
		// Tested separately in TestConvertPlaybooksObservationSection
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := ConvertObservationsGeneric([]playbooks.Observation{tc.input}, ExtractStandardObservation[playbooks.Observation])
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			result := results[0]
			if result.Text != tc.wantText {
				t.Errorf("expected text %q, got %q", tc.wantText, result.Text)
			}
		})
	}
}

func TestConvertPlaybooksObservationSection(t *testing.T) {
	input := playbooks.NewSectionObservation("🏗️", "Building phase")
	results := ConvertObservationsGeneric([]playbooks.Observation{input}, ExtractStandardObservation[playbooks.Observation])
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	result := results[0]

	if result.Section != "Building phase" {
		t.Errorf("expected section %q, got %q", "Building phase", result.Section)
	}
	if result.Icon != "🏗️" {
		t.Errorf("expected icon %q, got %q", "🏗️", result.Icon)
	}
}

func TestConvertPlaybooksFailureClass(t *testing.T) {
	tests := []struct {
		input shared.FailureClass
		want  shared.FailureClass
	}{
		{shared.FailureClassMisconfiguration, shared.FailureClassMisconfiguration},
		{shared.FailureClassMissingDependency, shared.FailureClassMissingDependency},
		{shared.FailureClassSystem, shared.FailureClassSystem},
		{shared.FailureClassExecution, shared.FailureClassSystem}, // execution maps to system
	}

	for _, tc := range tests {
		t.Run(string(tc.input), func(t *testing.T) {
			result := shared.StandardizeFailureClass(tc.input)
			if result != tc.want {
				t.Errorf("expected %q, got %q", tc.want, result)
			}
		})
	}
}

func TestResolveScenarioPort(t *testing.T) {
	// This test validates the parsing logic of ResolveScenarioPort
	// The actual vrooli CLI call is mocked in integration tests

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Will cause command to fail

	_, err := ResolveScenarioPort(ctx, io.Discard, "test-scenario", "API_PORT")
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

func TestResolveScenarioBaseURL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ResolveScenarioBaseURL(ctx, io.Discard, "test-scenario")
	if err == nil {
		t.Error("expected error when port resolution fails")
	}
}

func TestStartScenario(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := StartScenario(ctx, "test-scenario", io.Discard)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

// Test helper to verify observations contain expected text
func observationsContain(observations []Observation, substr string) bool {
	for _, obs := range observations {
		if strings.Contains(obs.Text, substr) || strings.Contains(obs.Section, substr) {
			return true
		}
	}
	return false
}

func TestRunPlaybooksPhaseDeprecatedPlaybooksFallback(t *testing.T) {
	h := newPlaybooksTestHarness(t)
	// Use deprecated_playbooks field which should fall back to playbooks
	h.writeRegistry(t, `{"deprecated_playbooks": []}`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	// Should succeed with empty playbooks (from deprecated fallback)
	if report.Err != nil {
		t.Fatalf("expected success for deprecated playbooks fallback, got error: %v", report.Err)
	}
}

// Benchmark tests

func BenchmarkRunPlaybooksPhaseNoUI(b *testing.B) {
	tempDir := b.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenarios", "bench-scenario")
	os.MkdirAll(filepath.Join(scenarioDir, "test"), 0o755)
	// Intentionally don't create ui/ directory

	env := workspace.Environment{
		ScenarioName: "bench-scenario",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
		AppRoot:      tempDir,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runPlaybooksPhase(context.Background(), env, io.Discard)
	}
}

func BenchmarkRunPlaybooksPhaseEmptyRegistry(b *testing.B) {
	tempDir := b.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenarios", "bench-scenario")
	os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0o755)
	playbooksDir := filepath.Join(scenarioDir, "test", "playbooks")
	os.MkdirAll(playbooksDir, 0o755)
	os.WriteFile(filepath.Join(playbooksDir, "registry.json"), []byte(`{"playbooks":[]}`), 0o644)

	env := workspace.Environment{
		ScenarioName: "bench-scenario",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
		AppRoot:      tempDir,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runPlaybooksPhase(context.Background(), env, io.Discard)
	}
}
