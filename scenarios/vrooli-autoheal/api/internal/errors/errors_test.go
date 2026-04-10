// Package errors provides structured error handling tests
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001]
package errors

import (
	"encoding/json"
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
	if err.Recovery.Action != RecoveryRetry {
		t.Errorf("expected recovery action %s, got %s", RecoveryRetry, err.Recovery.Action)
	}
	if !err.Recovery.Retryable {
		t.Error("database errors should be retryable")
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
	if err.Recovery.Action != RecoveryNone {
		t.Errorf("expected recovery action %s, got %s", RecoveryNone, err.Recovery.Action)
	}
	if err.Recovery.Retryable {
		t.Error("not found errors should not be retryable")
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
	if err.Recovery.Action != RecoveryRetry {
		t.Errorf("expected recovery action %s, got %s", RecoveryRetry, err.Recovery.Action)
	}
	if !err.Recovery.Retryable {
		t.Error("timeout errors should be retryable")
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
	if err.Recovery.Action != RecoveryReport {
		t.Errorf("expected recovery action %s, got %s", RecoveryReport, err.Recovery.Action)
	}
	if err.Recovery.Retryable {
		t.Error("internal errors should NOT be retryable")
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
	if err.Recovery.Action != RecoveryRetry {
		t.Errorf("expected recovery action %s, got %s", RecoveryRetry, err.Recovery.Action)
	}
	if !err.Recovery.Retryable {
		t.Error("service unavailable errors should be retryable")
	}
}

func TestNewValidationError(t *testing.T) {
	cause := errors.New("invalid JSON at line 5")
	err := NewValidationError("config", "parse request body", cause)

	if err.Code != CodeValidation {
		t.Errorf("expected code %s, got %s", CodeValidation, err.Code)
	}
	// Message should NOT contain the cause text (avoid leaking internals)
	if strings.Contains(err.Message, "invalid JSON") {
		t.Error("validation error message should not contain cause text")
	}
	if err.Message != "Invalid input: parse request body" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if err.Recovery.Action != RecoveryFixInput {
		t.Errorf("expected recovery action %s, got %s", RecoveryFixInput, err.Recovery.Action)
	}
	if err.Recovery.Retryable {
		t.Error("validation errors should NOT be retryable (fix input first)")
	}
}

func TestNewValidationError_NilCause(t *testing.T) {
	err := NewValidationError("config", "scenario name is required", nil)

	if err.Code != CodeValidation {
		t.Errorf("expected code %s, got %s", CodeValidation, err.Code)
	}
	if err.Message != "Invalid input: scenario name is required" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("tick", "A health check cycle is already running")

	if err.Code != CodeConflict {
		t.Errorf("expected code %s, got %s", CodeConflict, err.Code)
	}
	if err.StatusCode() != http.StatusConflict {
		t.Errorf("expected status 409, got %d", err.StatusCode())
	}
	if err.Recovery.Action != RecoveryWait {
		t.Errorf("expected recovery action %s, got %s", RecoveryWait, err.Recovery.Action)
	}
	if !err.Recovery.Retryable {
		t.Error("conflict errors should be retryable (after waiting)")
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
		{CodeConflict, http.StatusConflict},
	}

	for _, tc := range tests {
		err := &APIError{Code: tc.code}
		if got := err.StatusCode(); got != tc.expected {
			t.Errorf("code %s: expected status %d, got %d", tc.code, tc.expected, got)
		}
	}
}

func TestAPIErrorStatusCode_UnknownCode(t *testing.T) {
	err := &APIError{Code: Code("UNKNOWN_ERROR")}
	if got := err.StatusCode(); got != http.StatusInternalServerError {
		t.Errorf("unknown code should default to 500, got %d", got)
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

func TestLogAndRespond_IncludesRecovery(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewDatabaseError("test", "query database", errors.New("db down"))

	LogAndRespond(w, apiErr)

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Recovery.Action != RecoveryRetry {
		t.Errorf("expected recovery action %s, got %s", RecoveryRetry, resp.Recovery.Action)
	}
	if !resp.Recovery.Retryable {
		t.Error("expected retryable=true for database error")
	}
	if resp.Recovery.Hint == "" {
		t.Error("expected non-empty recovery hint")
	}
}

func TestLogAndRespond_NotFoundStatus(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewNotFoundError("test", "resource", "missing-id")

	LogAndRespond(w, apiErr)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Recovery.Retryable {
		t.Error("not found should not be retryable")
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

func TestLogAndRespond_ConflictStatus(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewConflictError("tick", "operation in progress")

	LogAndRespond(w, apiErr)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Recovery.Action != RecoveryWait {
		t.Errorf("expected recovery action %s, got %s", RecoveryWait, resp.Recovery.Action)
	}
}

func TestLogAndRespond_InternalError_NotRetryable(t *testing.T) {
	w := httptest.NewRecorder()
	apiErr := NewInternalError("handler", "unexpected failure", errors.New("nil pointer"))

	LogAndRespond(w, apiErr)

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Recovery.Retryable {
		t.Error("internal errors should NOT be retryable")
	}
	if resp.Recovery.Action != RecoveryReport {
		t.Errorf("expected recovery action %s, got %s", RecoveryReport, resp.Recovery.Action)
	}
}

func TestAPIErrorError(t *testing.T) {
	cause := errors.New("underlying issue")
	err := NewInternalError("component", "something failed", cause)

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

func TestAPIErrorError_WithoutCause(t *testing.T) {
	err := NewNotFoundError("component", "resource", "id")

	errStr := err.Error()
	if !strings.Contains(errStr, "component") {
		t.Error("expected component in error string")
	}
	if !strings.Contains(errStr, "not found") {
		t.Error("expected message in error string")
	}
	if strings.HasSuffix(errStr, ": <nil>") {
		t.Error("error string should not end with nil cause")
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

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateRequestID()
		if ids[id] {
			t.Fatalf("duplicate request ID generated: %s (after %d iterations)", id, i)
		}
		ids[id] = true
		// Should be 8 hex chars
		if len(id) != 8 {
			t.Errorf("expected 8-char request ID, got %d chars: %s", len(id), id)
		}
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

// TestRecoveryPaths verifies that each error code has the correct recovery semantics.
// This is a safeguard against changes that break the recovery contract.
func TestRecoveryPaths(t *testing.T) {
	tests := []struct {
		name      string
		err       *APIError
		action    RecoveryAction
		retryable bool
	}{
		{"database", NewDatabaseError("c", "op", nil), RecoveryRetry, true},
		{"not_found", NewNotFoundError("c", "type", "id"), RecoveryNone, false},
		{"timeout", NewTimeoutError("c", "op", nil), RecoveryRetry, true},
		{"internal", NewInternalError("c", "msg", nil), RecoveryReport, false},
		{"validation", NewValidationError("c", "desc", nil), RecoveryFixInput, false},
		{"service_unavailable", NewServiceUnavailableError("c", "svc", nil), RecoveryRetry, true},
		{"conflict", NewConflictError("c", "msg"), RecoveryWait, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Recovery.Action != tc.action {
				t.Errorf("recovery action = %s, want %s", tc.err.Recovery.Action, tc.action)
			}
			if tc.err.Recovery.Retryable != tc.retryable {
				t.Errorf("retryable = %v, want %v", tc.err.Recovery.Retryable, tc.retryable)
			}
		})
	}
}
