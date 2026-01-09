// Package errors provides structured error handling tests
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001]
package errors

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewDatabaseError(t *testing.T) {
	cause := errors.New("connection refused")
	err := NewDatabaseError("handlers", "save result", cause)

	if err.Code != CodeDatabaseError {
		t.Errorf("expected code %s, got %s", CodeDatabaseError, err.Code)
	}
	if err.Message != "Failed to save result" {
		t.Errorf("expected message 'Failed to save result', got '%s'", err.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("expected cause to be wrapped")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("checks", "check result", "test-check")

	if err.Code != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, err.Code)
	}
	if err.Message != "check result 'test-check' not found" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

func TestNewTimeoutError(t *testing.T) {
	cause := errors.New("context deadline exceeded")
	err := NewTimeoutError("timeline", "fetch events", cause)

	if err.Code != CodeTimeout {
		t.Errorf("expected code %s, got %s", CodeTimeout, err.Code)
	}
	if err.Message != "fetch events timed out" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

func TestAPIErrorStatusCode(t *testing.T) {
	tests := []struct {
		code     Code
		expected int
	}{
		{CodeDatabaseError, http.StatusInternalServerError},
		{CodeNotFound, http.StatusNotFound},
		{CodeTimeout, http.StatusGatewayTimeout},
		{CodeInternalError, http.StatusInternalServerError},
		{CodeValidation, http.StatusBadRequest},
		{CodeServiceUnavailable, http.StatusServiceUnavailable},
	}

	for _, tc := range tests {
		err := &APIError{Code: tc.code}
		if got := err.StatusCode(); got != tc.expected {
			t.Errorf("code %s: expected status %d, got %d", tc.code, tc.expected, got)
		}
	}
}

func TestLogAndRespond(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewDatabaseError("test", "query database", errors.New("db down"))

	LogAndRespond(w, apiErr)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"success":false`) {
		t.Error("expected success:false in response")
	}
	if !strings.Contains(body, `"error":"DATABASE_ERROR"`) {
		t.Error("expected error code in response")
	}
	if !strings.Contains(body, `"message":"Failed to query database"`) {
		t.Errorf("expected user-friendly message in response, got: %s", body)
	}
	if !strings.Contains(body, `"requestId"`) {
		t.Error("expected requestId in response")
	}
	// Should NOT contain the actual error cause
	if strings.Contains(body, "db down") {
		t.Error("response should not contain internal error details")
	}
}

func TestAPIErrorError(t *testing.T) {
	cause := errors.New("underlying issue")
	err := NewInternalError("component", "something failed", cause)

	// Error() should include component, message, and cause
	errStr := err.Error()
	if !strings.Contains(errStr, "component") {
		t.Error("expected component in error string")
	}
	if !strings.Contains(errStr, "something failed") {
		t.Error("expected message in error string")
	}
	if !strings.Contains(errStr, "underlying issue") {
		t.Error("expected cause in error string")
	}
}

func TestAPIErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := NewDatabaseError("store", "insert", cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find wrapped cause")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Error("Unwrap should return original cause")
	}
}

func TestResponseContentType(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewNotFoundError("test", "item", "123")

	LogAndRespond(w, apiErr)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestNewServiceUnavailableError(t *testing.T) {
	cause := errors.New("service down")
	err := NewServiceUnavailableError("health", "database", cause)

	if err.Code != CodeServiceUnavailable {
		t.Errorf("expected code %s, got %s", CodeServiceUnavailable, err.Code)
	}
	if err.Message != "database is currently unavailable" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if err.StatusCode() != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", err.StatusCode())
	}
	if !errors.Is(err, cause) {
		t.Error("expected cause to be wrapped")
	}
}

func TestNewInternalError(t *testing.T) {
	cause := errors.New("unexpected panic")
	err := NewInternalError("handler", "process request", cause)

	if err.Code != CodeInternalError {
		t.Errorf("expected code %s, got %s", CodeInternalError, err.Code)
	}
	if err.Message != "process request" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

func TestAPIErrorError_WithoutCause(t *testing.T) {
	// Test Error() when there is no cause (else branch)
	err := NewNotFoundError("component", "resource", "id")

	errStr := err.Error()
	if !strings.Contains(errStr, "component") {
		t.Error("expected component in error string")
	}
	if !strings.Contains(errStr, "not found") {
		t.Error("expected message in error string")
	}
	// Should not have ":" followed by cause
	if strings.HasSuffix(errStr, ": <nil>") {
		t.Error("error string should not end with nil cause")
	}
}

func TestAPIErrorStatusCode_UnknownCode(t *testing.T) {
	// Test with an unknown code to hit default case
	err := &APIError{Code: Code("UNKNOWN_ERROR")}
	if got := err.StatusCode(); got != http.StatusInternalServerError {
		t.Errorf("unknown code should default to 500, got %d", got)
	}
}

func TestLogError(t *testing.T) {
	// Just verify it doesn't panic
	LogError("test", "some operation", errors.New("test error"))
}

func TestLogInfo(t *testing.T) {
	// Test with details
	LogInfo("test", "operation completed", "detail1", 123)

	// Test without details
	LogInfo("test", "simple message")
}

func TestLogAndRespond_NotFoundStatus(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewNotFoundError("test", "resource", "missing-id")

	LogAndRespond(w, apiErr)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestLogAndRespond_TimeoutStatus(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewTimeoutError("test", "slow operation", errors.New("context deadline"))

	LogAndRespond(w, apiErr)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", w.Code)
	}
}

func TestLogAndRespond_ServiceUnavailableStatus(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewServiceUnavailableError("test", "external service", nil)

	LogAndRespond(w, apiErr)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}
