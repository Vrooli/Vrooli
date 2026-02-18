// Package errors provides structured error handling for the autoheal API.
// It enables consistent error responses, logging, and observability without leaking
// sensitive details to clients.
//
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001]
package errors

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ╔════════════════════════════════════════════════════════════════╗
// ║  ERROR CODES - Read before modifying                          ║
// ║                                                                ║
// ║  Each code maps to a specific HTTP status and recovery path.  ║
// ║  Clients use these to determine appropriate recovery actions.  ║
// ║                                                                ║
// ║  To add a new code:                                            ║
// ║  1. Ensure it has a distinct recovery path from existing codes ║
// ║  2. Add an HTTP status mapping in StatusCode()                ║
// ║  3. Add a recovery hint in the constructor function           ║
// ║  4. Update the UI APIError.getUserMessage() to handle it      ║
// ║  5. Add tests in errors_test.go                               ║
// ╚════════════════════════════════════════════════════════════════╝

// Code represents a machine-readable error code for categorization.
// Clients can use these to provide appropriate recovery actions.
type Code string

const (
	// CodeDatabaseError indicates a database operation failed.
	// Recovery: retry after a short delay; if persistent, check database connectivity.
	CodeDatabaseError Code = "DATABASE_ERROR"

	// CodeNotFound indicates the requested resource doesn't exist.
	// Recovery: verify the resource identifier; the resource may have been removed.
	CodeNotFound Code = "NOT_FOUND"

	// CodeTimeout indicates an operation timed out.
	// Recovery: retry after a short delay; the operation may complete on next attempt.
	CodeTimeout Code = "TIMEOUT"

	// CodeInternalError indicates an unexpected internal error (bug or invariant violation).
	// Recovery: do not retry automatically; report the issue with the request ID.
	CodeInternalError Code = "INTERNAL_ERROR"

	// CodeValidation indicates invalid input was provided.
	// Recovery: fix the input and resubmit; do not retry with the same input.
	CodeValidation Code = "VALIDATION_ERROR"

	// CodeServiceUnavailable indicates a dependency is unavailable.
	// Recovery: retry after backoff; the dependency may come back online.
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"

	// CodeConflict indicates a conflicting operation is already in progress.
	// Recovery: wait for the current operation to complete, then retry.
	CodeConflict Code = "CONFLICT"
)

// RecoveryAction indicates what the client should do about the error.
type RecoveryAction string

const (
	// RecoveryRetry means the client should retry the same request after a delay.
	RecoveryRetry RecoveryAction = "retry"
	// RecoveryFixInput means the client should correct the input and resubmit.
	RecoveryFixInput RecoveryAction = "fix_input"
	// RecoveryReport means the error is unexpected and should be reported.
	RecoveryReport RecoveryAction = "report"
	// RecoveryWait means another operation must finish first.
	RecoveryWait RecoveryAction = "wait"
	// RecoveryNone means no automatic recovery is possible.
	RecoveryNone RecoveryAction = "none"
)

// Recovery provides machine-readable recovery guidance to clients.
type Recovery struct {
	// Action is the recommended recovery action.
	Action RecoveryAction `json:"action"`
	// Retryable indicates whether the client should attempt to retry.
	Retryable bool `json:"retryable"`
	// Hint is a short human-readable suggestion (e.g., "Check if the database is running").
	Hint string `json:"hint,omitempty"`
}

