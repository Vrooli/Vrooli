// Package httputil provides HTTP request/response utilities for API handlers.
package httputil

import (
	"encoding/json"
	"net/http"
)

// APIError represents a structured error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// APIErrorEnvelope wraps an APIError for JSON encoding.
type APIErrorEnvelope struct {
	Error APIError `json:"error"`
}

// WriteAPIError writes a JSON error response with the given status code.
func WriteAPIError(w http.ResponseWriter, status int, apiErr APIError) {
	WriteJSON(w, status, APIErrorEnvelope{Error: apiErr})
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
