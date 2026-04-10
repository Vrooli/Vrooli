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
//   - api/internal/backlog/handler.go - uses these utilities for backlog operations
//   - api/internal/scenarios/handler.go - uses these utilities for scenario operations
//
// DOC: docs/internal/ERROR-SEMANTICS.md
// DOC: docs/internal/SECURITY-POSTURE.md
// DOC: docs/internal/SEAMS.md
package httputil

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
)

const dryRunHeader = "X-Dry-Run"

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

// TruncateErrorMessage normalizes whitespace and truncates an error message
// for safe inclusion in HTTP responses. Returns "unknown" for nil errors.
func TruncateErrorMessage(err error, maxLen int) string {
	if err == nil {
		return "unknown"
	}
	msg := strings.Join(strings.Fields(strings.TrimSpace(err.Error())), " ")
	if msg == "" {
		return "unknown"
	}
	if maxLen > 0 && len(msg) > maxLen {
		return msg[:maxLen] + "..."
	}
	return msg
}

// IsDryRun reports whether request should validate without mutating state.
func IsDryRun(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get(dryRunHeader)), "true")
}
