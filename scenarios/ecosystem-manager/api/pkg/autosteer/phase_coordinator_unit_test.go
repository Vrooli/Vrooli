package autosteer

import (
	"testing"
)

// MockConditionEvaluator is a mock implementation of ConditionEvaluatorAPI for testing.
type MockConditionEvaluator struct {
	// EvaluateResult is returned by Evaluate calls
	EvaluateResult bool
	// EvaluateError is returned by Evaluate calls if not nil
	EvaluateError error
	// FormatResult is returned by FormatCondition calls
	FormatResult string

	// Call tracking
	EvaluateCalls       int
	FormatConditionCalls int
	LastCondition       StopCondition
	LastMetrics         MetricsSnapshot
}

func (m *MockConditionEvaluator) Evaluate(condition StopCondition, metrics MetricsSnapshot) (bool, error) {
	m.EvaluateCalls++
	m.LastCondition = condition
	m.LastMetrics = metrics
	if m.EvaluateError != nil {
		return false, m.EvaluateError
	}
	return m.EvaluateResult, nil
}

func (m *MockConditionEvaluator) FormatCondition(condition StopCondition, metrics MetricsSnapshot) string {
	m.FormatConditionCalls++
	return m.FormatResult
}

func TestPhaseCoordinator_ShouldAdvancePhase_MaxIterations(t *testing.T) {
	// Arrange
	mockEvaluator := &MockConditionEvaluator{}
	coordinator := NewPhaseCoordinator(mockEvaluator)

	phase := SteerPhase{
		Mode:          ModeProgress,
		MaxIterations: 10,
		StopConditions: []StopCondition{
			{Type: ConditionTypeSimple, Metric: "operational_targets_percentage", CompareOperator: OpGreaterThanEquals, Value: 80},
		},
	}
	metrics := MetricsSnapshot{OperationalTargetsPercentage: 50}

	// Act - iteration at max
	decision := coordinator.ShouldAdvancePhase(phase, metrics, 10)

	// Assert
	if !decision.ShouldStop {
		t.Error("Expected ShouldStop=true when at max iterations")
	}
	if decision.Reason != "max_iterations" {
		t.Errorf("Expected reason='max_iterations', got '%s'", decision.Reason)
	}
	// Should not evaluate conditions when max iterations reached
	if mockEvaluator.EvaluateCalls != 0 {
		t.Errorf("Expected 0 Evaluate calls when max iterations reached, got %d", mockEvaluator.EvaluateCalls)
	}
}

func TestPhaseCoordinator_ShouldAdvancePhase_ConditionMet(t *testing.T) {
	// Arrange
	mockEvaluator := &MockConditionEvaluator{EvaluateResult: true}
	coordinator := NewPhaseCoordinator(mockEvaluator)

	phase := SteerPhase{
		Mode:          ModeProgress,
		MaxIterations: 10,
		StopConditions: []StopCondition{
			{Type: ConditionTypeSimple, Metric: "operational_targets_percentage", CompareOperator: OpGreaterThanEquals, Value: 80},
		},
	}
	metrics := MetricsSnapshot{OperationalTargetsPercentage: 85}

	// Act
	decision := coordinator.ShouldAdvancePhase(phase, metrics, 5)

	// Assert
	if !decision.ShouldStop {
		t.Error("Expected ShouldStop=true when condition is met")
	}
	if decision.Reason != "condition_met" {
		t.Errorf("Expected reason='condition_met', got '%s'", decision.Reason)
	}
	if mockEvaluator.EvaluateCalls != 1 {
		t.Errorf("Expected 1 Evaluate call, got %d", mockEvaluator.EvaluateCalls)
	}
}

func TestPhaseCoordinator_ShouldAdvancePhase_Continue(t *testing.T) {
	// Arrange
	mockEvaluator := &MockConditionEvaluator{EvaluateResult: false}
	coordinator := NewPhaseCoordinator(mockEvaluator)

	phase := SteerPhase{
		Mode:          ModeProgress,
		MaxIterations: 10,
		StopConditions: []StopCondition{
			{Type: ConditionTypeSimple, Metric: "operational_targets_percentage", CompareOperator: OpGreaterThanEquals, Value: 80},
		},
	}
	metrics := MetricsSnapshot{OperationalTargetsPercentage: 50}

	// Act
	decision := coordinator.ShouldAdvancePhase(phase, metrics, 5)

	// Assert
	if decision.ShouldStop {
		t.Error("Expected ShouldStop=false when no conditions met and not at max")
	}
	if decision.Reason != "continue" {
		t.Errorf("Expected reason='continue', got '%s'", decision.Reason)
	}
}

func TestPhaseCoordinator_ShouldAdvancePhase_UnavailableMetric(t *testing.T) {
	// Arrange - evaluator returns metric unavailable error
	mockEvaluator := &MockConditionEvaluator{
		EvaluateError: &MetricUnavailableError{Metric: "test_metric", Reason: "not collected"},
	}
	coordinator := NewPhaseCoordinator(mockEvaluator)

	phase := SteerPhase{
		Mode:          ModeProgress,
		MaxIterations: 10,
		StopConditions: []StopCondition{
			{Type: ConditionTypeSimple, Metric: "test_metric", CompareOperator: OpGreaterThanEquals, Value: 80},
		},
	}
	metrics := MetricsSnapshot{}

	// Act
	decision := coordinator.ShouldAdvancePhase(phase, metrics, 5)

	// Assert - should continue (not fail) when metric unavailable
	if decision.ShouldStop {
		t.Error("Expected ShouldStop=false when metric is unavailable")
	}
	if decision.Reason != "continue" {
		t.Errorf("Expected reason='continue', got '%s'", decision.Reason)
	}
}

func TestPhaseCoordinator_EvaluateQualityGates(t *testing.T) {
	// Arrange
	mockEvaluator := &MockConditionEvaluator{EvaluateResult: true}
	coordinator := NewPhaseCoordinator(mockEvaluator)

	gates := []QualityGate{
		{
			Name:          "build_health",
			Condition:     StopCondition{Type: ConditionTypeSimple, Metric: "build_status", CompareOperator: OpEquals, Value: 1},
			FailureAction: ActionHalt,
			Message:       "Build must be passing",
		},
	}
	metrics := MetricsSnapshot{BuildStatus: 1}

	// Act
	results := coordinator.EvaluateQualityGates(gates, metrics)

	// Assert
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Error("Expected gate to pass")
	}
	if results[0].GateName != "build_health" {
		t.Errorf("Expected gate name 'build_health', got '%s'", results[0].GateName)
	}
}

func TestPhaseCoordinator_DetermineStopReason(t *testing.T) {
	coordinator := NewPhaseCoordinator(&MockConditionEvaluator{})

	tests := []struct {
		current  int
		max      int
		expected string
	}{
		{5, 10, "condition_met"},
		{10, 10, "max_iterations"},
		{15, 10, "max_iterations"},
	}

	for _, tc := range tests {
		reason := coordinator.DetermineStopReason(tc.current, tc.max)
		if reason != tc.expected {
			t.Errorf("DetermineStopReason(%d, %d) = '%s', expected '%s'",
				tc.current, tc.max, reason, tc.expected)
		}
	}
}

func TestPhaseCoordinator_Evaluator(t *testing.T) {
	// Arrange
	mockEvaluator := &MockConditionEvaluator{}
	coordinator := NewPhaseCoordinator(mockEvaluator)

	// Act
	returnedEvaluator := coordinator.Evaluator()

	// Assert
	if returnedEvaluator != mockEvaluator {
		t.Error("Expected Evaluator() to return the injected evaluator")
	}
}
