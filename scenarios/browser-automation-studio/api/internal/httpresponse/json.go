// Package httpresponse owns common HTTP JSON response side effects.
package httpresponse

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes a JSON response with the supplied status. onEncodeError is
// optional so each route can retain its established logging policy.
func WriteJSON(w http.ResponseWriter, status int, value any, onEncodeError func(error)) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && onEncodeError != nil {
		onEncodeError(err)
	}
}

// WriteError preserves the legacy import API error envelope while sharing its
// response writer across caller packages.
func WriteError(w http.ResponseWriter, status int, message string, onEncodeError func(error)) {
	WriteJSON(w, status, map[string]any{"error": true, "message": message}, onEncodeError)
}