// APIError represents a structured error that can be returned to clients.
// It separates the user-safe message from internal details.
type APIError struct {
	Code     Code     `json:"code"`
	Message  string   `json:"message"`  // Safe for users
	Recovery Recovery `json:"recovery"` // Recovery guidance
	// RequestID is set by LogAndRespond; not populated at construction time.
	RequestID string `json:"requestId"`

	// Internal fields - not serialized
	cause     error  // Original error
	component string // Which component failed
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.component, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.component, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As.
func (e *APIError) Unwrap() error {
	return e.cause
}

// StatusCode returns the appropriate HTTP status code for this error.
func (e *APIError) StatusCode() int {
	switch e.Code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeValidation:
		return http.StatusBadRequest
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case CodeConflict:
		return http.StatusConflict
	case CodeDatabaseError, CodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse is the JSON structure returned to clients.
type ErrorResponse struct {
	Success   bool     `json:"success"`
	Error     Code     `json:"error"`
	Message   string   `json:"message"`
	Recovery  Recovery `json:"recovery"`
	RequestID string   `json:"requestId,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// NewDatabaseError creates an error for database failures.
// The cause is logged but not exposed to clients.
func NewDatabaseError(component string, operation string, cause error) *APIError {
	return &APIError{
		Code:    CodeDatabaseError,
		Message: fmt.Sprintf("Failed to %s", operation),
		Recovery: Recovery{
			Action:    RecoveryRetry,
			Retryable: true,
			Hint:      "Database may be temporarily unavailable. Your data is safe.",
		},
		cause:     cause,
		component: component,
	}
}

// NewNotFoundError creates an error when a resource isn't found.
func NewNotFoundError(component string, resourceType string, resourceID string) *APIError {
	return &APIError{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s '%s' not found", resourceType, resourceID),
		Recovery: Recovery{
			Action:    RecoveryNone,
			Retryable: false,
			Hint:      "The requested item may have been removed or the identifier is incorrect.",
		},
		component: component,
	}
}

// NewTimeoutError creates an error for operation timeouts.
func NewTimeoutError(component string, operation string, cause error) *APIError {
	return &APIError{
		Code:    CodeTimeout,
		Message: fmt.Sprintf("%s timed out", operation),
		Recovery: Recovery{
			Action:    RecoveryRetry,
			Retryable: true,
			Hint:      "The operation took too long. Try again shortly.",
		},
		cause:     cause,
		component: component,
	}
}

// NewInternalError creates a generic internal error.
// Use sparingly - prefer more specific error types.
func NewInternalError(component string, message string, cause error) *APIError {
	return &APIError{
		Code:    CodeInternalError,
		Message: message,
		Recovery: Recovery{
			Action:    RecoveryReport,
			Retryable: false,
			Hint:      "An unexpected error occurred. If this persists, report the request ID.",
		},
		cause:     cause,
		component: component,
	}
}

// NewServiceUnavailableError creates an error when a dependency is down.
func NewServiceUnavailableError(component string, service string, cause error) *APIError {
	return &APIError{
		Code:    CodeServiceUnavailable,
		Message: fmt.Sprintf("%s is currently unavailable", service),
		Recovery: Recovery{
			Action:    RecoveryRetry,
			Retryable: true,
			Hint:      "A required service is down. It may recover automatically.",
		},
		cause:     cause,
		component: component,
	}
}

// NewValidationError creates an error for invalid input.
// The cause message is logged but NOT included in the user-facing message
// to avoid leaking internal details.
func NewValidationError(component string, description string, cause error) *APIError {
	return &APIError{
		Code:    CodeValidation,
		Message: fmt.Sprintf("Invalid input: %s", description),
		Recovery: Recovery{
			Action:    RecoveryFixInput,
			Retryable: false,
			Hint:      "Check the request data and try again with corrected input.",
		},
		cause:     cause,
		component: component,
	}
}

// NewConflictError creates an error when a conflicting operation is in progress.
func NewConflictError(component string, message string) *APIError {
	return &APIError{
		Code:    CodeConflict,
		Message: message,
		Recovery: Recovery{
			Action:    RecoveryWait,
			Retryable: true,
			Hint:      "Wait for the current operation to complete, then try again.",
		},
		component: component,
	}
}

// generateRequestID creates a short, collision-resistant request ID.
// Uses 4 bytes of crypto/rand → 8 hex chars (4 billion possible values).
func generateRequestID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails (extremely unlikely)
		return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	}
	return hex.EncodeToString(b)
}

// LogAndRespond logs the error with full context and writes a safe response.
// This is the primary function for handling errors in HTTP handlers.
//
// Usage:
//
//	if err := someOperation(); err != nil {
//	    errors.LogAndRespond(w, errors.NewDatabaseError("timeline", "fetch events", err))
//	    return
//	}
func LogAndRespond(w http.ResponseWriter, apiErr *APIError) {
	requestID := generateRequestID()
	apiErr.RequestID = requestID

	// Log full error details for debugging (cause included here, never in response)
	log.Printf("[ERROR] request=%s component=%s code=%s message=%q cause=%v",
		requestID, apiErr.component, apiErr.Code, apiErr.Message, apiErr.cause)

	// Send safe response to client
	response := ErrorResponse{
		Success:   false,
		Error:     apiErr.Code,
		Message:   apiErr.Message,
		Recovery:  apiErr.Recovery,
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode())
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] component=errors operation=encode_response error=%v", err)
	}
}

// LogError logs an error without sending an HTTP response.
// Use this for non-fatal errors that shouldn't stop the request.
//
// Example: Persistence failures during a tick that shouldn't fail the tick.
func LogError(component string, operation string, err error) {
	log.Printf("[WARN] component=%s operation=%s error=%v", component, operation, err)
}

// LogInfo logs informational messages for observability.
func LogInfo(component string, message string, details ...interface{}) {
	if len(details) > 0 {
		log.Printf("[INFO] component=%s message=%s details=%v", component, message, details)
	} else {
		log.Printf("[INFO] component=%s message=%s", component, message)
	}
}
