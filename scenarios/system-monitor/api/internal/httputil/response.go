package httputil

// DOC: docs/internal/ERROR-SEMANTICS.md#writeerror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/protobuf/proto"

	"system-monitor-api/internal/apierrors"
)

func JSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

func JSONWithStatus(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// ErrorDetail is the wire-format error detail.
type ErrorDetail struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	Retryable      bool   `json:"retryable"`
	RetryAfterSecs int    `json:"retry_after_seconds,omitempty"`
	Field          string `json:"field,omitempty"`
	Recovery       string `json:"recovery,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
}

// ErrorBody is the top-level error envelope.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

func categoryToStatus(cat apierrors.Category) int {
	switch cat {
	case apierrors.CategoryValidation:
		return http.StatusBadRequest
	case apierrors.CategoryUnauthorized:
		return http.StatusUnauthorized
	case apierrors.CategoryForbidden:
		return http.StatusForbidden
	case apierrors.CategoryNotFound:
		return http.StatusNotFound
	case apierrors.CategoryConflict:
		return http.StatusConflict
	case apierrors.CategoryCooldown:
		return http.StatusTooManyRequests
	case apierrors.CategoryUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func isRetryable(cat apierrors.Category) bool {
	return cat == apierrors.CategoryConflict || cat == apierrors.CategoryCooldown || cat == apierrors.CategoryUnavailable
}

// recoveryFromStatus maps an HTTP status code to a recovery hint for use in WriteError.
func recoveryFromStatus(status int) string {
	switch {
	case status == http.StatusBadRequest:
		return apierrors.RecoveryFixInput
	case status == http.StatusUnauthorized:
		return apierrors.RecoveryAuthenticate
	case status == http.StatusForbidden:
		return apierrors.RecoveryContactAdmin
	case status == http.StatusConflict,
		status == http.StatusTooManyRequests,
		status == http.StatusServiceUnavailable:
		return apierrors.RecoveryWait
	default:
		return apierrors.RecoveryNone
	}
}

// isStatusRetryable returns true for HTTP statuses that are inherently retryable.
func isStatusRetryable(status int) bool {
	return status == http.StatusConflict ||
		status == http.StatusTooManyRequests ||
		status == http.StatusServiceUnavailable
}

// WriteError writes a structured JSON error response.
func WriteError(w http.ResponseWriter, log *slog.Logger, r *http.Request, status int, code, userMessage, logDetail string) {
	reqID := r.Header.Get("X-Request-ID")
	if log != nil && logDetail != "" {
		log.Error(logDetail, "status", status, "code", code, "path", r.URL.Path, "request_id", reqID)
	}
	body := ErrorBody{
		Error: ErrorDetail{
			Code:      code,
			Message:   userMessage,
			Retryable: isStatusRetryable(status),
			Recovery:  recoveryFromStatus(status),
			RequestID: reqID,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

// WriteAPIError maps an APIError to the correct HTTP status and writes JSON.
func WriteAPIError(w http.ResponseWriter, log *slog.Logger, r *http.Request, apiErr *apierrors.APIError) {
	status := categoryToStatus(apiErr.Category)
	reqID := r.Header.Get("X-Request-ID")
	if log != nil && apiErr.InternalDetail != "" {
		log.Error(apiErr.InternalDetail, "status", status, "code", string(apiErr.Category), "path", r.URL.Path, "request_id", reqID)
	}
	body := ErrorBody{
		Error: ErrorDetail{
			Code:           string(apiErr.Category),
			Message:        apiErr.UserMessage,
			Retryable:      isRetryable(apiErr.Category),
			RetryAfterSecs: apiErr.RetryAfterSecs,
			Field:          apiErr.Field,
			Recovery:       apiErr.Recovery,
			RequestID:      reqID,
		},
	}
	// RFC 7231: set Retry-After header for 429/503 when a retry interval is specified.
	if apiErr.RetryAfterSecs > 0 && (status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable) {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", apiErr.RetryAfterSecs))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

// HandleError checks for *apierrors.APIError via errors.As and writes the appropriate response.
// Context cancellation and deadline exceeded errors are mapped to 503.
// Untyped errors become 500 with a generic message (no internal detail leak).
func HandleError(w http.ResponseWriter, log *slog.Logger, r *http.Request, err error) {
	var apiErr *apierrors.APIError
	if errors.As(err, &apiErr) {
		WriteAPIError(w, log, r, apiErr)
		return
	}
	// Context cancellation / deadline — map to 503 (retryable).
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if log != nil {
			log.Warn("request context error", "error", err.Error(), "path", r.URL.Path)
		}
		WriteError(w, nil, r, http.StatusServiceUnavailable, "unavailable", "The request was interrupted. Please try again.", "")
		return
	}
	// Generic 500 - don't leak internal details
	if log != nil {
		log.Error("unhandled error", "error", err.Error(), "path", r.URL.Path)
	}
	WriteError(w, nil, r, http.StatusInternalServerError, "internal", "An internal error occurred", "")
}

// SafeProtoJSON writes a proto message as JSON. On marshal failure it writes a 500 error.
func SafeProtoJSON(w http.ResponseWriter, log *slog.Logger, r *http.Request, msg proto.Message) {
	if err := ProtoJSON(w, msg); err != nil {
		log.Error("proto marshal failed", "error", err, "path", r.URL.Path)
		// Only attempt error response if headers haven't been sent
		WriteError(w, nil, r, http.StatusInternalServerError, "internal", "An internal error occurred", "")
	}
}

// SafeProtoJSONWithStatus writes a proto message with a status code. On failure it writes a 500 error.
func SafeProtoJSONWithStatus(w http.ResponseWriter, log *slog.Logger, r *http.Request, status int, msg proto.Message) {
	if err := ProtoJSONWithStatus(w, status, msg); err != nil {
		log.Error("proto marshal failed", "error", err, "path", r.URL.Path)
		WriteError(w, nil, r, http.StatusInternalServerError, "internal", "An internal error occurred", "")
	}
}
