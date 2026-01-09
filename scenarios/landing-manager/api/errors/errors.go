// Package errors provides standardized error types and handling for landing-manager.
// It enables consistent error responses, user-friendly messages, and observability.
package errors

import (
	"fmt"
	"net/http"
)

// ============================================================================
// Error Code Definitions
// ============================================================================
// Error codes are organized by HTTP status code category to make the
// code-to-status mapping explicit and discoverable.

// ErrorCode identifies the type of error for programmatic handling
type ErrorCode string

const (
	// Client errors (4xx) - problems with the request that the client can fix
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"      // 400 Bad Request
	ErrCodeMissingRequired  ErrorCode = "MISSING_REQUIRED"   // 400 Bad Request
	ErrCodeInvalidFormat    ErrorCode = "INVALID_FORMAT"     // 400 Bad Request
	ErrCodeScenarioNotFound ErrorCode = "SCENARIO_NOT_FOUND" // 404 Not Found
	ErrCodeTemplateNotFound ErrorCode = "TEMPLATE_NOT_FOUND" // 404 Not Found
	ErrCodeConflict         ErrorCode = "CONFLICT"           // 409 Conflict

	// External dependency errors (502 Bad Gateway) - upstream service failures
	ErrCodeIssueTrackerError ErrorCode = "ISSUE_TRACKER"     // 502 Bad Gateway
	ErrCodeExternalService   ErrorCode = "EXTERNAL_SERVICE"  // 502 Bad Gateway

	// Server errors (5xx) - problems on our side
	ErrCodeDatabaseError    ErrorCode = "DATABASE_ERROR"    // 500 Internal Server Error
	ErrCodeFileSystemError  ErrorCode = "FILESYSTEM_ERROR"  // 500 Internal Server Error
	ErrCodeCommandExecError ErrorCode = "COMMAND_EXEC"      // 500 Internal Server Error
	ErrCodeLifecycleError   ErrorCode = "LIFECYCLE_ERROR"   // 500 Internal Server Error
	ErrCodeGenerationError  ErrorCode = "GENERATION_ERROR"  // 500 Internal Server Error
	ErrCodeTemplateError    ErrorCode = "TEMPLATE_ERROR"    // 500 Internal Server Error
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"    // 500 Internal Server Error
)

// ============================================================================
// HTTP Status Code Mapping Decision
// ============================================================================
// This map defines the canonical mapping from error codes to HTTP status codes.
// The decision logic is centralized here rather than scattered in handlers.

// errorCodeToHTTPStatus maps error codes to HTTP status codes.
// Decision: Each error code has exactly one HTTP status code.
var errorCodeToHTTPStatus = map[ErrorCode]int{
	// 400 Bad Request - client provided invalid input
	ErrCodeInvalidInput:    http.StatusBadRequest,
	ErrCodeMissingRequired: http.StatusBadRequest,
	ErrCodeInvalidFormat:   http.StatusBadRequest,

	// 404 Not Found - requested resource doesn't exist
	ErrCodeScenarioNotFound: http.StatusNotFound,
	ErrCodeTemplateNotFound: http.StatusNotFound,

	// 409 Conflict - request conflicts with current state
	ErrCodeConflict: http.StatusConflict,

	// 502 Bad Gateway - external service failed
	ErrCodeIssueTrackerError: http.StatusBadGateway,
	ErrCodeExternalService:   http.StatusBadGateway,

	// 500 Internal Server Error - server-side problems
	ErrCodeDatabaseError:    http.StatusInternalServerError,
	ErrCodeFileSystemError:  http.StatusInternalServerError,
	ErrCodeCommandExecError: http.StatusInternalServerError,
	ErrCodeLifecycleError:   http.StatusInternalServerError,
	ErrCodeGenerationError:  http.StatusInternalServerError,
	ErrCodeTemplateError:    http.StatusInternalServerError,
	ErrCodeInternal:         http.StatusInternalServerError,
}

