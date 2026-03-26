// Package apierr provides structured, categorized API errors with recovery hints.
//
// Each error carries a machine-readable Code, a human-readable Message, and an
// optional Recovery hint that tells the caller what to do next. Handlers use the
// constructors (Validation, NotFound, Internal, …) instead of ad-hoc strings so
// that clients—human and agent alike—can classify failures and act on them.
//
// DOC: docs/internal/SEAMS.md
package apierr

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Code classifies an API error for programmatic handling.
type Code string

const (
	CodeValidation Code = "validation" // caller input is malformed or missing
	CodeNotFound   Code = "not_found"  // requested resource does not exist
	CodeConflict   Code = "conflict"   // operation conflicts with current state
	CodeInternal   Code = "internal"   // unexpected server-side failure
	CodeDependency Code = "dependency" // an external dependency is unavailable
)

// Error is the structured JSON error returned by all API endpoints.
type Error struct {
	Code     Code   `json:"code"`
	Message  string `json:"message"`
	Recovery string `json:"recovery,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// --- Constructors ---

// Validation returns a 400 error for bad caller input.
func Validation(msg string) *Error {
	return &Error{Code: CodeValidation, Message: msg, Recovery: "Check the request body and try again."}
}

// NotFound returns a 404 error for missing resources.
func NotFound(resource string) *Error {
	return &Error{Code: CodeNotFound, Message: resource + " not found", Recovery: "Verify the ID or name exists."}
}

// Internal returns a 500 error for unexpected failures.
// The detail is logged server-side but NOT exposed to the client.
func Internal(action string, err error) *Error {
	log.Printf("[error] %s: %v", action, err)
	return &Error{Code: CodeInternal, Message: "failed to " + action, Recovery: "Retry the request. If the problem persists, check server logs."}
}

// Conflict returns a 409 error when the operation conflicts with the current state.
// Used for optimistic locking failures when a resource has been modified since it was read.
func Conflict(msg string) *Error {
	return &Error{Code: CodeConflict, Message: msg, Recovery: "Re-read the resource and retry with the current version."}
}

// Dependency returns a 503 error when an external system is unavailable.
func Dependency(system string, err error) *Error {
	log.Printf("[error] dependency %s: %v", system, err)
	return &Error{Code: CodeDependency, Message: system + " is unavailable", Recovery: "Wait a moment and retry. The dependency may be temporarily down."}
}

// --- HTTP helpers ---

// StatusCode maps an error Code to its HTTP status.
func (e *Error) StatusCode() int {
	switch e.Code {
	case CodeValidation:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeDependency:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Write sends the structured error as a JSON response.
func Write(w http.ResponseWriter, apiErr *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode())
	json.NewEncoder(w).Encode(apiErr)
}
