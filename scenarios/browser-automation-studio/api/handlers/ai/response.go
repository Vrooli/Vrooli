package ai

import (
	"encoding/json"
	"net/http"
)

// APIError represents a structured API error response
// This is a duplicate of handlers.APIError to avoid import cycles
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

// Common errors
var (
	ErrInvalidRequest = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_REQUEST",
		Message: "Invalid request body or parameters",
	}

	ErrMissingRequiredField = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "MISSING_REQUIRED_FIELD",
		Message: "Required field is missing",
	}

	ErrInternalServer = &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "An internal server error occurred",
	}
)

// RespondError writes a JSON error response
func RespondError(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

// RespondSuccess writes a JSON success response
func RespondSuccess(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
