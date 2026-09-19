package apierr

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// MapError writes an appropriate HTTP error response based on the error type.
//
// If err is a *DomainError with a Code or Details set, the response body is
// JSON ({"error": code, "message": ..., "details": ...}); otherwise it falls
// back to plain text (http.Error). Untyped errors render as a generic 500.
//
// logPrefix is included in log output for tracing (e.g., "[execution] create").
func MapError(w http.ResponseWriter, logPrefix string, err error) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		if logPrefix != "" {
			slog.Error("domain error", "prefix", logPrefix, "message", domainErr.Message, "status", domainErr.Status, "code", domainErr.Code)
		}
		if domainErr.Code != "" || domainErr.Details != nil {
			writeJSONError(w, domainErr)
			return
		}
		http.Error(w, domainErr.Message, domainErr.Status)
		return
	}

	// Fallback: untyped error → 500.
	msg := truncateMessage(err, 240)
	if logPrefix != "" {
		slog.Error("unexpected error", "prefix", logPrefix, "error", err)
	}
	http.Error(w, msg, http.StatusInternalServerError)
}

// writeJSONError emits a structured JSON error envelope:
//
//	{"error": "<code>", "message": "...", "details": <details>}
//
// Code defaults to "error" when unset but Details is present.
func writeJSONError(w http.ResponseWriter, e *DomainError) {
	code := e.Code
	if code == "" {
		code = "error"
	}
	body := map[string]any{
		"error":   code,
		"message": e.Message,
	}
	if e.Details != nil {
		body["details"] = e.Details
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(body)
}

// truncateMessage normalizes and truncates an error message for HTTP responses.
func truncateMessage(err error, maxLen int) string {
	if err == nil {
		return "unknown error"
	}
	msg := strings.Join(strings.Fields(strings.TrimSpace(err.Error())), " ")
	if msg == "" {
		return "unknown error"
	}
	if maxLen > 0 && len(msg) > maxLen {
		return msg[:maxLen] + "..."
	}
	return msg
}
