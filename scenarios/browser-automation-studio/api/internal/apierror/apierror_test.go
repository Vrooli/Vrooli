package apierror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		Status:  http.StatusBadRequest,
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}

	if err.Error() != "Test error message" {
		t.Errorf("Expected 'Test error message', got '%s'", err.Error())
	}
}

func TestAPIError_WithDetails(t *testing.T) {
	original := &APIError{
		Status:  http.StatusBadRequest,
		Code:    "TEST_ERROR",
		Message: "Test message",
	}

	details := map[string]string{"field": "name"}
	withDetails := original.WithDetails(details)

	// Original should be unchanged
	if original.Details != nil {
		t.Error("Original error should not have details")
	}

	// New error should have details
	if withDetails.Details == nil {
		t.Error("New error should have details")
	}
	if withDetails.Code != original.Code {
		t.Errorf("Expected code '%s', got '%s'", original.Code, withDetails.Code)
	}
	if withDetails.Message != original.Message {
		t.Errorf("Expected message '%s', got '%s'", original.Message, withDetails.Message)
	}
}

func TestAPIError_WithMessage(t *testing.T) {
	original := &APIError{
		Status:  http.StatusBadRequest,
		Code:    "TEST_ERROR",
		Message: "Original message",
		Details: map[string]string{"key": "value"},
	}

	withMessage := original.WithMessage("New message")

	// Original should be unchanged
	if original.Message != "Original message" {
		t.Error("Original message should not change")
	}

	// New error should have new message but same details
	if withMessage.Message != "New message" {
		t.Errorf("Expected 'New message', got '%s'", withMessage.Message)
	}
	if withMessage.Code != original.Code {
		t.Errorf("Expected code '%s', got '%s'", original.Code, withMessage.Code)
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	err := &APIError{
		Status:  http.StatusNotFound,
		Code:    "NOT_FOUND",
		Message: "Resource not found",
	}

	RespondError(w, err)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}

	var response APIError
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Code != "NOT_FOUND" {
		t.Errorf("Expected code 'NOT_FOUND', got '%s'", response.Code)
	}
}

func TestRespondSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}

	RespondSuccess(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["key"] != "value" {
		t.Errorf("Expected 'value', got '%s'", response["key"])
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		status int
		code   string
	}{
		{"ErrInvalidRequest", ErrInvalidRequest, http.StatusBadRequest, "INVALID_REQUEST"},
		{"ErrMissingRequiredField", ErrMissingRequiredField, http.StatusBadRequest, "MISSING_REQUIRED_FIELD"},
		{"ErrWorkflowNotFound", ErrWorkflowNotFound, http.StatusNotFound, "WORKFLOW_NOT_FOUND"},
		{"ErrProjectNotFound", ErrProjectNotFound, http.StatusNotFound, "PROJECT_NOT_FOUND"},
		{"ErrConflict", ErrConflict, http.StatusConflict, "WORKFLOW_CONFLICT"},
		{"ErrInternalServer", ErrInternalServer, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"},
		{"ErrAIServiceError", ErrAIServiceError, http.StatusBadGateway, "AI_SERVICE_ERROR"},
		{"ErrRequestTimeout", ErrRequestTimeout, http.StatusRequestTimeout, "REQUEST_TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, tt.err.Status)
			}
			if tt.err.Code != tt.code {
				t.Errorf("Expected code '%s', got '%s'", tt.code, tt.err.Code)
			}
		})
	}
}
