// Package handlers provides HTTP request handlers for the reference scenario.
// Handlers orchestrate request processing, delegate business logic to domain
// packages, and use repository interfaces for data access.
//
// DOC: docs/concepts/ARCHITECTURE.md#presentation-layer
// DOC: docs/reference/api-endpoints.md
// DOC: docs/internal/SEAMS.md#http-handler-seam
// DOC: docs/internal/ERROR_SEMANTICS.md
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// ╔════════════════════════════════════════════════════════════════════════════╗
// ║  ERROR CATEGORIES - Read before modifying                                  ║
// ║                                                                            ║
// ║  Each category has a specific recovery path. Changing these affects UI     ║
// ║  error states and automated retry logic.                                   ║
// ║                                                                            ║
// ║  Categories by recovery path:                                              ║
// ║  - BAD_REQUEST: User must fix request format (check API docs)              ║
// ║  - VALIDATION_ERROR: User must fix input values (field-specific feedback)  ║
// ║  - NOT_FOUND: Resource missing (verify ID, list available resources)       ║
// ║  - INTERNAL_ERROR: System issue (retry with backoff, check service health) ║
// ║  - CONFLICT: State conflict (refresh resource, resolve conflict)           ║
// ║  - UNAUTHORIZED: Auth issue (login/refresh token)                          ║
// ║                                                                            ║
// ║  To add a new category:                                                    ║
// ║  1. Ensure it has a distinct recovery path                                 ║
// ║  2. Add recovery hint in errorRecoveryHints                                ║
// ║  3. Document in docs/internal/ERROR_SEMANTICS.md                           ║
// ╚════════════════════════════════════════════════════════════════════════════╝

// APIError represents a standardized error response format.
// All API endpoints return errors in this consistent shape.
//
// Fields:
//   - Code: Machine-readable error category for programmatic handling
//   - Message: Human-readable error description
//   - Details: Additional context (e.g., field-level validation errors)
//   - Recovery: Suggested action for resolving the error
//   - Retryable: Whether the operation can be retried
//   - RequestID: Correlation ID for debugging
type APIError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Recovery  string                 `json:"recovery,omitempty"`
	Retryable bool                   `json:"retryable"`
	RequestID string                 `json:"request_id,omitempty"`
}

// Error codes for machine-readable error identification.
// Each code maps to a specific HTTP status and recovery path.
const (
	ErrCodeBadRequest   = "BAD_REQUEST"     // 400 - Malformed request syntax
	ErrCodeNotFound     = "NOT_FOUND"       // 404 - Resource doesn't exist
	ErrCodeValidation   = "VALIDATION_ERROR" // 422 - Invalid field values
	ErrCodeInternal     = "INTERNAL_ERROR"  // 500 - Server-side failure
	ErrCodeConflict     = "CONFLICT"        // 409 - State conflict
	ErrCodeUnauthorized = "UNAUTHORIZED"    // 401 - Authentication required
)

// errorRecoveryHints maps error codes to user-actionable recovery guidance.
// These hints help users and agents understand what to do next.
var errorRecoveryHints = map[string]string{
	ErrCodeBadRequest:   "Check the request format and ensure JSON is valid. See API documentation for expected structure.",
	ErrCodeNotFound:     "Verify the resource ID is correct. Use the list endpoint to find available resources.",
	ErrCodeValidation:   "Review the field values in 'details' and correct invalid inputs.",
	ErrCodeInternal:     "This is a temporary server issue. Please retry after a short delay.",
	ErrCodeConflict:     "The resource was modified. Refresh and try again with the latest version.",
	ErrCodeUnauthorized: "Authentication required. Please login or refresh your session.",
}

// retryableErrors indicates which error codes represent transient failures.
var retryableErrors = map[string]bool{
	ErrCodeInternal: true, // Database timeouts, temporary unavailability
}

// writeError writes a standardized error response with recovery guidance.
// It logs the error for observability and includes correlation metadata.
func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string, details map[string]interface{}) {
	requestID := getRequestID(r)

	// Log structured error for observability
	// Format: [ERROR] request_id=X code=Y status=Z path=W message=M
	log.Printf("[ERROR] request_id=%s code=%s status=%d path=%s message=%q",
		requestID, code, statusCode, r.URL.Path, message)

	apiErr := APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		Recovery:  errorRecoveryHints[code],
		Retryable: retryableErrors[code],
		RequestID: requestID,
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
