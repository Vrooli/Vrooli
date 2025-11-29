package errors

import (
	"errors"
	"testing"
	"time"
)

// [REQ:SCS-CORE-003] Test structured error types
func TestCollectorError(t *testing.T) {
	cause := errors.New("connection refused")
	err := NewCollectorError("requirements", "failed to load data", cause)

	if err.Collector != "requirements" {
		t.Errorf("Expected collector 'requirements', got '%s'", err.Collector)
	}
	if err.Severity != SeverityError {
		t.Errorf("Expected severity 'error', got '%s'", err.Severity)
	}
	if !err.Recoverable {
		t.Error("Expected error to be recoverable")
	}
	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error string should not be empty")
	}
}

// [REQ:SCS-CORE-003] Test collector warning
func TestCollectorWarning(t *testing.T) {
	err := NewCollectorWarning("ui", "partial data available")

	if err.Severity != SeverityWarning {
		t.Errorf("Expected severity 'warning', got '%s'", err.Severity)
	}
	if !err.PartialData {
		t.Error("Warning should indicate partial data")
	}
	if !err.Recoverable {
		t.Error("Warning should be recoverable")
	}
}

// [REQ:SCS-CORE-003] Test scoring error with collector errors
func TestScoringError(t *testing.T) {
	collErr1 := NewCollectorError("requirements", "failed", nil)
	collErr2 := NewCollectorError("tests", "timeout", nil)

	err := NewScoringError("my-scenario", "collection failed", CategoryCollector, nil).
		WithPartialResults().
		WithCollectorErrors(collErr1, collErr2)

	if err.Scenario != "my-scenario" {
		t.Errorf("Expected scenario 'my-scenario', got '%s'", err.Scenario)
	}
	if !err.PartialResults {
		t.Error("Expected partial results flag to be set")
	}
	if len(err.CollectorErrs) != 2 {
		t.Errorf("Expected 2 collector errors, got %d", len(err.CollectorErrs))
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error string should not be empty")
	}
}

// [REQ:SCS-CORE-004] Test partial result tracking
func TestPartialResult(t *testing.T) {
	pr := NewPartialResult()

	if !pr.IsComplete {
		t.Error("New partial result should be complete")
	}
	if pr.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", pr.Confidence)
	}

	// Mark some collectors as available
	pr.MarkAvailable("requirements")
	pr.MarkAvailable("service")

	if !pr.Available["requirements"] {
		t.Error("Requirements should be marked available")
	}

	// Mark a collector as failed
	pr.MarkFailed("ui", NewCollectorWarning("ui", "circuit breaker open"))

	if pr.IsComplete {
		t.Error("Partial result should not be complete after failure")
	}
	if pr.Confidence >= 1.0 {
		t.Error("Confidence should be reduced after failure")
	}
	if len(pr.Missing) != 1 || pr.Missing[0] != "ui" {
		t.Errorf("Expected 'ui' in missing, got %v", pr.Missing)
	}

	msg := pr.GetMessage()
	if msg == "" {
		t.Error("Message should not be empty for partial results")
	}
}

// [REQ:SCS-CORE-004] Test partial result confidence calculation
func TestPartialResultConfidence(t *testing.T) {
	testCases := []struct {
		name       string
		collectors []string
		minConf    float64
		maxConf    float64
	}{
		{"requirements fail", []string{"requirements"}, 0.5, 0.8},
		{"tests fail", []string{"tests"}, 0.5, 0.8},
		{"ui fail", []string{"ui"}, 0.7, 0.9},
		{"service fail", []string{"service"}, 0.9, 1.0},
		{"multiple fail", []string{"requirements", "tests"}, 0.3, 0.6},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := NewPartialResult()
			for _, c := range tc.collectors {
				pr.MarkFailed(c, nil)
			}
			if pr.Confidence < tc.minConf || pr.Confidence > tc.maxConf {
				t.Errorf("Expected confidence in [%f, %f], got %f",
					tc.minConf, tc.maxConf, pr.Confidence)
			}
		})
	}
}

// [REQ:SCS-CORE-003] Test API error with next steps
func TestAPIError(t *testing.T) {
	err := NewAPIError(ErrCodeScenarioNotFound, "Scenario not found", CategoryValidation).
		WithDetails("Scenario 'foo' does not exist").
		WithNextSteps("Check the name", "List available scenarios").
		AsRecoverable()

	if err.Code != ErrCodeScenarioNotFound {
		t.Errorf("Expected code '%s', got '%s'", ErrCodeScenarioNotFound, err.Code)
	}
	if err.Details == "" {
		t.Error("Details should not be empty")
	}
	if len(err.NextSteps) != 2 {
		t.Errorf("Expected 2 next steps, got %d", len(err.NextSteps))
	}
	if !err.Recoverable {
		t.Error("Error should be recoverable")
	}
	if err.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

// [REQ:SCS-CORE-003] Test pre-defined errors
func TestPredefinedErrors(t *testing.T) {
	// Test that pre-defined errors have useful next steps
	if len(ErrScenarioNotFound.NextSteps) == 0 {
		t.Error("ErrScenarioNotFound should have next steps")
	}
	if len(ErrDatabaseUnavailable.NextSteps) == 0 {
		t.Error("ErrDatabaseUnavailable should have next steps")
	}
	if len(ErrCollectorCircuitOpen.NextSteps) == 0 {
		t.Error("ErrCollectorCircuitOpen should have next steps")
	}

	// Test recovery flags
	if !ErrDatabaseUnavailable.Recoverable {
		t.Error("Database unavailable should be recoverable")
	}
	if !ErrCollectorCircuitOpen.Recoverable {
		t.Error("Circuit open should be recoverable")
	}
}

// [REQ:SCS-CORE-003] Test severity levels
func TestSeverityLevels(t *testing.T) {
	severities := []Severity{SeverityInfo, SeverityWarning, SeverityError, SeverityCritical}
	for _, s := range severities {
		if s == "" {
			t.Errorf("Severity should not be empty")
		}
	}
}

// [REQ:SCS-CORE-003] Test category types
func TestCategoryTypes(t *testing.T) {
	categories := []Category{
		CategoryCollector,
		CategoryDatabase,
		CategoryConfig,
		CategoryValidation,
		CategoryNetwork,
		CategoryFileSystem,
		CategoryInternal,
	}
	for _, c := range categories {
		if c == "" {
			t.Errorf("Category should not be empty")
		}
	}
}

// [REQ:SCS-CORE-003] Test collector error timestamp
func TestCollectorErrorTimestamp(t *testing.T) {
	before := time.Now()
	err := NewCollectorError("test", "test error", nil)
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Timestamp should be within test bounds")
	}
}
