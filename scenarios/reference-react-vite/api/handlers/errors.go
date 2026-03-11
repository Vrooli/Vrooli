// Package handlers provides HTTP request handlers for the reference scenario.
// Handlers orchestrate request processing, delegate business logic to domain
// packages, and use repository interfaces for data access.
//
// DOC: docs/concepts/ARCHITECTURE.md#presentation-layer
// DOC: docs/reference/api-endpoints.md
// DOC: docs/internal/SEAMS.md#http-handler-seam
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// APIError represents a standardized error response format.
// All API endpoints return errors in this consistent shape.
type APIError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// Error codes for machine-readable error identification.
const (
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
)

// writeError writes a standardized error response.
func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string, details map[string]interface{}) {
	apiErr := APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: getRequestID(r),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(apiErr)
}

// writeBadRequest writes a 400 error response.
func writeBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	writeError(w, r, http.StatusBadRequest, ErrCodeBadRequest, message, nil)
}

// writeValidationError writes a 422 error response with validation details.
func writeValidationError(w http.ResponseWriter, r *http.Request, message string, details map[string]interface{}) {
	writeError(w, r, http.StatusUnprocessableEntity, ErrCodeValidation, message, details)
}

// writeNotFound writes a 404 error response.
func writeNotFound(w http.ResponseWriter, r *http.Request, resource string) {
	writeError(w, r, http.StatusNotFound, ErrCodeNotFound, resource+" not found", nil)
}

// writeInternalError writes a 500 error response.
func writeInternalError(w http.ResponseWriter, r *http.Request, message string) {
	writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, message, nil)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// getRequestID extracts or generates a request ID.
func getRequestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return uuid.New().String()
}

// PaginationMeta provides pagination information in list responses.
type PaginationMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// ListResponse wraps list data with pagination metadata.
type ListResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// newListResponse creates a list response with pagination.
func newListResponse(data interface{}, total, limit, offset int) ListResponse {
	return ListResponse{
		Data: data,
		Pagination: PaginationMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}
}
