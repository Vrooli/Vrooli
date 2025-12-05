package lint

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"test-genie/internal/shared"
)

func TestRunner_Run_NoLanguagesDetected(t *testing.T) {
	// Create runner with a non-existent scenario dir
	config := Config{
		ScenarioDir:  "/nonexistent",
		ScenarioName: "test-scenario",
	}

	var buf bytes.Buffer
	runner := New(config, WithLogger(&buf))

	result := runner.Run(context.Background())

	if !result.Success {
		t.Error("expected success when no languages detected")
	}

	if len(result.Observations) == 0 {
		t.Error("expected at least one observation")
	}
}

func TestRunner_Run_ContextCancelled(t *testing.T) {
	config := Config{
		ScenarioDir:  "/tmp",
		ScenarioName: "test-scenario",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := New(config)
	result := runner.Run(ctx)

	if result.Success {
		t.Error("expected failure when context is cancelled")
	}

	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected FailureClassSystem, got %v", result.FailureClass)
	}

	if !errors.Is(result.Error, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", result.Error)
	}
}

func TestLintSummary_TotalChecks(t *testing.T) {
	tests := []struct {
		name     string
		summary  LintSummary
		expected int
	}{
		{
			name:     "no languages checked",
			summary:  LintSummary{},
			expected: 0,
		},
		{
			name:     "only go checked",
			summary:  LintSummary{GoChecked: true},
			expected: 1,
		},
		{
			name:     "all languages checked",
			summary:  LintSummary{GoChecked: true, NodeChecked: true, PythonChecked: true},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.TotalChecks(); got != tt.expected {
				t.Errorf("TotalChecks() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLintSummary_TotalIssues(t *testing.T) {
	summary := LintSummary{
		GoIssues:     5,
		NodeIssues:   3,
		PythonIssues: 2,
	}

	if got := summary.TotalIssues(); got != 10 {
		t.Errorf("TotalIssues() = %v, want 10", got)
	}
}

func TestLintSummary_HasTypeErrors(t *testing.T) {
	tests := []struct {
		name     string
		summary  LintSummary
		expected bool
	}{
		{
			name:     "no type errors",
			summary:  LintSummary{TypeErrors: 0},
			expected: false,
		},
		{
			name:     "has type errors",
			summary:  LintSummary{TypeErrors: 3},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.HasTypeErrors(); got != tt.expected {
				t.Errorf("HasTypeErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLintSummary_String(t *testing.T) {
	tests := []struct {
		name     string
		summary  LintSummary
		contains string
	}{
		{
			name:     "no languages",
			summary:  LintSummary{},
			contains: "no languages checked",
		},
		{
			name:     "go only",
			summary:  LintSummary{GoChecked: true, GoIssues: 2},
			contains: "Go: 2 issues",
		},
		{
			name: "multiple languages",
			summary: LintSummary{
				GoChecked:     true,
				NodeChecked:   true,
				GoIssues:      1,
				NodeIssues:    3,
			},
			contains: "Go: 1 issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.String()
			if !bytes.Contains([]byte(got), []byte(tt.contains)) {
				t.Errorf("String() = %v, want to contain %v", got, tt.contains)
			}
		})
	}
}

func TestNewObservations(t *testing.T) {
	// Test section observation
	section := NewSectionObservation("icon", "Section message")
	if section.Type != shared.ObservationSection {
		t.Errorf("expected ObservationSection, got %v", section.Type)
	}
	if section.Icon != "icon" {
		t.Errorf("expected icon 'icon', got %v", section.Icon)
	}

	// Test success observation
	success := NewSuccessObservation("Success message")
	if success.Type != shared.ObservationSuccess {
		t.Errorf("expected ObservationSuccess, got %v", success.Type)
	}

	// Test warning observation
	warning := NewWarningObservation("Warning message")
	if warning.Type != shared.ObservationWarning {
		t.Errorf("expected ObservationWarning, got %v", warning.Type)
	}

	// Test error observation
	errorObs := NewErrorObservation("Error message")
	if errorObs.Type != shared.ObservationError {
		t.Errorf("expected ObservationError, got %v", errorObs.Type)
	}

	// Test info observation
	info := NewInfoObservation("Info message")
	if info.Type != shared.ObservationInfo {
		t.Errorf("expected ObservationInfo, got %v", info.Type)
	}
}
