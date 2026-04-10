package apierrors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestConstructors(t *testing.T) {
	tests := []struct {
		name           string
		err            *APIError
		wantCategory   Category
		wantRecovery   string
		wantField      string
		wantRetryAfter int
		wantMsgSubstr  string
	}{
		{
			name:          "Validation",
			err:           Validation("email", "required"),
			wantCategory:  CategoryValidation,
			wantRecovery:  RecoveryFixInput,
			wantField:     "email",
			wantMsgSubstr: "email",
		},
		{
			name:          "Unauthorized",
			err:           Unauthorized("bad token"),
			wantCategory:  CategoryUnauthorized,
			wantRecovery:  RecoveryAuthenticate,
			wantMsgSubstr: "bad token",
		},
		{
			name:          "Forbidden",
			err:           Forbidden("no access"),
			wantCategory:  CategoryForbidden,
			wantRecovery:  RecoveryContactAdmin,
			wantMsgSubstr: "no access",
		},
		{
			name:          "NotFound",
			err:           NotFound("user", "42"),
			wantCategory:  CategoryNotFound,
			wantRecovery:  RecoveryNone,
			wantMsgSubstr: "user not found: 42",
		},
		{
			name:          "Conflict",
			err:           Conflict("duplicate entry"),
			wantCategory:  CategoryConflict,
			wantRecovery:  RecoveryWait,
			wantMsgSubstr: "duplicate",
		},
		{
			name:           "Cooldown",
			err:            Cooldown(60),
			wantCategory:   CategoryCooldown,
			wantRecovery:   RecoveryWait,
			wantRetryAfter: 60,
			wantMsgSubstr:  "60 seconds",
		},
		{
			name:          "Unavailable",
			err:           Unavailable("agent-manager"),
			wantCategory:  CategoryUnavailable,
			wantRecovery:  RecoveryWait,
			wantMsgSubstr: "agent-manager",
		},
		{
			name:          "Internal",
			err:           Internal("something broke", fmt.Errorf("db connection refused")),
			wantCategory:  CategoryInternal,
			wantRecovery:  RecoveryNone,
			wantMsgSubstr: "something broke",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != tt.wantCategory {
				t.Errorf("Category = %q, want %q", tt.err.Category, tt.wantCategory)
			}
			if tt.err.Recovery != tt.wantRecovery {
				t.Errorf("Recovery = %q, want %q", tt.err.Recovery, tt.wantRecovery)
			}
			if tt.wantField != "" && tt.err.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", tt.err.Field, tt.wantField)
			}
			if tt.err.RetryAfterSecs != tt.wantRetryAfter {
				t.Errorf("RetryAfterSecs = %d, want %d", tt.err.RetryAfterSecs, tt.wantRetryAfter)
			}
			if !strings.Contains(tt.err.UserMessage, tt.wantMsgSubstr) {
				t.Errorf("UserMessage = %q, want substring %q", tt.err.UserMessage, tt.wantMsgSubstr)
			}
		})
	}
}

func TestError_WithUnderlying(t *testing.T) {
	cause := fmt.Errorf("db connection refused")
	err := Internal("oops", cause)

	errStr := err.Error()
	if !strings.Contains(errStr, "internal") {
		t.Errorf("Error() = %q, want to contain 'internal'", errStr)
	}
	if !strings.Contains(errStr, "oops") {
		t.Errorf("Error() = %q, want to contain 'oops'", errStr)
	}
	if !strings.Contains(errStr, "db connection refused") {
		t.Errorf("Error() = %q, want to contain underlying cause", errStr)
	}
}

func TestError_WithoutUnderlying(t *testing.T) {
	err := NotFound("item", "123")
	errStr := err.Error()
	if !strings.Contains(errStr, "not_found") {
		t.Errorf("Error() = %q, want to contain 'not_found'", errStr)
	}
	if !strings.Contains(errStr, "item not found: 123") {
		t.Errorf("Error() = %q, want to contain user message", errStr)
	}
}

func TestErrorsIs_UnwrapChain(t *testing.T) {
	cause := fmt.Errorf("disk full")
	wrapped := fmt.Errorf("save failed: %w", cause)
	apiErr := Internal("unable to save", wrapped)

	if !errors.Is(apiErr, wrapped) {
		t.Error("expected errors.Is to find wrapped error through APIError.Unwrap()")
	}
	if !errors.Is(apiErr, cause) {
		t.Error("expected errors.Is to find root cause through unwrap chain")
	}
}

func TestErrorsAs_APIError(t *testing.T) {
	apiErr := Validation("name", "too short")
	var wrapped error = fmt.Errorf("handler: %w", apiErr)

	var target *APIError
	if !errors.As(wrapped, &target) {
		t.Fatal("expected errors.As to find *APIError")
	}
	if target.Category != CategoryValidation {
		t.Errorf("Category = %q, want %q", target.Category, CategoryValidation)
	}
}
