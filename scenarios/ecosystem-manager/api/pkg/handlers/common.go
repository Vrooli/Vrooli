package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const defaultMaxBodyBytes int64 = 1 << 20 // 1MB hard cap to prevent oversized payloads

// isDryRun reports whether the request has the X-Dry-Run header set,
// indicating the caller wants validation without mutation.
func isDryRun(r *http.Request) bool {
	return r.Header.Get("X-Dry-Run") == "true"
}

// maxBodyBytes is kept mutable for tests; production uses defaultMaxBodyBytes.
var maxBodyBytes int64 = defaultMaxBodyBytes

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

// ErrorOpts describes a structured API error with recovery guidance.
type ErrorOpts struct {
	Code         string
	Message      string
	RecoveryHint string
	Details      map[string]any
	ManualSteps  []string
}

// writeStructuredError writes a JSON error response with optional structured fields
// (code, recovery_hint, details, manual_steps) that cli-core's ParseAPIError can consume.
func writeStructuredError(w http.ResponseWriter, opts ErrorOpts, statusCode int) {
	payload := map[string]any{
		"success": false,
		"error":   opts.Message,
	}
	if opts.Code != "" {
		payload["code"] = opts.Code
	}
	if opts.RecoveryHint != "" {
		payload["recovery_hint"] = opts.RecoveryHint
	}
	if len(opts.Details) > 0 {
		payload["details"] = opts.Details
	}
	if len(opts.ManualSteps) > 0 {
		payload["manual_steps"] = opts.ManualSteps
	}
	writeJSON(w, payload, statusCode)
}

// decodeJSONBody decodes request body into type T with consistent error handling.
// Returns (pointer to decoded value, true) on success or (nil, false) on error.
// The error response is automatically written to w, so callers should return immediately on false.
func decodeJSONBody[T any](w http.ResponseWriter, r *http.Request) (*T, bool) {
	defer r.Body.Close() // Best-effort close; MaxBytesReader will also close on overflow.

	limitedReader := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(limitedReader)
	decoder.DisallowUnknownFields()

	var result T
	if err := decoder.Decode(&result); err != nil {
		status := http.StatusBadRequest
		message := fmt.Sprintf("Invalid JSON: %v", err)

		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		var maxBytesErr *http.MaxBytesError

		switch {
		case errors.Is(err, io.EOF):
			message = "Request body must not be empty"
		case errors.As(err, &syntaxErr):
			message = fmt.Sprintf("Invalid JSON at character %d", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			message = fmt.Sprintf("Incorrect JSON type for field %q", typeErr.Field)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			message = fmt.Sprintf("Unknown field %s", strings.TrimPrefix(err.Error(), "json: unknown field "))
		case errors.As(err, &maxBytesErr):
			status = http.StatusRequestEntityTooLarge
			message = fmt.Sprintf("Request body too large (limit %d bytes)", maxBytesErr.Limit)
		}

		writeError(w, message, status)
		return nil, false
	}

	// Disallow trailing garbage after the first JSON object.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, "Request body must contain a single JSON object", http.StatusBadRequest)
		return nil, false
	}
	return &result, true
}
