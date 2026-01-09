package shared

import (
	"errors"
	"testing"
)

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "invalid input" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
	if !IsValidationError(err) {
		t.Fatal("expected IsValidationError to detect custom error")
	}
}

func TestIsValidationErrorFalse(t *testing.T) {
	if IsValidationError(errors.New("other")) {
		t.Fatal("non-validation error should not match")
	}
	if IsValidationError(nil) {
		t.Fatal("nil error should not be treated as validation error")
	}
}
