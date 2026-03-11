// DOC: docs/internal/SEAMS.md#error-semantics
//
// Package errors provides structured error types with clear categories,
// recovery paths, and observability for both users and automated systems.
//
// ╔════════════════════════════════════════════════════════════════════╗
// ║  ERROR CATEGORIES - Read before modifying                          ║
// ║                                                                    ║
// ║  Each category has a specific recovery path. Changing these        ║
// ║  affects UI error states and automated retry logic.                ║
// ║                                                                    ║
// ║  To add a new category:                                            ║
// ║  1. Ensure it has a distinct recovery path                         ║
// ║  2. Update HTTP status mapping in ToHTTPStatus()                   ║
// ║  3. Update client error handling to recognize new codes            ║
// ╚════════════════════════════════════════════════════════════════════╝
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Category classifies errors by their recovery strategy.
// Each category maps to specific HTTP status codes and client behavior.
type Category string

const (
	// CategoryValidation indicates user input failed validation.
	// Recovery: User must correct input and retry.
	// HTTP: 400 Bad Request
	CategoryValidation Category = "validation"

	// CategoryNotFound indicates the requested resource doesn't exist.
	// Recovery: Check identifier, resource may have been deleted.
	// HTTP: 404 Not Found
	CategoryNotFound Category = "not_found"

	// CategoryConflict indicates a conflict with existing state.
	// Recovery: Check for existing resources, use different identifier.
	// HTTP: 409 Conflict
	CategoryConflict Category = "conflict"

	// CategoryDatabase indicates a database operation failed.
	// Recovery: If transient, retry with backoff. If persistent, contact admin.
	// HTTP: 500 (transient) or 503 (unavailable)
	CategoryDatabase Category = "database"

	// CategoryInternal indicates an unexpected internal error.
	// Recovery: Retry may help for transient issues. Log for debugging.
	// HTTP: 500 Internal Server Error
	CategoryInternal Category = "internal"

	// CategoryDependency indicates an external dependency failed.
	// Recovery: Retry with backoff, check dependency health.
	// HTTP: 502 Bad Gateway or 503 Service Unavailable
	CategoryDependency Category = "dependency"
)

// Severity indicates how serious the error is for logging/alerting.
type Severity string

const (
	SeverityLow      Severity = "low"      // Minor issues, info logging
	SeverityMedium   Severity = "medium"   // Notable issues, warn logging
	SeverityHigh     Severity = "high"     // Significant issues, error logging
	SeverityCritical Severity = "critical" // System health affected, alert
)

