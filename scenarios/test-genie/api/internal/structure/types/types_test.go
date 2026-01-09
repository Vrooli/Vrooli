package types

import (
	"errors"
	"testing"
)

// =============================================================================
// Observation Tests
// =============================================================================

func TestNewSectionObservation(t *testing.T) {
	obs := NewSectionObservation("🔍", "Checking files...")

	if obs.Type != ObservationSection {
		t.Errorf("expected type ObservationSection, got %v", obs.Type)
	}
	if obs.Icon != "🔍" {
		t.Errorf("expected icon '🔍', got %s", obs.Icon)
	}
	if obs.Message != "Checking files..." {
		t.Errorf("expected message 'Checking files...', got %s", obs.Message)
	}
}

func TestNewSuccessObservation(t *testing.T) {
	obs := NewSuccessObservation("All checks passed")

	if obs.Type != ObservationSuccess {
		t.Errorf("expected type ObservationSuccess, got %v", obs.Type)
	}
	if obs.Message != "All checks passed" {
		t.Errorf("expected message 'All checks passed', got %s", obs.Message)
	}
	// Success observations don't set icon by default
	if obs.Icon != "" {
		t.Errorf("expected empty icon, got %s", obs.Icon)
	}
}

func TestNewWarningObservation(t *testing.T) {
	obs := NewWarningObservation("Optional file missing")

	if obs.Type != ObservationWarning {
		t.Errorf("expected type ObservationWarning, got %v", obs.Type)
	}
	if obs.Message != "Optional file missing" {
		t.Errorf("expected message 'Optional file missing', got %s", obs.Message)
	}
}

func TestNewErrorObservation(t *testing.T) {
	obs := NewErrorObservation("Required file not found")

	if obs.Type != ObservationError {
		t.Errorf("expected type ObservationError, got %v", obs.Type)
	}
	if obs.Message != "Required file not found" {
		t.Errorf("expected message 'Required file not found', got %s", obs.Message)
	}
}

func TestNewInfoObservation(t *testing.T) {
	obs := NewInfoObservation("Processing 10 files")

	if obs.Type != ObservationInfo {
		t.Errorf("expected type ObservationInfo, got %v", obs.Type)
	}
	if obs.Message != "Processing 10 files" {
		t.Errorf("expected message 'Processing 10 files', got %s", obs.Message)
	}
}

func TestNewSkipObservation(t *testing.T) {
	obs := NewSkipObservation("Smoke test disabled")

	if obs.Type != ObservationSkip {
		t.Errorf("expected type ObservationSkip, got %v", obs.Type)
	}
	if obs.Message != "Smoke test disabled" {
		t.Errorf("expected message 'Smoke test disabled', got %s", obs.Message)
	}
}

// =============================================================================
// FailureClass Tests
// =============================================================================

func TestFailureClassConstants(t *testing.T) {
	if FailureClassNone != "" {
		t.Errorf("expected FailureClassNone to be empty string, got %s", FailureClassNone)
	}
	if FailureClassMisconfiguration != "misconfiguration" {
		t.Errorf("expected FailureClassMisconfiguration='misconfiguration', got %s", FailureClassMisconfiguration)
	}
	if FailureClassSystem != "system" {
		t.Errorf("expected FailureClassSystem='system', got %s", FailureClassSystem)
	}
}

// =============================================================================
// Result Tests
// =============================================================================

func TestOK(t *testing.T) {
	result := OK()

	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if result.FailureClass != FailureClassNone {
		t.Errorf("expected FailureClass=none, got %s", result.FailureClass)
	}
	if result.Remediation != "" {
		t.Errorf("expected empty remediation, got %s", result.Remediation)
	}
	if len(result.Observations) != 0 {
		t.Errorf("expected empty observations, got %v", result.Observations)
	}
	if result.ItemsChecked != 0 {
		t.Errorf("expected ItemsChecked=0, got %d", result.ItemsChecked)
	}
}

func TestOKWithCount(t *testing.T) {
	result := OKWithCount(42)

	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.ItemsChecked != 42 {
		t.Errorf("expected ItemsChecked=42, got %d", result.ItemsChecked)
	}
}

func TestFail(t *testing.T) {
	err := errors.New("something went wrong")
	result := Fail(err, FailureClassSystem, "Restart the service")

	if result.Success {
		t.Error("expected Success=false")
	}
	if result.Error != err {
		t.Errorf("expected error=%v, got %v", err, result.Error)
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected FailureClass=system, got %s", result.FailureClass)
	}
	if result.Remediation != "Restart the service" {
		t.Errorf("expected remediation='Restart the service', got %s", result.Remediation)
	}
}

