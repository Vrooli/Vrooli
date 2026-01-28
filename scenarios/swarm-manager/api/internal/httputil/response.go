// Package httputil provides shared HTTP response utilities for API handlers.
//
// This package consolidates common patterns for:
//   - JSON response writing with proper Content-Type headers
//   - Structured error responses with appropriate HTTP status codes
//   - Path validation for file operations
//
// Design Goals:
//   - Reduce boilerplate in handlers (each handler had 10+ duplicate patterns)
//   - Centralize error message formatting for consistency
//   - Ensure all JSON responses set proper Content-Type headers
//
// Related files:
//   - api/internal/ideas/handler.go - uses these utilities for idea operations
//   - api/internal/scenarios/handler.go - uses these utilities for scenario operations
package httputil

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// JSON writes a JSON response with proper Content-Type header.
// Returns any encoding error, though errors are unlikely for valid Go structs.
func JSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

// JSONWithStatus writes a JSON response with a specific HTTP status code.
func JSONWithStatus(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// Error writes an error response using http.Error.
// This is a thin wrapper for consistent logging patterns.
func Error(w http.ResponseWriter, logPrefix, message string, code int) {
	if logPrefix != "" {
		log.Printf("%s: %s (status=%d)", logPrefix, message, code)
	}
	http.Error(w, message, code)
}

// BadRequest writes a 400 Bad Request response.
func BadRequest(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusBadRequest)
}

// NotFound writes a 404 Not Found response.
func NotFound(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusNotFound)
}

// InternalError writes a 500 Internal Server Error response.
func InternalError(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusInternalServerError)
}

// Conflict writes a 409 Conflict response.
func Conflict(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusConflict)
}

// ServiceUnavailable writes a 503 Service Unavailable response.
func ServiceUnavailable(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusServiceUnavailable)
}

// ValidatePath checks if a file path is safely within a base directory.
// Returns true if the path is valid and within baseDir.
// This prevents path traversal attacks like "../../../etc/passwd".
func ValidatePath(baseDir, relativePath string) bool {
	fullPath := filepath.Join(baseDir, relativePath)
	cleanPath := filepath.Clean(fullPath)
	cleanBase := filepath.Clean(baseDir)

	// Path must start with base directory
	if !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) && cleanPath != cleanBase {
		return false
	}
	return true
}

// SafeFilePath resolves a relative path within a base directory safely.
// Returns the full path and true if valid, empty string and false otherwise.
func SafeFilePath(baseDir, relativePath string) (string, bool) {
	if !ValidatePath(baseDir, relativePath) {
		return "", false
	}
	return filepath.Join(baseDir, relativePath), true
}
