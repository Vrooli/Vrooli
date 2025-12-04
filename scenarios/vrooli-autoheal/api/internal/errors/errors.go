// Package errors provides structured error handling for the autoheal API.
// It enables consistent error responses, logging, and observability without leaking
// sensitive details to clients.
//
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001]
package errors

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Code represents a machine-readable error code for categorization.
// Clients can use these to provide appropriate recovery actions.
type Code string

const (
	// CodeDatabaseError indicates a database operation failed.
	CodeDatabaseError Code = "DATABASE_ERROR"
	// CodeNotFound indicates the requested resource doesn't exist.
	CodeNotFound Code = "NOT_FOUND"
	// CodeTimeout indicates an operation timed out.
	CodeTimeout Code = "TIMEOUT"
	// CodeInternalError indicates an unexpected internal error.
	CodeInternalError Code = "INTERNAL_ERROR"
	// CodeValidation indicates invalid input was provided.
	CodeValidation Code = "VALIDATION_ERROR"
	// CodeServiceUnavailable indicates a dependency is unavailable.
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
)

// APIError represents a structured error that can be returned to clients.
// It separates the user-safe message from internal details.
type APIError struct {
	Code      Code   `json:"code"`
	Message   string `json:"message"`   // Safe for users
	RequestID string `json:"requestId"` // For correlation

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
	case CodeDatabaseError, CodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse is the JSON structure returned to clients.
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Error     Code   `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
	Timestamp string `json:"timestamp"`
}

// NewDatabaseError creates an error for database failures.
// The cause is logged but not exposed to clients.
func NewDatabaseError(component string, operation string, cause error) *APIError {
	return &APIError{
		Code:      CodeDatabaseError,
		Message:   fmt.Sprintf("Failed to %s", operation),
		cause:     cause,
		component: component,
	}
}

// NewNotFoundError creates an error when a resource isn't found.
func NewNotFoundError(component string, resourceType string, resourceID string) *APIError {
	return &APIError{
		Code:      CodeNotFound,
		Message:   fmt.Sprintf("%s '%s' not found", resourceType, resourceID),
		component: component,
	}
}

// NewTimeoutError creates an error for operation timeouts.
func NewTimeoutError(component string, operation string, cause error) *APIError {
	return &APIError{
		Code:      CodeTimeout,
		Message:   fmt.Sprintf("%s timed out", operation),
		cause:     cause,
		component: component,
	}
}

// NewInternalError creates a generic internal error.
// Use sparingly - prefer more specific error types.
func NewInternalError(component string, message string, cause error) *APIError {
	return &APIError{
		Code:      CodeInternalError,
		Message:   message,
		cause:     cause,
		component: component,
	}
}

// NewServiceUnavailableError creates an error when a dependency is down.
func NewServiceUnavailableError(component string, service string, cause error) *APIError {
	return &APIError{
		Code:      CodeServiceUnavailable,
		Message:   fmt.Sprintf("%s is currently unavailable", service),
		cause:     cause,
		component: component,
	}
}

// NewValidationError creates an error for invalid input.
func NewValidationError(component string, operation string, cause error) *APIError {
	msg := fmt.Sprintf("Failed to %s", operation)
	if cause != nil {
		msg = fmt.Sprintf("Failed to %s: %v", operation, cause)
	}
	return &APIError{
		Code:      CodeValidation,
		Message:   msg,
		cause:     cause,
		component: component,
	}
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
	// Generate a simple request ID for correlation
	// In production, this would come from middleware
	requestID := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	apiErr.RequestID = requestID

	// Log full error details for debugging
	log.Printf("[ERROR] request=%s component=%s code=%s message=%q cause=%v",
		requestID, apiErr.component, apiErr.Code, apiErr.Message, apiErr.cause)

	// Send safe response to client
	response := ErrorResponse{
		Success:   false,
		Error:     apiErr.Code,
		Message:   apiErr.Message,
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode())
	json.NewEncoder(w).Encode(response)
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
