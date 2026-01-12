// Package http provides HTTP utilities for handlers including JSON responses,
// error handling, and common middleware.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"scenario-to-desktop-api/shared/errors"
)

// ErrorResponse represents the JSON structure for error responses.
// This ensures consistent error formatting across all endpoints.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response with optional message.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// WriteJSON writes a JSON response with the given status code.
// It sets the Content-Type header and encodes the payload.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			// Log encoding error but don't try to write another response
			slog.Error("failed to encode JSON response", "error", err)
		}
	}
}

// WriteJSONOK writes a 200 OK JSON response.
func WriteJSONOK(w http.ResponseWriter, payload interface{}) {
	WriteJSON(w, http.StatusOK, payload)
}

// WriteJSONCreated writes a 201 Created JSON response.
func WriteJSONCreated(w http.ResponseWriter, payload interface{}) {
	WriteJSON(w, http.StatusCreated, payload)
}

// WriteJSONAccepted writes a 202 Accepted JSON response (for async operations).
func WriteJSONAccepted(w http.ResponseWriter, payload interface{}) {
	WriteJSON(w, http.StatusAccepted, payload)
}

// WriteJSONNoContent writes a 204 No Content response (no body).
func WriteJSONNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteSuccess writes a simple success response with an optional message.
func WriteSuccess(w http.ResponseWriter, message string) {
	WriteJSONOK(w, SuccessResponse{
		Success: true,
		Message: message,
	})
}

// WriteError writes an error response, automatically mapping DomainErrors to
// appropriate HTTP status codes and formatting.
//
// For DomainErrors, it uses the error's code, message, and details.
// For other errors, it returns a 500 Internal Server Error with a generic message.
func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error: "unknown error",
			Code:  string(errors.CodeInternal),
		})
		return
	}

	if de, ok := errors.IsDomainError(err); ok {
		WriteJSON(w, de.HTTPStatus(), ErrorResponse{
			Error:   de.Message,
			Code:    string(de.Code),
			Details: de.Details,
		})
		return
	}

	// For non-domain errors, return 500 with the error message
	// In production, you might want to hide internal error details
	WriteJSON(w, http.StatusInternalServerError, ErrorResponse{
		Error: err.Error(),
		Code:  string(errors.CodeInternal),
	})
}

// WriteErrorWithStatus writes an error with a specific HTTP status.
// This is useful when you want to override the default status mapping.
func WriteErrorWithStatus(w http.ResponseWriter, status int, err error) {
	if err == nil {
		WriteJSON(w, status, ErrorResponse{
			Error: "unknown error",
		})
		return
	}

	if de, ok := errors.IsDomainError(err); ok {
		WriteJSON(w, status, ErrorResponse{
			Error:   de.Message,
			Code:    string(de.Code),
			Details: de.Details,
		})
		return
	}

	WriteJSON(w, status, ErrorResponse{
		Error: err.Error(),
	})
}

// WriteNotFound writes a 404 Not Found error response.
func WriteNotFound(w http.ResponseWriter, resource string) {
	WriteError(w, errors.ErrNotFound(resource))
}

// WriteBadRequest writes a 400 Bad Request error response.
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, errors.ErrBadRequest(message))
}

// WriteValidationError writes a 422 Unprocessable Entity error with details.
func WriteValidationError(w http.ResponseWriter, message string, details map[string]interface{}) {
	WriteError(w, errors.ErrValidation(message, details))
}

// WriteInternalError writes a 500 Internal Server Error response.
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, errors.ErrInternal(message))
}

// WriteUnavailable writes a 503 Service Unavailable response.
func WriteUnavailable(w http.ResponseWriter, service string) {
	WriteError(w, errors.ErrUnavailable(service))
}

// DecodeJSON decodes a JSON request body into the target struct.
// If decoding fails, it writes an appropriate error response and returns false.
// If successful, returns true and the caller should continue processing.
//
// Usage:
//
//	var req MyRequest
//	if !http.DecodeJSON(w, r, &req) {
//	    return // Error already written
//	}
//	// Use req
func DecodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		WriteBadRequest(w, "invalid JSON format")
		return false
	}
	return true
}

// DecodeJSONStrict is like DecodeJSON but disallows unknown fields.
// Useful for strict API contracts.
func DecodeJSONStrict(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteBadRequest(w, "invalid JSON format or unknown fields")
		return false
	}
	return true
}

// RequireParam checks that a query parameter is present and non-empty.
// If missing, it writes a 400 error and returns false.
//
// Usage:
//
//	buildID := r.URL.Query().Get("build_id")
//	if !http.RequireParam(w, "build_id", buildID) {
//	    return
//	}
func RequireParam(w http.ResponseWriter, name, value string) bool {
	if value == "" {
		WriteBadRequest(w, name+" is required")
		return false
	}
	return true
}

// RequireParams checks that multiple query parameters are present.
// Returns the first missing parameter name, or empty string if all present.
// If any are missing, writes a 400 error and returns false.
func RequireParams(w http.ResponseWriter, params map[string]string) bool {
	for name, value := range params {
		if value == "" {
			WriteBadRequest(w, name+" is required")
			return false
		}
	}
	return true
}
