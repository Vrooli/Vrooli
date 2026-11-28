package smoketest

import (
	"strings"
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

// Test every constructor against independent recovery-contract expectations.
// Each kind's policy stays visible while the common safety checks stay in one place.
func TestErrorRecoveryContracts(t *testing.T) {
	tests := []struct {
		name          string
		kind          ErrorKind
		newError      func() *Error
		recoverable   bool
		retryExpected bool
		autoFix       bool
		recoveryPath  string
		manualHint    string
	}{
		{"artifact", ErrKindArtifact, func() *Error { return NewArtifactError("test error", nil, "/path/to/artifact") }, false, false, false, "Check build stage output", "artifact exists"},
		{"execution", ErrKindExecution, func() *Error { return NewExecutionError("test error", nil, nil) }, true, true, false, "Check app startup logs", "run manually"},
		{"timeout", ErrKindTimeout, func() *Error { return NewTimeoutError("test error", nil, nil) }, true, true, false, "Increase timeout", "TIMEOUT_MS"},
		{"validation", ErrKindValidation, func() *Error { return NewValidationError("test error", nil) }, false, false, false, "Ensure app outputs", "SMOKE_TEST_RESULT=passed"},
		{"telemetry", ErrKindTelemetry, func() *Error { return NewTelemetryError("test error", nil, nil) }, true, true, false, "Check telemetry service", "network connectivity"},
		{"platform/linux", ErrKindPlatform, func() *Error { return NewPlatformError("test error", nil, "linux") }, false, false, true, "Install required dependencies", "xvfb"},
		{"platform/mac", ErrKindPlatform, func() *Error { return NewPlatformError("test error", nil, "mac") }, false, false, false, "Install required dependencies", "Gatekeeper"},
		{"platform/windows", ErrKindPlatform, func() *Error { return NewPlatformError("test error", nil, "win") }, false, false, false, "Install required dependencies", "Defender"},
		{"store", ErrKindStore, func() *Error { return NewStoreError("test error", nil) }, true, true, false, "Check disk space", "permissions"},
		{"cancelled", ErrKindCancelled, func() *Error { return NewCancelledError("test error") }, false, false, false, "Re-run smoke test", "cancellation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.newError()
			if err.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", err.Kind, tt.kind)
			}
			if err.Recoverable != tt.recoverable {
				t.Errorf("Recoverable = %v, want %v", err.Recoverable, tt.recoverable)
			}
			if (err.RetryStrategy != nil) != tt.retryExpected {
				t.Errorf("RetryStrategy present = %v, want %v", err.RetryStrategy != nil, tt.retryExpected)
			}
			if err.RetryStrategy != nil && err.RetryStrategy.MaxAttempts <= 0 {
				t.Errorf("RetryStrategy.MaxAttempts = %d, want positive", err.RetryStrategy.MaxAttempts)
			}
			if (err.AutoFix != nil) != tt.autoFix {
				t.Errorf("AutoFix present = %v, want %v", err.AutoFix != nil, tt.autoFix)
			}
			if err.AutoFix != nil && err.AutoFix.Command == "" {
				t.Error("AutoFix.Command should be set")
			}
			if err.SuggestedAction == "" {
				t.Error("SuggestedAction should be set")
			}
			if len(err.ManualSteps) == 0 {
				t.Fatal("ManualSteps should not be empty")
			}
			for i, step := range err.ManualSteps {
				if len(step) < 10 {
					t.Errorf("ManualSteps[%d] = %q, want an actionable instruction", i, step)
				}
			}
			if !containsSubstring(strings.Join(err.ManualSteps, " "), tt.manualHint) {
				t.Errorf("ManualSteps %q do not include independent hint %q", err.ManualSteps, tt.manualHint)
			}
			if got := RecoveryPaths[tt.kind]; got == "" || !containsSubstring(got, tt.recoveryPath) {
				t.Errorf("RecoveryPaths[%v] = %q, want a path containing %q", tt.kind, got, tt.recoveryPath)
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
