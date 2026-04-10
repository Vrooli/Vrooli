package main

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeResourceLoadError writes a standardized 500 error for resource loading failures.
func writeResourceLoadError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error": "failed to load resources: " + err.Error(),
	})
}

// decodeJSONBody reads the request body as JSON into dst.
// Returns true on success, false if decoding failed (error response already written).
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON body: " + err.Error(),
		})
		return false
	}
	return true
}
