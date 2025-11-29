// Package apierror provides structured API error types and response utilities.
// This is a cross-cutting concern used by both handlers and handlers/ai packages.
package apierror

import (
	"encoding/json"
	"net/http"
)

// APIError represents a structured API error response.
type APIError struct {
	Status  int    `json:"-"`                 // HTTP status code (not serialized)
	Code    string `json:"code"`              // Machine-readable error code
	Message string `json:"message"`           // Human-readable error message
	Details any    `json:"details,omitempty"` // Optional additional context
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// WithDetails returns a copy of the error with additional details.
func (e *APIError) WithDetails(details any) *APIError {
	return &APIError{
		Status:  e.Status,
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}

// WithMessage returns a copy of the error with a custom message.
func (e *APIError) WithMessage(message string) *APIError {
	return &APIError{
		Status:  e.Status,
		Code:    e.Code,
		Message: message,
		Details: e.Details,
	}
}

// RespondError writes a JSON error response with the appropriate status code.
func RespondError(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

// RespondSuccess writes a JSON success response.
func RespondSuccess(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Common errors - 400 Bad Request
var (
	ErrInvalidRequest = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_REQUEST",
		Message: "Invalid request body or parameters",
	}

	ErrInvalidWorkflowID = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_WORKFLOW_ID",
		Message: "Invalid workflow ID format",
	}

	ErrInvalidProjectID = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_PROJECT_ID",
		Message: "Invalid project ID format",
	}

	ErrInvalidExecutionID = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_EXECUTION_ID",
		Message: "Invalid execution ID format",
	}

	ErrMissingRequiredField = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "MISSING_REQUIRED_FIELD",
		Message: "Required field is missing",
	}

	ErrInvalidWorkflowPayload = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_WORKFLOW_PAYLOAD",
		Message: "Workflow definition payload could not be parsed",
	}

	ErrWorkflowValidationFailed = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "WORKFLOW_VALIDATION_FAILED",
		Message: "Workflow definition failed validation",
	}
)

// Common errors - 404 Not Found
var (
	ErrWorkflowNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "WORKFLOW_NOT_FOUND",
		Message: "Workflow not found",
	}

	ErrProjectNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "PROJECT_NOT_FOUND",
		Message: "Project not found",
	}

	ErrWorkflowVersionNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "WORKFLOW_VERSION_NOT_FOUND",
		Message: "Workflow version not found",
	}

	ErrExecutionNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "EXECUTION_NOT_FOUND",
		Message: "Execution not found",
	}

	ErrScreenshotNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "SCREENSHOT_NOT_FOUND",
		Message: "Screenshot not found",
	}
)

// Common errors - 409 Conflict
var (
	ErrProjectAlreadyExists = &APIError{
		Status:  http.StatusConflict,
		Code:    "PROJECT_ALREADY_EXISTS",
		Message: "A project with this name or folder path already exists",
	}

	ErrWorkflowAlreadyExists = &APIError{
		Status:  http.StatusConflict,
		Code:    "WORKFLOW_ALREADY_EXISTS",
		Message: "A workflow with this name already exists in the target project",
	}

	ErrConflict = &APIError{
		Status:  http.StatusConflict,
		Code:    "WORKFLOW_CONFLICT",
		Message: "The workflow was updated elsewhere. Refresh and retry.",
	}
)

// Common errors - 500 Internal Server Error
var (
	ErrInternalServer = &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "An internal server error occurred",
	}

	ErrDatabaseError = &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "DATABASE_ERROR",
		Message: "Database operation failed",
	}

	ErrWorkflowExecutionFailed = &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "WORKFLOW_EXECUTION_FAILED",
		Message: "Failed to execute workflow",
	}
)

// Common errors - Other status codes
var (
	ErrAIServiceError = &APIError{
		Status:  http.StatusBadGateway,
		Code:    "AI_SERVICE_ERROR",
		Message: "AI service request failed",
	}

	ErrServiceUnavailable = &APIError{
		Status:  http.StatusServiceUnavailable,
		Code:    "SERVICE_UNAVAILABLE",
		Message: "Service temporarily unavailable",
	}

	ErrRequestTooLarge = &APIError{
		Status:  http.StatusRequestEntityTooLarge,
		Code:    "REQUEST_TOO_LARGE",
		Message: "Request entity too large",
	}

	ErrRequestTimeout = &APIError{
		Status:  http.StatusRequestTimeout,
		Code:    "REQUEST_TIMEOUT",
		Message: "Request timeout",
	}
)
