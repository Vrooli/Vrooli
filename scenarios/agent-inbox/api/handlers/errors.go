// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains error response utilities for consistent API error formatting.
//
// Error Response Design:
//   - All errors return a consistent JSON structure with machine-readable codes
//   - HTTP status codes are derived from error categories
//   - Recovery hints guide both users and automated agents
//   - Request IDs enable correlation between client errors and server logs
package handlers

import (
	"net/http"

	"agent-inbox/domain"
	"agent-inbox/middleware"
)

// APIErrorResponse is the standard error response format.
// This structure is designed for both human and machine consumption.
type APIErrorResponse struct {
	// Error contains the structured error details.
	Error APIErrorDetail `json:"error"`

	// RequestID allows correlation with server logs.
	RequestID string `json:"request_id,omitempty"`
}

// APIErrorDetail contains the error specifics.
type APIErrorDetail struct {
	// Code is a machine-readable error identifier (e.g., "V001", "N001").
	Code string `json:"code"`

	// Category groups the error type (validation, not_found, dependency, etc.).
	Category string `json:"category"`

	// Message is a user-friendly error description.
	Message string `json:"message"`

	// Recovery suggests what action to take next.
	Recovery string `json:"recovery"`

	// Details provides additional context when available.
	Details map[string]interface{} `json:"details,omitempty"`
}

// WriteAppError writes a structured AppError as an HTTP response.
// It maps the error category to an appropriate HTTP status code.
func (h *Handlers) WriteAppError(w http.ResponseWriter, r *http.Request, err *domain.AppError) {
	status := domain.CategoryToHTTPStatus(err.Category)

	response := APIErrorResponse{
		Error: APIErrorDetail{
			Code:     string(err.Code),
			Category: string(err.Category),
			Message:  err.Message,
			Recovery: string(err.Recovery),
			Details:  err.Details,
		},
		RequestID: GetRequestID(r),
	}

	h.JSONResponse(w, response, status)
}

// WriteError handles any error, converting it to a structured response.
// It checks if the error is already an AppError; if not, wraps it as internal.
func (h *Handlers) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if appErr, ok := err.(*domain.AppError); ok {
		h.WriteAppError(w, r, appErr)
		return
	}

	// Wrap unknown errors as internal errors
	internalErr := domain.ErrInternal("an unexpected error occurred", err)
	h.WriteAppError(w, r, internalErr)
}

// WriteValidationError is a convenience method for validation errors.
func (h *Handlers) WriteValidationError(w http.ResponseWriter, r *http.Request, message string) {
	h.WriteAppError(w, r, domain.ErrInvalidInput(message))
}

// WriteNotFoundError is a convenience method for not-found errors.
func (h *Handlers) WriteNotFoundError(w http.ResponseWriter, r *http.Request, resource, id string) {
	err := domain.NewError(
		domain.ErrCodeChatNotFound, // Will be overridden below for other resources
		domain.CategoryNotFound,
		resource+" not found",
		domain.ActionVerifyResource,
	).WithDetail(resource+"_id", id)

	// Use specific code based on resource type
	switch resource {
	case "chat":
		err.Code = domain.ErrCodeChatNotFound
	case "label":
		err.Code = domain.ErrCodeLabelNotFound
	case "message":
		err.Code = domain.ErrCodeMessageNotFound
	case "tool":
		err.Code = domain.ErrCodeToolNotFound
	}

	h.WriteAppError(w, r, err)
}

// WriteDependencyError is a convenience method for external service errors.
func (h *Handlers) WriteDependencyError(w http.ResponseWriter, r *http.Request, service string, err error) {
	appErr := domain.NewError(
		domain.ErrCodeDatabaseQueryFailed,
		domain.CategoryDependency,
		service+" temporarily unavailable",
		domain.ActionRetryWithBackoff,
	).WithCause(err).WithDetail("service", service)

	h.WriteAppError(w, r, appErr)
}

// GetRequestID retrieves the request ID from the request context.
// Returns empty string if not set.
func GetRequestID(r *http.Request) string {
	return middleware.GetRequestID(r.Context())
}
