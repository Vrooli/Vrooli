// DOC: docs/internal/ERROR_SEMANTICS.md
//
// Package errors defines structured error types for the lifestyle dashboard.
// ╔════════════════════════════════════════════════════════════════════════════╗
// ║  ERROR CATEGORIES - Read before modifying                                   ║
// ║                                                                             ║
// ║  Each category has a specific recovery path. Changing these affects         ║
// ║  UI error states and automated retry logic.                                 ║
// ║                                                                             ║
// ║  Categories:                                                                ║
// ║  - validation:   User can fix input (400)                                   ║
// ║  - not_found:    Resource doesn't exist (404)                               ║
// ║  - conflict:     Resource state prevents operation (409)                    ║
// ║  - internal:     Server error, retry might help (500)                       ║
// ║  - unavailable:  Dependency down, retry later (503)                         ║
// ║                                                                             ║
// ║  To add a new category:                                                     ║
// ║  1. Ensure it has a distinct recovery path                                  ║
// ║  2. Update WriteAPIError in handlers                                        ║
// ║  3. Update UI error handling                                                ║
// ╚════════════════════════════════════════════════════════════════════════════╝
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorCategory represents the type of error for recovery path selection.
type ErrorCategory string

const (
	// CategoryValidation - user input is invalid, fix and retry
	CategoryValidation ErrorCategory = "validation"

	// CategoryNotFound - resource doesn't exist
	CategoryNotFound ErrorCategory = "not_found"

	// CategoryConflict - resource state prevents operation
	CategoryConflict ErrorCategory = "conflict"

	// CategoryInternal - server error, retry might help
	CategoryInternal ErrorCategory = "internal"

	// CategoryUnavailable - dependency down, retry later
	CategoryUnavailable ErrorCategory = "unavailable"
)

// ErrorCode provides machine-readable error identification.
type ErrorCode string

// Validation error codes
const (
	CodeInvalidJSON      ErrorCode = "INVALID_JSON"
	CodeMissingField     ErrorCode = "MISSING_FIELD"
	CodeInvalidField     ErrorCode = "INVALID_FIELD"
	CodeInvalidTimeRange ErrorCode = "INVALID_TIME_RANGE"
)

// Not found error codes
const (
	CodeEventNotFound  ErrorCode = "EVENT_NOT_FOUND"
	CodeDomainNotFound ErrorCode = "DOMAIN_NOT_FOUND"
)

// Internal error codes
const (
	CodeDatabaseError    ErrorCode = "DATABASE_ERROR"
	CodeHealthCheckError ErrorCode = "HEALTH_CHECK_ERROR"
)

// Unavailable error codes
const (
	CodeDependencyUnavailable ErrorCode = "DEPENDENCY_UNAVAILABLE"
)

// RecoveryHint provides guidance for error recovery.
type RecoveryHint string

const (
	HintFixInput      RecoveryHint = "Check the request body and fix validation errors"
	HintCheckID       RecoveryHint = "Verify the resource ID exists"
	HintRetryLater    RecoveryHint = "Wait a moment and retry the request"
	HintCheckScenario RecoveryHint = "Ensure the scenario is running with 'vrooli scenario status lifestyle-dashboard'"
)

// APIError represents a structured error response.
type APIError struct {
	// IsError is always true for error responses (JSON key: "error")
	IsError bool `json:"error"`

	// Category for recovery path selection
	Category ErrorCategory `json:"category"`

	// Code for machine-readable identification
	Code ErrorCode `json:"code"`

	// Message is human-readable error description
	Message string `json:"message"`

	// Details provides additional context (optional)
	Details map[string]interface{} `json:"details,omitempty"`

	// Recovery provides guidance on how to fix the error
	Recovery RecoveryHint `json:"recovery,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// StatusCode returns the HTTP status code for this error category.
func (e *APIError) StatusCode() int {
	switch e.Category {
	case CategoryValidation:
		return http.StatusBadRequest
	case CategoryNotFound:
		return http.StatusNotFound
	case CategoryConflict:
		return http.StatusConflict
	case CategoryInternal:
		return http.StatusInternalServerError
	case CategoryUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// ToJSON returns the JSON representation of the error.
func (e *APIError) ToJSON() []byte {
	data, _ := json.Marshal(e)
	return data
}

// NewValidationError creates a validation category error.
func NewValidationError(code ErrorCode, message string) *APIError {
	return &APIError{
		IsError:  true,
		Category: CategoryValidation,
		Code:     code,
		Message:  message,
		Recovery: HintFixInput,
	}
}

// NewNotFoundError creates a not_found category error.
func NewNotFoundError(code ErrorCode, entity, id string) *APIError {
	return &APIError{
		IsError:  true,
		Category: CategoryNotFound,
		Code:     code,
		Message:  fmt.Sprintf("%s not found: %s", entity, id),
		Details: map[string]interface{}{
			"entity": entity,
			"id":     id,
		},
		Recovery: HintCheckID,
	}
}

// NewInternalError creates an internal category error.
// Note: The detail message is logged but not exposed to clients.
func NewInternalError(code ErrorCode, publicMessage string) *APIError {
	return &APIError{
		IsError:  true,
		Category: CategoryInternal,
		Code:     code,
		Message:  publicMessage,
		Recovery: HintRetryLater,
	}
}

// NewUnavailableError creates an unavailable category error.
func NewUnavailableError(code ErrorCode, message string) *APIError {
	return &APIError{
		IsError:  true,
		Category: CategoryUnavailable,
		Code:     code,
		Message:  message,
		Recovery: HintCheckScenario,
	}
}

// WithDetails adds extra context to the error.
func (e *APIError) WithDetails(key string, value interface{}) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Common pre-built errors for frequent use cases

// ErrInvalidJSON is returned when request body is not valid JSON.
var ErrInvalidJSON = NewValidationError(CodeInvalidJSON, "Request body is not valid JSON")

// ErrMissingDomain is returned when domain field is missing.
var ErrMissingDomain = NewValidationError(CodeMissingField, "Field 'domain' is required").WithDetails("field", "domain")

// ErrMissingEventType is returned when event_type field is missing.
var ErrMissingEventType = NewValidationError(CodeMissingField, "Field 'event_type' is required").WithDetails("field", "event_type")

// ErrMissingName is returned when name field is missing.
var ErrMissingName = NewValidationError(CodeMissingField, "Field 'name' is required").WithDetails("field", "name")

// ErrMissingDisplayName is returned when display_name field is missing.
var ErrMissingDisplayName = NewValidationError(CodeMissingField, "Field 'display_name' is required").WithDetails("field", "display_name")