// HTTPStatus returns the HTTP status code for this error.
// Decision: Unknown error codes default to 500 Internal Server Error.
func (e *AppError) HTTPStatus() int {
	if status, ok := errorCodeToHTTPStatus[e.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// IsClientError returns true if this is a 4xx client error.
// Decision: Client errors are recoverable by fixing the request.
func (e *AppError) IsClientError() bool {
	status := e.HTTPStatus()
	return status >= 400 && status < 500
}

// IsServerError returns true if this is a 5xx server error.
// Decision: Server errors are not recoverable by the client.
func (e *AppError) IsServerError() bool {
	return e.HTTPStatus() >= 500
}

// AppError is a structured error with code, user-friendly message, and details
type AppError struct {
	// Code identifies the error type for programmatic handling
	Code ErrorCode `json:"code"`
	// Message is a user-friendly description of what went wrong
	Message string `json:"message"`
	// Details contains additional context (may include technical info)
	Details string `json:"details,omitempty"`
	// Recoverable indicates if the user can retry or take action to fix
	Recoverable bool `json:"recoverable"`
	// Suggestion provides guidance on what the user can do next
	Suggestion string `json:"suggestion,omitempty"`
	// Cause is the underlying error (not serialized to client)
	Cause error `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error chain inspection
func (e *AppError) Unwrap() error {
	return e.Cause
}

// IsRecoverable returns true if the error can be resolved by user action
func (e *AppError) IsRecoverable() bool {
	return e.Recoverable
}

// ToResponse returns a map suitable for JSON response
func (e *AppError) ToResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"success":     false,
		"error_code":  e.Code,
		"message":     e.Message,
		"recoverable": e.Recoverable,
	}
	if e.Details != "" {
		resp["details"] = e.Details
	}
	if e.Suggestion != "" {
		resp["suggestion"] = e.Suggestion
	}
	return resp
}

// Constructor functions for common error types

// NewValidationError creates an error for invalid user input
func NewValidationError(field, message string) *AppError {
	return &AppError{
		Code:        ErrCodeInvalidInput,
		Message:     message,
		Details:     fmt.Sprintf("Field: %s", field),
		Recoverable: true,
		Suggestion:  "Please check your input and try again",
	}
}

// NewMissingFieldError creates an error for required fields
func NewMissingFieldError(field string) *AppError {
	return &AppError{
		Code:        ErrCodeMissingRequired,
		Message:     fmt.Sprintf("Required field '%s' is missing", field),
		Recoverable: true,
		Suggestion:  fmt.Sprintf("Please provide a value for '%s'", field),
	}
}

// NewNotFoundError creates an error when a resource doesn't exist
func NewNotFoundError(resourceType, identifier string) *AppError {
	code := ErrCodeScenarioNotFound
	if resourceType == "template" {
		code = ErrCodeTemplateNotFound
	}
	return &AppError{
		Code:        code,
		Message:     fmt.Sprintf("%s '%s' not found", resourceType, identifier),
		Recoverable: false,
		Suggestion:  fmt.Sprintf("Verify the %s exists and the identifier is correct", resourceType),
	}
}

// NewConflictError creates an error when there's a resource conflict
func NewConflictError(resource, message string) *AppError {
	return &AppError{
		Code:        ErrCodeConflict,
		Message:     message,
		Details:     fmt.Sprintf("Resource: %s", resource),
		Recoverable: true,
		Suggestion:  "Resolve the conflict and try again",
	}
}

// NewDatabaseError creates an error for database operations
func NewDatabaseError(operation string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeDatabaseError,
		Message:     "A database error occurred",
		Details:     fmt.Sprintf("Operation: %s", operation),
		Recoverable: true,
		Suggestion:  "Try again in a few moments. If the problem persists, check database connectivity.",
		Cause:       cause,
	}
}

// NewExternalServiceError creates an error for external service failures
func NewExternalServiceError(service string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeExternalService,
		Message:     fmt.Sprintf("Failed to communicate with %s", service),
		Recoverable: true,
		Suggestion:  fmt.Sprintf("Check if %s is running and try again", service),
		Cause:       cause,
	}
}

// NewFileSystemError creates an error for file system operations
func NewFileSystemError(operation, path string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeFileSystemError,
		Message:     fmt.Sprintf("File system operation failed: %s", operation),
		Details:     fmt.Sprintf("Path: %s", sanitizePath(path)),
		Recoverable: false,
		Suggestion:  "Check file permissions and disk space",
		Cause:       cause,
	}
}

// NewCommandError creates an error for CLI command execution
func NewCommandError(command, output string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeCommandExecError,
		Message:     "Command execution failed",
		Details:     output,
		Recoverable: true,
		Suggestion:  "Check the scenario configuration and try again",
		Cause:       cause,
	}
}

// NewLifecycleError creates an error for scenario lifecycle operations
func NewLifecycleError(operation, scenarioID string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeLifecycleError,
		Message:     fmt.Sprintf("Failed to %s scenario", operation),
		Details:     fmt.Sprintf("Scenario: %s", scenarioID),
		Recoverable: true,
		Suggestion:  fmt.Sprintf("Try stopping the scenario first, then %s again", operation),
		Cause:       cause,
	}
}

// NewGenerationError creates an error for scenario generation
func NewGenerationError(templateID, slug string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeGenerationError,
		Message:     "Failed to generate scenario",
		Details:     fmt.Sprintf("Template: %s, Slug: %s", templateID, slug),
		Recoverable: true,
		Suggestion:  "Check the template configuration and try again with a different slug if needed",
		Cause:       cause,
	}
}

// NewIssueTrackerError creates an error for issue tracker communication
func NewIssueTrackerError(operation string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeIssueTrackerError,
		Message:     "Issue tracker communication failed",
		Details:     fmt.Sprintf("Operation: %s", operation),
		Recoverable: true,
		Suggestion:  "Check if app-issue-tracker is running and configured",
		Cause:       cause,
	}
}

// NewInternalError creates a generic internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Code:        ErrCodeInternal,
		Message:     message,
		Recoverable: false,
		Suggestion:  "Please try again later or contact support if the problem persists",
		Cause:       cause,
	}
}

// sanitizePath removes sensitive path components for safe logging
func sanitizePath(path string) string {
	// This is a simple implementation - could be enhanced
	// to match util.SanitizeCommandOutput patterns
	return path
}

// IsNotFound checks if an error is a not-found error
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrCodeScenarioNotFound || appErr.Code == ErrCodeTemplateNotFound
	}
	return false
}

// IsValidation checks if an error is a validation error
func IsValidation(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrCodeInvalidInput ||
			appErr.Code == ErrCodeMissingRequired ||
			appErr.Code == ErrCodeInvalidFormat
	}
	return false
}

// IsExternalDependency checks if an error is from an external dependency
func IsExternalDependency(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrCodeDatabaseError ||
			appErr.Code == ErrCodeExternalService ||
			appErr.Code == ErrCodeIssueTrackerError ||
			appErr.Code == ErrCodeCommandExecError
	}
	return false
}
