package business

import (
	"strings"
	"testing"
)

// =============================================================================
// ValidationSummary Tests
// =============================================================================

func TestValidationSummary_TotalChecks_AllCategories(t *testing.T) {
	summary := ValidationSummary{
		ModulesFound:      3,
		RequirementsFound: 15,
		ValidationErrors:  2,
		ValidationWarns:   5,
	}

	// Errors and warnings don't count as checks
	expected := 18 // 3 + 15
	if got := summary.TotalChecks(); got != expected {
		t.Errorf("TotalChecks() = %d, want %d", got, expected)
	}
}

func TestValidationSummary_TotalChecks_Empty(t *testing.T) {
	summary := ValidationSummary{}
	if got := summary.TotalChecks(); got != 0 {
		t.Errorf("TotalChecks() = %d, want 0", got)
	}
}

func TestValidationSummary_TotalChecks_OnlyModules(t *testing.T) {
	summary := ValidationSummary{ModulesFound: 5}
	if got := summary.TotalChecks(); got != 5 {
		t.Errorf("TotalChecks() = %d, want 5", got)
	}
}

func TestValidationSummary_TotalChecks_OnlyRequirements(t *testing.T) {
	summary := ValidationSummary{RequirementsFound: 10}
	if got := summary.TotalChecks(); got != 10 {
		t.Errorf("TotalChecks() = %d, want 10", got)
	}
}

func TestValidationSummary_TotalChecks_ErrorsAndWarningsNotCounted(t *testing.T) {
	summary := ValidationSummary{
		ModulesFound:      0,
		RequirementsFound: 0,
		ValidationErrors:  10,
		ValidationWarns:   20,
	}

	// Errors and warnings should NOT contribute to TotalChecks
	if got := summary.TotalChecks(); got != 0 {
		t.Errorf("TotalChecks() = %d, want 0 (errors/warnings should not count)", got)
	}
}

func TestValidationSummary_String_ContainsAllFields(t *testing.T) {
	summary := ValidationSummary{
		ModulesFound:      2,
		RequirementsFound: 5,
		ValidationErrors:  1,
		ValidationWarns:   3,
	}

	str := summary.String()

	// Verify all counts are present
	expectations := []string{"2", "5", "1", "3", "modules", "requirements", "errors", "warnings"}
	for _, exp := range expectations {
		if !strings.Contains(str, exp) {
			t.Errorf("String() missing expected substring %q in %q", exp, str)
		}
	}
}

func TestValidationSummary_String_ZeroValues(t *testing.T) {
	summary := ValidationSummary{}
	str := summary.String()

	if str == "" {
		t.Error("String() should not be empty for zero-value summary")
	}

	// Should contain zeros
	if !strings.Contains(str, "0") {
		t.Error("String() should contain zero values")
	}
}

func TestValidationSummary_String_Format(t *testing.T) {
	summary := ValidationSummary{
		ModulesFound:      2,
		RequirementsFound: 5,
		ValidationErrors:  1,
		ValidationWarns:   3,
	}

	str := summary.String()
	expected := "2 modules, 5 requirements, 1 errors, 3 warnings"

	if str != expected {
		t.Errorf("String() = %q, want %q", str, expected)
	}
}

// =============================================================================
// RunResult Tests
// =============================================================================

