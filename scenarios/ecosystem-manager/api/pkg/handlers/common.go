package handlers

import (
	"encoding/json"
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
