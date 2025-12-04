package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/performance"
)

func TestRunPerformancePhaseValidatesBuilds(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Create api/ directory with go.mod for Go build detection
	apiDir := filepath.Join(scenarioDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	goMod := `module test-demo

go 1.21
`
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	// Create a minimal main.go for the build
	mainGo := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}

	// Note: This test requires Go to be installed on the system
	// It serves as an integration test for the full phase
	report := runPerformancePhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		// If Go is not installed, skip this test
		t.Skipf("performance phase failed (Go may not be installed): %v", report.Err)
	}
	if len(report.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestRunPerformancePhaseHandlesContextCancellation(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}

	report := runPerformancePhase(ctx, env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected error when context cancelled")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Fatalf("expected system failure classification, got %s", report.FailureClassification)
	}
}

func TestRunPerformancePhaseHandlesInvalidTestingJSON(t *testing.T) {
	root := t.TempDir()
	scenarioDir := createScenarioLayout(t, root, "demo")

	// Write invalid testing.json
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid testing.json: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
	}

	report := runPerformancePhase(context.Background(), env, io.Discard)
	if report.Err == nil {
		t.Fatalf("expected error for invalid testing.json")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration failure, got %s", report.FailureClassification)
	}
}

// =============================================================================
// Observation Conversion Tests
// =============================================================================

func TestConvertPerformanceObservations(t *testing.T) {
	obs := []performance.Observation{
		{Type: performance.ObservationSection, Icon: "🔨", Message: "section"},
		{Type: performance.ObservationSuccess, Message: "success"},
		{Type: performance.ObservationWarning, Message: "warning"},
		{Type: performance.ObservationError, Message: "error"},
		{Type: performance.ObservationInfo, Message: "info"},
		{Type: performance.ObservationSkip, Message: "skip"},
	}

	result := convertPerformanceObservations(obs)

	if len(result) != len(obs) {
		t.Fatalf("expected %d observations, got %d", len(obs), len(result))
	}

	// Check that each type was converted correctly
	if result[0].Icon != "🔨" || result[0].Section != "section" {
		t.Errorf("section observation not converted correctly: %+v", result[0])
	}
	if result[1].Text != "success" {
		t.Errorf("success observation not converted correctly: %+v", result[1])
	}
}

func TestConvertPerformanceObservation_AllTypes(t *testing.T) {
	tests := []struct {
		name        string
		input       performance.Observation
		wantText    string
		wantSection string
	}{
		{
			name:        "section",
			input:       performance.Observation{Type: performance.ObservationSection, Icon: "📋", Message: "test section"},
			wantSection: "test section",
		},
		{
			name:     "success",
			input:    performance.Observation{Type: performance.ObservationSuccess, Message: "test success"},
			wantText: "test success",
		},
		{
			name:     "warning",
			input:    performance.Observation{Type: performance.ObservationWarning, Message: "test warning"},
			wantText: "test warning",
		},
		{
			name:     "error",
			input:    performance.Observation{Type: performance.ObservationError, Message: "test error"},
			wantText: "test error",
		},
		{
			name:     "info",
			input:    performance.Observation{Type: performance.ObservationInfo, Message: "test info"},
			wantText: "test info",
		},
		{
			name:     "skip",
			input:    performance.Observation{Type: performance.ObservationSkip, Message: "test skip"},
			wantText: "test skip",
		},
		{
			name:     "unknown",
			input:    performance.Observation{Type: 99, Message: "unknown type"},
			wantText: "unknown type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := convertPerformanceObservation(tc.input)
			if tc.wantSection != "" {
				if result.Section != tc.wantSection {
					t.Errorf("expected Section=%q, got %q", tc.wantSection, result.Section)
				}
			} else if result.Text != tc.wantText {
				t.Errorf("expected Text=%q, got %q", tc.wantText, result.Text)
			}
		})
	}
}

func TestConvertPerformanceFailureClass(t *testing.T) {
	tests := []struct {
		input    performance.FailureClass
		expected string
	}{
		{performance.FailureClassMisconfiguration, FailureClassMisconfiguration},
		{performance.FailureClassMissingDependency, FailureClassMissingDependency},
		{performance.FailureClassSystem, FailureClassSystem},
		{performance.FailureClassNone, FailureClassSystem}, // default
		{"unknown", FailureClassSystem},                    // default
	}

	for _, tc := range tests {
		t.Run(string(tc.input), func(t *testing.T) {
			result := convertPerformanceFailureClass(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