func TestFailMisconfiguration(t *testing.T) {
	err := errors.New("invalid config")
	result := FailMisconfiguration(err, "Fix the configuration file")

	if result.Success {
		t.Error("expected Success=false")
	}
	if result.Error != err {
		t.Errorf("expected error=%v, got %v", err, result.Error)
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected FailureClass=misconfiguration, got %s", result.FailureClass)
	}
	if result.Remediation != "Fix the configuration file" {
		t.Errorf("expected remediation='Fix the configuration file', got %s", result.Remediation)
	}
}

func TestFailSystem(t *testing.T) {
	err := errors.New("disk full")
	result := FailSystem(err, "Free up disk space")

	if result.Success {
		t.Error("expected Success=false")
	}
	if result.Error != err {
		t.Errorf("expected error=%v, got %v", err, result.Error)
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected FailureClass=system, got %s", result.FailureClass)
	}
	if result.Remediation != "Free up disk space" {
		t.Errorf("expected remediation='Free up disk space', got %s", result.Remediation)
	}
}

func TestResult_WithObservations(t *testing.T) {
	result := OK()
	obs1 := NewSuccessObservation("Check 1 passed")
	obs2 := NewSuccessObservation("Check 2 passed")

	newResult := result.WithObservations(obs1, obs2)

	// Original should be unchanged (value receiver copies)
	if len(result.Observations) != 0 {
		t.Error("expected original result to be unchanged")
	}

	// New result should have observations
	if len(newResult.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(newResult.Observations))
	}
	if newResult.Observations[0].Message != "Check 1 passed" {
		t.Errorf("expected first observation message 'Check 1 passed', got %s", newResult.Observations[0].Message)
	}
	if newResult.Observations[1].Message != "Check 2 passed" {
		t.Errorf("expected second observation message 'Check 2 passed', got %s", newResult.Observations[1].Message)
	}
}

func TestResult_WithObservationsChaining(t *testing.T) {
	result := OK().
		WithObservations(NewSuccessObservation("First")).
		WithObservations(NewSuccessObservation("Second"))

	if len(result.Observations) != 2 {
		t.Fatalf("expected 2 observations after chaining, got %d", len(result.Observations))
	}
}

func TestResult_WithObservationsEmpty(t *testing.T) {
	result := OK().WithObservations()

	if len(result.Observations) != 0 {
		t.Errorf("expected 0 observations when adding none, got %d", len(result.Observations))
	}
}

func TestFailWithObservations(t *testing.T) {
	err := errors.New("validation failed")
	result := FailMisconfiguration(err, "Fix issues").
		WithObservations(
			NewErrorObservation("Missing file A"),
			NewErrorObservation("Missing file B"),
		)

	if result.Success {
		t.Error("expected Success=false")
	}
	if len(result.Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(result.Observations))
	}
	if result.Observations[0].Type != ObservationError {
		t.Errorf("expected error observation type, got %v", result.Observations[0].Type)
	}
}

// =============================================================================
// ObservationType Tests
// =============================================================================

func TestObservationTypeValues(t *testing.T) {
	// Verify iota ordering is preserved
	tests := []struct {
		obsType  ObservationType
		expected int
	}{
		{ObservationSection, 0},
		{ObservationSuccess, 1},
		{ObservationWarning, 2},
		{ObservationError, 3},
		{ObservationInfo, 4},
		{ObservationSkip, 5},
	}

	for _, tc := range tests {
		if int(tc.obsType) != tc.expected {
			t.Errorf("expected %d, got %d", tc.expected, int(tc.obsType))
		}
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestOKWithCountZero(t *testing.T) {
	result := OKWithCount(0)

	if !result.Success {
		t.Error("expected Success=true even with zero count")
	}
	if result.ItemsChecked != 0 {
		t.Errorf("expected ItemsChecked=0, got %d", result.ItemsChecked)
	}
}

func TestFailWithNilError(t *testing.T) {
	result := Fail(nil, FailureClassSystem, "Something went wrong")

	if result.Success {
		t.Error("expected Success=false even with nil error")
	}
	if result.Error != nil {
		t.Errorf("expected nil error, got %v", result.Error)
	}
}

func TestEmptyMessages(t *testing.T) {
	obs := NewSuccessObservation("")
	if obs.Message != "" {
		t.Errorf("expected empty message, got %s", obs.Message)
	}

	result := FailMisconfiguration(errors.New("err"), "")
	if result.Remediation != "" {
		t.Errorf("expected empty remediation, got %s", result.Remediation)
	}
}
