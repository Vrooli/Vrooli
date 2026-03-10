// DOC: docs/internal/ERROR_SEMANTICS.md
package errors

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestAPIErrorStatusCodes verifies correct HTTP status code mapping for each category.
// [REQ:LD-ERROR-SEMANTICS] Error categories map to appropriate HTTP status codes.
func TestAPIErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name     string
		category ErrorCategory
		want     int
	}{
		{"validation returns 400", CategoryValidation, http.StatusBadRequest},
		{"not_found returns 404", CategoryNotFound, http.StatusNotFound},
		{"conflict returns 409", CategoryConflict, http.StatusConflict},
		{"internal returns 500", CategoryInternal, http.StatusInternalServerError},
		{"unavailable returns 503", CategoryUnavailable, http.StatusServiceUnavailable},
		{"unknown category returns 500", ErrorCategory("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{Category: tt.category}
			if got := err.StatusCode(); got != tt.want {
				t.Errorf("StatusCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestNewValidationError verifies validation error creation.
func TestNewValidationError(t *testing.T) {
	err := NewValidationError(CodeInvalidJSON, "test message")

	if !err.IsError {
		t.Error("IsError should be true")
	}
	if err.Category != CategoryValidation {
		t.Errorf("Category = %s, want %s", err.Category, CategoryValidation)
	}
	if err.Code != CodeInvalidJSON {
		t.Errorf("Code = %s, want %s", err.Code, CodeInvalidJSON)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %s, want 'test message'", err.Message)
	}
	if err.Recovery != HintFixInput {
		t.Errorf("Recovery = %s, want %s", err.Recovery, HintFixInput)
	}
}

// TestNewNotFoundError verifies not found error creation with entity details.
func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError(CodeEventNotFound, "event", "abc-123")

	if !err.IsError {
		t.Error("IsError should be true")
	}
	if err.Category != CategoryNotFound {
		t.Errorf("Category = %s, want %s", err.Category, CategoryNotFound)
	}
	if err.Code != CodeEventNotFound {
		t.Errorf("Code = %s, want %s", err.Code, CodeEventNotFound)
	}
	if err.Message != "event not found: abc-123" {
		t.Errorf("Message = %s, want 'event not found: abc-123'", err.Message)
	}
	if err.Details["entity"] != "event" {
		t.Errorf("Details[entity] = %v, want 'event'", err.Details["entity"])
	}
	if err.Details["id"] != "abc-123" {
		t.Errorf("Details[id] = %v, want 'abc-123'", err.Details["id"])
	}
}

// TestNewInternalError verifies internal error creation.
func TestNewInternalError(t *testing.T) {
	err := NewInternalError(CodeDatabaseError, "Database temporarily unavailable")

	if !err.IsError {
		t.Error("IsError should be true")
	}
	if err.Category != CategoryInternal {
		t.Errorf("Category = %s, want %s", err.Category, CategoryInternal)
	}
	if err.Code != CodeDatabaseError {
		t.Errorf("Code = %s, want %s", err.Code, CodeDatabaseError)
	}
	if err.Recovery != HintRetryLater {
		t.Errorf("Recovery = %s, want %s", err.Recovery, HintRetryLater)
	}
}

// TestNewUnavailableError verifies unavailable error creation.
func TestNewUnavailableError(t *testing.T) {
	err := NewUnavailableError(CodeDependencyUnavailable, "Dependency down")

	if !err.IsError {
		t.Error("IsError should be true")
	}
	if err.Category != CategoryUnavailable {
		t.Errorf("Category = %s, want %s", err.Category, CategoryUnavailable)
	}
	if err.Recovery != HintCheckScenario {
		t.Errorf("Recovery = %s, want %s", err.Recovery, HintCheckScenario)
	}
}

// TestWithDetails verifies adding details to an error.
func TestWithDetails(t *testing.T) {
	err := NewValidationError(CodeMissingField, "Field required")
	err = err.WithDetails("field", "domain")
	err = err.WithDetails("constraint", "required")

	if err.Details["field"] != "domain" {
		t.Errorf("Details[field] = %v, want 'domain'", err.Details["field"])
	}
	if err.Details["constraint"] != "required" {
		t.Errorf("Details[constraint] = %v, want 'required'", err.Details["constraint"])
	}
}

// TestAPIErrorError verifies the error interface implementation.
func TestAPIErrorError(t *testing.T) {
	err := NewValidationError(CodeInvalidJSON, "test message")
	if err.Error() != "test message" {
		t.Errorf("Error() = %s, want 'test message'", err.Error())
	}
}

// TestAPIErrorToJSON verifies JSON serialization.
func TestAPIErrorToJSON(t *testing.T) {
	err := NewNotFoundError(CodeEventNotFound, "event", "test-id")
	data := err.ToJSON()

	var parsed map[string]interface{}
	if jsonErr := json.Unmarshal(data, &parsed); jsonErr != nil {
		t.Fatalf("Failed to parse JSON: %v", jsonErr)
	}

	// Verify key JSON fields
	if parsed["error"] != true {
		t.Error("JSON 'error' should be true")
	}
	if parsed["category"] != "not_found" {
		t.Errorf("JSON 'category' = %v, want 'not_found'", parsed["category"])
	}
	if parsed["code"] != "EVENT_NOT_FOUND" {
		t.Errorf("JSON 'code' = %v, want 'EVENT_NOT_FOUND'", parsed["code"])
	}
}

// TestPrebuiltErrors verifies common pre-built errors are configured correctly.
func TestPrebuiltErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		code   ErrorCode
		hasField bool
		field  string
	}{
		{"ErrInvalidJSON", ErrInvalidJSON, CodeInvalidJSON, false, ""},
		{"ErrMissingDomain", ErrMissingDomain, CodeMissingField, true, "domain"},
		{"ErrMissingEventType", ErrMissingEventType, CodeMissingField, true, "event_type"},
		{"ErrMissingName", ErrMissingName, CodeMissingField, true, "name"},
		{"ErrMissingDisplayName", ErrMissingDisplayName, CodeMissingField, true, "display_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.err.IsError {
				t.Error("IsError should be true")
			}
			if tt.err.Category != CategoryValidation {
				t.Errorf("Category = %s, want %s", tt.err.Category, CategoryValidation)
			}
			if tt.err.Code != tt.code {
				t.Errorf("Code = %s, want %s", tt.err.Code, tt.code)
			}
			if tt.hasField {
				if tt.err.Details["field"] != tt.field {
					t.Errorf("Details[field] = %v, want %s", tt.err.Details["field"], tt.field)
				}
			}
		})
	}
}