// Error is a structured error with category, code, and recovery hints.
// It implements the error interface and can be serialized to JSON for API responses.
type Error struct {
	// Category groups errors by recovery strategy
	Category Category `json:"category"`

	// Code is a machine-readable identifier (e.g., "invalid_slug")
	Code string `json:"code"`

	// Message is a human-readable description
	Message string `json:"message"`

	// Details provides additional context (optional)
	Details map[string]interface{} `json:"details,omitempty"`

	// Recovery suggests what the user/agent should do next
	Recovery string `json:"recovery,omitempty"`

	// Transient indicates if the error is likely temporary
	Transient bool `json:"transient,omitempty"`

	// Severity for logging/alerting purposes (not exposed in API)
	Severity Severity `json:"-"`

	// Cause is the underlying error (not exposed in API)
	Cause error `json:"-"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetails adds context to the error.
func (e *Error) WithDetails(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause wraps an underlying error.
func (e *Error) WithCause(err error) *Error {
	e.Cause = err
	return e
}

// ToHTTPStatus returns the appropriate HTTP status code for this error.
func (e *Error) ToHTTPStatus() int {
	switch e.Category {
	case CategoryValidation:
		return http.StatusBadRequest
	case CategoryNotFound:
		return http.StatusNotFound
	case CategoryConflict:
		return http.StatusConflict
	case CategoryDatabase:
		if e.Transient {
			return http.StatusServiceUnavailable
		}
		return http.StatusInternalServerError
	case CategoryDependency:
		if e.Transient {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadGateway
	case CategoryInternal:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}

// ToJSON returns the error as a JSON-encoded response body.
func (e *Error) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"error": e,
	})
}

// IsTransient returns true if the error might resolve on retry.
func (e *Error) IsTransient() bool {
	return e.Transient
}

// ─────────────────────────────────────────────────────────────────────────────
// Error Constructors - Use these instead of creating errors directly
// ─────────────────────────────────────────────────────────────────────────────

// Validation creates a validation error for invalid user input.
func Validation(code, message string) *Error {
	return &Error{
		Category: CategoryValidation,
		Code:     code,
		Message:  message,
		Recovery: "Check your input and try again",
		Severity: SeverityLow,
	}
}

// NotFound creates an error for missing resources.
func NotFound(resourceType, identifier string) *Error {
	return &Error{
		Category: CategoryNotFound,
		Code:     "not_found",
		Message:  fmt.Sprintf("%s not found", resourceType),
		Details: map[string]interface{}{
			"resource": resourceType,
			"id":       identifier,
		},
		Recovery: "Check the identifier or refresh to get the latest data",
		Severity: SeverityLow,
	}
}

// Conflict creates an error for state conflicts.
func Conflict(code, message string) *Error {
	return &Error{
		Category: CategoryConflict,
		Code:     code,
		Message:  message,
		Recovery: "Check for existing resources with this identifier",
		Severity: SeverityLow,
	}
}

// Database creates an error for database operations.
func Database(message string, transient bool) *Error {
	e := &Error{
		Category:  CategoryDatabase,
		Code:      "database_error",
		Message:   message,
		Transient: transient,
		Severity:  SeverityHigh,
	}
	if transient {
		e.Recovery = "Please try again in a moment"
	} else {
		e.Recovery = "Please contact support if this persists"
	}
	return e
}

// Internal creates an error for unexpected internal failures.
func Internal(message string) *Error {
	return &Error{
		Category: CategoryInternal,
		Code:     "internal_error",
		Message:  message,
		Recovery: "Please try again. If this persists, contact support",
		Severity: SeverityCritical,
	}
}

// Dependency creates an error for external dependency failures.
func Dependency(service, message string, transient bool) *Error {
	e := &Error{
		Category:  CategoryDependency,
		Code:      "dependency_error",
		Message:   message,
		Transient: transient,
		Details: map[string]interface{}{
			"service": service,
		},
		Severity: SeverityHigh,
	}
	if transient {
		e.Recovery = "Please try again in a moment"
	} else {
		e.Recovery = "The service is currently unavailable"
	}
	return e
}

// ─────────────────────────────────────────────────────────────────────────────
// Domain-Specific Errors - Pre-defined errors for common cases
// ─────────────────────────────────────────────────────────────────────────────

// InvalidSlug returns a validation error for invalid slug format.
func InvalidSlug(slug string, minLen, maxLen int) *Error {
	return Validation("invalid_slug", "Slug must contain only lowercase letters, numbers, and hyphens").
		WithDetails("provided", slug).
		WithDetails("min_length", minLen).
		WithDetails("max_length", maxLen)
}

// SlugExists returns a conflict error for duplicate slugs.
func SlugExists(slug string) *Error {
	return Conflict("slug_exists", "A reference with this slug already exists").
		WithDetails("slug", slug)
}

// PathNotExists returns a validation error for non-existent paths.
func PathNotExists(path string) *Error {
	return Validation("path_not_exists", "The specified path does not exist").
		WithDetails("path", path)
}

// ReferenceNotFound returns a not-found error for references.
func ReferenceNotFound(id string) *Error {
	return NotFound("reference", id)
}

// InvalidRequestBody returns a validation error for malformed JSON.
func InvalidRequestBody(parseErr string) *Error {
	return Validation("invalid_request_body", "Could not parse request body").
		WithDetails("parse_error", parseErr)
}
