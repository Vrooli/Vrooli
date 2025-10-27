package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// writeJSON writes a JSON response with proper headers and error handling.
// This helper ensures consistent Content-Type headers and proper error handling
// across all handlers. Note that once encoding begins, the status code and headers
// are already sent, so encoding errors can only be logged, not changed to error responses.
func writeJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode) // Always set status code explicitly
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// At this point, headers are already sent, so we can only log the error
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeError writes a standardized JSON error response.
// This replaces http.Error to ensure all API errors return consistent JSON format
// instead of plain text, making frontend error handling more predictable.
func writeError(w http.ResponseWriter, message string, statusCode int) {
	writeJSON(w, map[string]any{
		"success": false,
		"error":   message,
	}, statusCode)
}

// decodeJSONBody decodes request body into type T with consistent error handling.
// Returns (pointer to decoded value, true) on success or (nil, false) on error.
// The error response is automatically written to w, so callers should return immediately on false.
func decodeJSONBody[T any](w http.ResponseWriter, r *http.Request) (*T, bool) {
	defer r.Body.Close()
	var result T
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return nil, false
	}
	return &result, true
}
