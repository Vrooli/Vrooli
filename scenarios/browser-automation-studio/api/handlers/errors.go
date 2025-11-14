package handlers

import (
	"encoding/json"
	"net/http"
)

// APIError represents a structured API error response
type APIError struct {
	Status  int    `json:"-"`                 // HTTP status code (not serialized)
	Code    string `json:"code"`              // Machine-readable error code
	Message string `json:"message"`           // Human-readable error message
	Details any    `json:"details,omitempty"` // Optional additional context
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// respondError writes a JSON error response with the appropriate status code
func (h *Handler) respondError(w http.ResponseWriter, err *APIError) {
	RespondError(w, err)
}

// respondSuccess writes a JSON success response
func (h *Handler) respondSuccess(w http.ResponseWriter, statusCode int, data any) {
	RespondSuccess(w, statusCode, data)
}

// RespondError writes a JSON error response (exported for use in subpackages)
func RespondError(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	if encodeErr := json.NewEncoder(w).Encode(err); encodeErr != nil {
		// Can't log here as we don't have access to logger
		// Subpackages should handle logging before calling this
	}
}

// RespondSuccess writes a JSON success response (exported for use in subpackages)
func RespondSuccess(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Predefined common errors
var (
	// 400 Bad Request errors
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

	// 404 Not Found errors
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

	// 409 Conflict errors
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

	// 500 Internal Server Error
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

	// 502 Bad Gateway (external service errors)
	ErrAIServiceError = &APIError{
		Status:  http.StatusBadGateway,
		Code:    "AI_SERVICE_ERROR",
		Message: "AI service request failed",
	}

	// 503 Service Unavailable
	ErrServiceUnavailable = &APIError{
		Status:  http.StatusServiceUnavailable,
		Code:    "SERVICE_UNAVAILABLE",
		Message: "Service temporarily unavailable",
	}

	// 413 Request Entity Too Large
	ErrRequestTooLarge = &APIError{
		Status:  http.StatusRequestEntityTooLarge,
		Code:    "REQUEST_TOO_LARGE",
		Message: "Request entity too large",
	}

	// 408 Request Timeout
	ErrRequestTimeout = &APIError{
		Status:  http.StatusRequestTimeout,
		Code:    "REQUEST_TIMEOUT",
		Message: "Request timeout",
	}
)

// WithDetails returns a copy of the error with additional details
func (e *APIError) WithDetails(details any) *APIError {
	return &APIError{
		Status:  e.Status,
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}

// WithMessage returns a copy of the error with a custom message
func (e *APIError) WithMessage(message string) *APIError {
	return &APIError{
		Status:  e.Status,
		Code:    e.Code,
		Message: message,
		Details: e.Details,
	}
}
