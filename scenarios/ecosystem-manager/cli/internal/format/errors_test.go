package format

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestWrapAPIError_StructuredError(t *testing.T) {
	apiErr := &cliutil.APIError{
		StatusCode:   409,
		Message:      "duplicate task",
		Code:         "duplicate_task",
		RecoveryHint: "View existing: ecosystem-manager task show abc-123",
	}

	err := WrapAPIError("Failed to create task", apiErr)
	msg := err.Error()

	if !strings.Contains(msg, "Failed to create task") {
		t.Errorf("expected prefix in error, got: %s", msg)
	}
	if !strings.Contains(msg, "duplicate_task") {
		t.Errorf("expected error code in output, got: %s", msg)
	}
	if !strings.Contains(msg, "Recovery:") {
		t.Errorf("expected recovery hint in output, got: %s", msg)
	}
}

func TestWrapAPIError_PlainError(t *testing.T) {
	plainErr := errors.New("connection refused")

	err := WrapAPIError("Failed to create task", plainErr)
	msg := err.Error()

	if !strings.Contains(msg, "Failed to create task: connection refused") {
		t.Errorf("expected simple wrapping, got: %s", msg)
	}
}

func TestWrapAPIError_UnstructuredAPIError(t *testing.T) {
	apiErr := &cliutil.APIError{
		StatusCode: 500,
		Message:    "internal server error",
		// No Code, RecoveryHint, etc. — IsStructured() returns false
	}

	err := WrapAPIError("Failed", apiErr)
	msg := err.Error()

	if !strings.Contains(msg, "Failed: ") {
		t.Errorf("expected simple wrapping for unstructured API error, got: %s", msg)
	}
}
