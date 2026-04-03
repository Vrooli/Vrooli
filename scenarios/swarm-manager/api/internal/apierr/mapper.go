package apierr

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

// MapError writes an appropriate HTTP error response based on the error type.
//
// If err is a *DomainError, the embedded status code and message are used.
// Otherwise a generic 500 is returned with a truncated error message.
//
// logPrefix is included in log output for tracing (e.g., "[execution] create").
func MapError(w http.ResponseWriter, logPrefix string, err error) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		if logPrefix != "" {
			log.Printf("%s: %s (status=%d)", logPrefix, domainErr.Message, domainErr.Status)
		}
		http.Error(w, domainErr.Message, domainErr.Status)
		return
	}

	// Fallback: untyped error → 500.
	msg := truncateMessage(err, 240)
	if logPrefix != "" {
		log.Printf("%s: unexpected error: %v", logPrefix, err)
	}
	http.Error(w, msg, http.StatusInternalServerError)
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
