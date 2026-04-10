package smoketest

import (
	"testing"
)

func TestErrorKind_String(t *testing.T) {
	tests := []struct {
		kind ErrorKind
		want string
	}{
		{ErrKindArtifact, "artifact"},
		{ErrKindExecution, "execution"},
		{ErrKindTimeout, "timeout"},
		{ErrKindValidation, "validation"},
		{ErrKindTelemetry, "telemetry"},
		{ErrKindPlatform, "platform"},
		{ErrKindStore, "store"},
		{ErrKindCancelled, "cancelled"},
		{ErrorKind(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("ErrorKind.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		err := NewArtifactError("file not found", &testError{msg: "no such file"}, "/path/to/file")
		got := err.Error()
		if got != "file not found: no such file" {
			t.Errorf("Error() = %q, want %q", got, "file not found: no such file")
		}
	})

	t.Run("without cause", func(t *testing.T) {
		err := NewValidationError("missing marker", nil)
		got := err.Error()
		if got != "missing marker" {
			t.Errorf("Error() = %q, want %q", got, "missing marker")
		}
	})
}

func TestError_Unwrap(t *testing.T) {
	cause := &testError{msg: "root cause"}
	err := NewExecutionError("wrapper", cause, nil)

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

// Test that each error kind has appropriate recovery info
func TestNewArtifactError_HasRecoveryInfo(t *testing.T) {
	err := NewArtifactError("test error", nil, "/path/to/artifact")

	if err.Kind != ErrKindArtifact {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindArtifact)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if err.Recoverable {
		t.Error("Artifact errors should not be recoverable")
	}
	if err.RetryStrategy != nil {
		t.Error("Artifact errors should not have retry strategy")
	}
}

func TestNewExecutionError_HasRecoveryInfo(t *testing.T) {
	err := NewExecutionError("test error", nil, nil)

	if err.Kind != ErrKindExecution {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindExecution)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if !err.Recoverable {
		t.Error("Execution errors should be recoverable")
	}
	if err.RetryStrategy == nil {
		t.Error("Execution errors should have retry strategy")
	}
	if err.RetryStrategy.MaxAttempts <= 0 {
		t.Error("RetryStrategy.MaxAttempts should be positive")
	}
}

func TestNewTimeoutError_HasRecoveryInfo(t *testing.T) {
	err := NewTimeoutError("test error", nil, nil)

	if err.Kind != ErrKindTimeout {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindTimeout)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if !err.Recoverable {
		t.Error("Timeout errors should be recoverable")
	}
	if err.RetryStrategy == nil {
		t.Error("Timeout errors should have retry strategy")
	}
}

func TestNewValidationError_HasRecoveryInfo(t *testing.T) {
	err := NewValidationError("test error", nil)

	if err.Kind != ErrKindValidation {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindValidation)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if err.Recoverable {
		t.Error("Validation errors should not be recoverable")
	}
}

func TestNewTelemetryError_HasRecoveryInfo(t *testing.T) {
	err := NewTelemetryError("test error", nil, nil)

	if err.Kind != ErrKindTelemetry {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindTelemetry)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if !err.Recoverable {
		t.Error("Telemetry errors should be recoverable")
	}
	if err.RetryStrategy == nil {
		t.Error("Telemetry errors should have retry strategy")
	}
}

func TestNewPlatformError_HasRecoveryInfo_Linux(t *testing.T) {
	err := NewPlatformError("test error", nil, "linux")

	if err.Kind != ErrKindPlatform {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindPlatform)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if err.Recoverable {
		t.Error("Platform errors should not be recoverable")
	}
	if err.AutoFix == nil {
		t.Error("Linux platform errors should have AutoFix")
	}
	if err.AutoFix.Command == "" {
		t.Error("AutoFix.Command should be set")
	}

	// Verify Linux-specific steps
	foundXvfb := false
	for _, step := range err.ManualSteps {
		if containsSubstring(step, "xvfb") {
			foundXvfb = true
			break
		}
	}
	if !foundXvfb {
		t.Error("Linux platform error should mention xvfb in ManualSteps")
	}
}

func TestNewPlatformError_HasRecoveryInfo_Mac(t *testing.T) {
	err := NewPlatformError("test error", nil, "mac")

	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}

	// Verify Mac-specific steps
	foundGatekeeper := false
	for _, step := range err.ManualSteps {
		if containsSubstring(step, "Gatekeeper") || containsSubstring(step, "signed") {
			foundGatekeeper = true
			break
		}
	}
	if !foundGatekeeper {
		t.Error("Mac platform error should mention signing or Gatekeeper")
	}
}

func TestNewPlatformError_HasRecoveryInfo_Windows(t *testing.T) {
	err := NewPlatformError("test error", nil, "win")

	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}

	// Verify Windows-specific steps
	foundDefender := false
	for _, step := range err.ManualSteps {
		if containsSubstring(step, "Defender") || containsSubstring(step, "Firewall") {
			foundDefender = true
			break
		}
	}
	if !foundDefender {
		t.Error("Windows platform error should mention Defender or Firewall")
	}
}

func TestNewStoreError_HasRecoveryInfo(t *testing.T) {
	err := NewStoreError("test error", nil)

	if err.Kind != ErrKindStore {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindStore)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if !err.Recoverable {
		t.Error("Store errors should be recoverable")
	}
	if err.RetryStrategy == nil {
		t.Error("Store errors should have retry strategy")
	}
}

func TestNewCancelledError_HasRecoveryInfo(t *testing.T) {
	err := NewCancelledError("test error")

	if err.Kind != ErrKindCancelled {
		t.Errorf("Kind = %v, want %v", err.Kind, ErrKindCancelled)
	}
	if err.SuggestedAction == "" {
		t.Error("SuggestedAction should be set")
	}
	if len(err.ManualSteps) == 0 {
		t.Error("ManualSteps should not be empty")
	}
	if err.Recoverable {
		t.Error("Cancelled errors should not be recoverable")
	}
}

// Test that Recoverable flag matches actual recoverability
func TestRecoverableFlag_MatchesRecoverability(t *testing.T) {
	tests := []struct {
		name        string
		err         *Error
		recoverable bool
	}{
		{"artifact", NewArtifactError("", nil, ""), false},
		{"execution", NewExecutionError("", nil, nil), true},
		{"timeout", NewTimeoutError("", nil, nil), true},
		{"validation", NewValidationError("", nil), false},
		{"telemetry", NewTelemetryError("", nil, nil), true},
		{"platform", NewPlatformError("", nil, "linux"), false},
		{"store", NewStoreError("", nil), true},
		{"cancelled", NewCancelledError(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Recoverable != tt.recoverable {
				t.Errorf("Recoverable = %v, want %v", tt.err.Recoverable, tt.recoverable)
			}
			// Verify recoverable errors have retry strategy
			if tt.recoverable && tt.err.RetryStrategy == nil {
				t.Error("Recoverable errors should have RetryStrategy")
			}
		})
	}
}

// Test that ManualSteps are actionable (non-empty strings)
func TestManualSteps_AreActionable(t *testing.T) {
	errors := []*Error{
		NewArtifactError("", nil, "/path"),
		NewExecutionError("", nil, nil),
		NewTimeoutError("", nil, nil),
		NewValidationError("", nil),
		NewTelemetryError("", nil, nil),
		NewPlatformError("", nil, "linux"),
		NewPlatformError("", nil, "mac"),
		NewPlatformError("", nil, "win"),
		NewStoreError("", nil),
		NewCancelledError(""),
	}

	for _, err := range errors {
		t.Run(err.Kind.String(), func(t *testing.T) {
			if len(err.ManualSteps) == 0 {
				t.Errorf("%s error should have ManualSteps", err.Kind.String())
				return
			}
			for i, step := range err.ManualSteps {
				if step == "" {
					t.Errorf("ManualSteps[%d] should not be empty", i)
				}
				if len(step) < 10 {
					t.Errorf("ManualSteps[%d] = %q seems too short to be actionable", i, step)
				}
			}
		})
	}
}

// Test RecoveryPaths coverage
func TestRecoveryPaths_AllKindsCovered(t *testing.T) {
	allKinds := []ErrorKind{
		ErrKindArtifact,
		ErrKindExecution,
		ErrKindTimeout,
		ErrKindValidation,
		ErrKindTelemetry,
		ErrKindPlatform,
		ErrKindStore,
		ErrKindCancelled,
	}

	for _, kind := range allKinds {
		t.Run(kind.String(), func(t *testing.T) {
			path, ok := RecoveryPaths[kind]
			if !ok {
				t.Errorf("RecoveryPaths missing entry for %v", kind)
			}
			if path == "" {
				t.Errorf("RecoveryPaths[%v] is empty", kind)
			}
		})
	}
}

// Helper types for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