func TestRunResult_SuccessState(t *testing.T) {
	result := RunResult{
		Success: true,
		Summary: ValidationSummary{
			ModulesFound:      2,
			RequirementsFound: 10,
		},
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Error != nil {
		t.Error("expected Error to be nil for success")
	}
	if result.FailureClass != FailureClassNone {
		t.Errorf("expected FailureClass to be None, got %s", result.FailureClass)
	}
}

func TestRunResult_FailureState(t *testing.T) {
	result := RunResult{
		Success:      false,
		Error:        errTestError,
		FailureClass: FailureClassMisconfiguration,
		Remediation:  "Fix the config",
	}

	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Error == nil {
		t.Error("expected Error to be non-nil for failure")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected FailureClass to be Misconfiguration, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected Remediation to be non-empty")
	}
}

func TestRunResult_WithObservations(t *testing.T) {
	result := RunResult{
		Success: true,
		Observations: []Observation{
			NewSuccessObservation("passed test 1"),
			NewSuccessObservation("passed test 2"),
			NewWarningObservation("minor issue"),
		},
	}

	if len(result.Observations) != 3 {
		t.Errorf("expected 3 observations, got %d", len(result.Observations))
	}
}

// =============================================================================
// Type Re-export Tests
// =============================================================================

func TestTypeReexports_FailureClassConstants(t *testing.T) {
	// Verify constants are accessible
	if FailureClassNone != "" {
		t.Errorf("expected FailureClassNone to be empty string, got %q", FailureClassNone)
	}
	if FailureClassMisconfiguration == "" {
		t.Error("expected FailureClassMisconfiguration to be non-empty")
	}
	if FailureClassSystem == "" {
		t.Error("expected FailureClassSystem to be non-empty")
	}
}

func TestTypeReexports_ObservationTypeConstants(t *testing.T) {
	// Verify observation type constants are distinct
	types := []ObservationType{
		ObservationSection,
		ObservationSuccess,
		ObservationWarning,
		ObservationError,
		ObservationInfo,
		ObservationSkip,
	}

	seen := make(map[ObservationType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("duplicate observation type: %v", typ)
		}
		seen[typ] = true
	}
}

func TestTypeReexports_ConstructorFunctions(t *testing.T) {
	// Verify constructor functions work
	obs := NewSuccessObservation("test")
	if obs.Type != ObservationSuccess {
		t.Errorf("NewSuccessObservation created wrong type: %v", obs.Type)
	}

	obs = NewWarningObservation("test")
	if obs.Type != ObservationWarning {
		t.Errorf("NewWarningObservation created wrong type: %v", obs.Type)
	}

	obs = NewErrorObservation("test")
	if obs.Type != ObservationError {
		t.Errorf("NewErrorObservation created wrong type: %v", obs.Type)
	}

	obs = NewInfoObservation("test")
	if obs.Type != ObservationInfo {
		t.Errorf("NewInfoObservation created wrong type: %v", obs.Type)
	}

	obs = NewSkipObservation("test")
	if obs.Type != ObservationSkip {
		t.Errorf("NewSkipObservation created wrong type: %v", obs.Type)
	}
}

func TestTypeReexports_ResultConstructors(t *testing.T) {
	// OK
	result := OK()
	if !result.Success {
		t.Error("OK() should return successful result")
	}

	// OKWithCount
	result = OKWithCount(5)
	if !result.Success || result.ItemsChecked != 5 {
		t.Error("OKWithCount() should set ItemsChecked")
	}

	// FailMisconfiguration
	result = FailMisconfiguration(errTestError, "fix it")
	if result.Success {
		t.Error("FailMisconfiguration() should not be successful")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Error("FailMisconfiguration() should set correct failure class")
	}

	// FailSystem
	result = FailSystem(errTestError, "system error")
	if result.Success {
		t.Error("FailSystem() should not be successful")
	}
	if result.FailureClass != FailureClassSystem {
		t.Error("FailSystem() should set correct failure class")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkValidationSummary_TotalChecks(b *testing.B) {
	summary := ValidationSummary{
		ModulesFound:      10,
		RequirementsFound: 100,
		ValidationErrors:  5,
		ValidationWarns:   20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = summary.TotalChecks()
	}
}

func BenchmarkValidationSummary_String(b *testing.B) {
	summary := ValidationSummary{
		ModulesFound:      10,
		RequirementsFound: 100,
		ValidationErrors:  5,
		ValidationWarns:   20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = summary.String()
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

// errTestError is a sentinel error for testing
var errTestError = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
