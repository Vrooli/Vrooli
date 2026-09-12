package shared

import (
	"encoding/json"
	"net/http"
)

// JSONError writes a JSON error response with the given status code and message.
// This centralizes error response formatting across all handlers.
func JSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// JSONSuccess writes a JSON success response with the given status code and data.
// This centralizes success response formatting across all handlers.
func JSONSuccess(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// JSONOK writes a JSON success response with status 200 OK.
func JSONOK(w http.ResponseWriter, data interface{}) {
	JSONSuccess(w, data, http.StatusOK)
}

// JSONCreated writes a JSON success response with status 201 Created.
func JSONCreated(w http.ResponseWriter, data interface{}) {
	JSONSuccess(w, data, http.StatusCreated)
}

// JSONErrorCode writes a structured JSON error payload at the given status code.
// Used when callers need to discriminate by a code field (e.g. 409
// "release_in_flight") rather than a plain message string.
func JSONErrorCode(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
