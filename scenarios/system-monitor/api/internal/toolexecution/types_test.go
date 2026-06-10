package toolexecution

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
)

func TestNewErrorResultFromAPIError_Cooldown(t *testing.T) {
	apiErr := apierrors.Cooldown(30)
	result := NewErrorResultFromAPIError(apiErr)

	if result.Code != ErrorCodeCooldown {
		t.Errorf("Code = %q, want %q", result.Code, ErrorCodeCooldown)
	}
	if !result.Retryable {
		t.Error("expected Retryable = true for cooldown")
	}
	if result.RetryAfterSecs != 30 {
		t.Errorf("RetryAfterSecs = %d, want 30", result.RetryAfterSecs)
	}
}

func TestNewErrorResultFromAPIError_Validation(t *testing.T) {
	apiErr := apierrors.Validation("x", "y")
	result := NewErrorResultFromAPIError(apiErr)

	if result.Code != ErrorCodeInvalidArgs {
		t.Errorf("Code = %q, want %q", result.Code, ErrorCodeInvalidArgs)
	}
	if result.Retryable {
		t.Error("expected Retryable = false for validation")
	}
}

func TestNewErrorResult_RetryableComputation(t *testing.T) {
	tests := []struct {
		code      string
		retryable bool
	}{
		{ErrorCodeCooldown, true},
		{ErrorCodeUnavailable, true},
		{ErrorCodeConflict, true},
		{ErrorCodeInternalError, false},
		{ErrorCodeInvalidArgs, false},
		{ErrorCodeNotFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := NewErrorResult(tt.code, "test")
			if result.Retryable != tt.retryable {
				t.Errorf("Retryable = %v, want %v for code %q", result.Retryable, tt.retryable, tt.code)
			}
		})
	}
}
