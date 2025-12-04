package types

import (
	"testing"
	"time"
)

func TestExecutionSummaryTotalWorkflows(t *testing.T) {
	s := ExecutionSummary{WorkflowsExecuted: 5}
	if got := s.TotalWorkflows(); got != 5 {
		t.Errorf("TotalWorkflows() = %d, want 5", got)
	}
}

func TestExecutionSummaryPassRate(t *testing.T) {
	tests := []struct {
		name     string
		summary  ExecutionSummary
		expected float64
	}{
		{
			name:     "all passed",
			summary:  ExecutionSummary{WorkflowsExecuted: 10, WorkflowsPassed: 10},
			expected: 100.0,
		},
		{
			name:     "half passed",
			summary:  ExecutionSummary{WorkflowsExecuted: 10, WorkflowsPassed: 5},
			expected: 50.0,
		},
		{
			name:     "none passed",
			summary:  ExecutionSummary{WorkflowsExecuted: 10, WorkflowsPassed: 0},
			expected: 0.0,
		},
		{
			name:     "no workflows",
			summary:  ExecutionSummary{WorkflowsExecuted: 0},
			expected: 0.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.PassRate(); got != tc.expected {
				t.Errorf("PassRate() = %f, want %f", got, tc.expected)
			}
		})
	}
}

func TestExecutionSummaryString(t *testing.T) {
	tests := []struct {
		name     string
		summary  ExecutionSummary
		contains string
	}{
		{
			name:     "no workflows",
			summary:  ExecutionSummary{},
			contains: "no workflows executed",
		},
		{
			name: "with workflows",
			summary: ExecutionSummary{
				WorkflowsExecuted: 5,
				WorkflowsPassed:   4,
				WorkflowsFailed:   1,
				TotalDuration:     2 * time.Second,
			},
			contains: "4/5 workflows passed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.summary.String()
			if !contains(str, tc.contains) {
				t.Errorf("String() = %q, want to contain %q", str, tc.contains)
			}
		})
	}
}

func TestObservationConstructors(t *testing.T) {
	tests := []struct {
		name       string
		obs        Observation
		expectType ObservationType
		expectMsg  string
	}{
		{"success", NewSuccessObservation("ok"), ObservationSuccess, "ok"},
		{"warning", NewWarningObservation("warn"), ObservationWarning, "warn"},
		{"error", NewErrorObservation("err"), ObservationError, "err"},
		{"info", NewInfoObservation("info"), ObservationInfo, "info"},
		{"skip", NewSkipObservation("skip"), ObservationSkip, "skip"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.obs.Type != tc.expectType {
				t.Errorf("Type = %v, want %v", tc.obs.Type, tc.expectType)
			}
			if tc.obs.Message != tc.expectMsg {
				t.Errorf("Message = %q, want %q", tc.obs.Message, tc.expectMsg)
			}
		})
	}
}

func TestNewSectionObservation(t *testing.T) {
	obs := NewSectionObservation("🔍", "Checking files")
	if obs.Type != ObservationSection {
		t.Errorf("Type = %v, want ObservationSection", obs.Type)
	}
	if obs.Icon != "🔍" {
		t.Errorf("Icon = %q, want %q", obs.Icon, "🔍")
	}
	if obs.Message != "Checking files" {
		t.Errorf("Message = %q, want %q", obs.Message, "Checking files")
	}
}

func TestResultBuilders(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		result := OK()
		if !result.Success {
			t.Error("OK() should return success=true")
		}
	})

	t.Run("OKWithResults", func(t *testing.T) {
		results := []Result{{Entry: Entry{File: "test.json"}}}
		result := OKWithResults(results)
		if !result.Success {
			t.Error("OKWithResults() should return success=true")
		}
		if len(result.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(result.Results))
		}
	})

	t.Run("FailMisconfiguration", func(t *testing.T) {
		result := FailMisconfiguration(nil, "fix it")
		if result.Success {
			t.Error("FailMisconfiguration() should return success=false")
		}
		if result.FailureClass != FailureClassMisconfiguration {
			t.Errorf("expected misconfiguration, got %s", result.FailureClass)
		}
	})

	t.Run("FailMissingDependency", func(t *testing.T) {
		result := FailMissingDependency(nil, "install it")
		if result.FailureClass != FailureClassMissingDependency {
			t.Errorf("expected missing_dependency, got %s", result.FailureClass)
		}
	})

	t.Run("FailSystem", func(t *testing.T) {
		result := FailSystem(nil, "check logs")
		if result.FailureClass != FailureClassSystem {
			t.Errorf("expected system, got %s", result.FailureClass)
		}
	})

	t.Run("FailExecution", func(t *testing.T) {
		result := FailExecution(nil, "debug workflow")
		if result.FailureClass != FailureClassExecution {
			t.Errorf("expected execution, got %s", result.FailureClass)
		}
	})
}

func TestRunResultWithObservations(t *testing.T) {
	result := OK().WithObservations(
		NewSuccessObservation("first"),
		NewInfoObservation("second"),
	)
	if len(result.Observations) != 2 {
		t.Errorf("expected 2 observations, got %d", len(result.Observations))
	}
}

func TestRunResultWithResults(t *testing.T) {
	results := []Result{
		{Entry: Entry{File: "a.json"}},
		{Entry: Entry{File: "b.json"}},
	}
	result := OK().WithResults(results)
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
