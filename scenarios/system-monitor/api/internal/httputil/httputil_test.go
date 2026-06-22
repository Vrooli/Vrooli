package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
)

// ---------------------------------------------------------------------------
// JSON / JSONWithStatus
// ---------------------------------------------------------------------------

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"hello": "world"}

	if err := JSON(w, data); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if got["hello"] != "world" {
		t.Errorf("body[hello] = %q, want %q", got["hello"], "world")
	}
}

func TestJSONWithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]int{"count": 42}

	if err := JSONWithStatus(w, http.StatusCreated, data); err != nil {
		t.Fatalf("JSONWithStatus returned error: %v", err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// ---------------------------------------------------------------------------
// Error helpers
// ---------------------------------------------------------------------------

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	WriteError(w, nil, r, http.StatusBadRequest, "validation", "bad input", "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Code != "validation" {
		t.Errorf("code = %q, want %q", body.Error.Code, "validation")
	}
	if body.Error.Message != "bad input" {
		t.Errorf("message = %q, want %q", body.Error.Message, "bad input")
	}
}

func TestHandleError_APIError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		status   int
		code     string
		recovery string
	}{
		{"Validation", apierrors.Validation("field", "required"), http.StatusBadRequest, "validation", "fix_input"},
		{"Unauthorized", apierrors.Unauthorized("bad creds"), http.StatusUnauthorized, "unauthorized", "authenticate"},
		{"Forbidden", apierrors.Forbidden("no access"), http.StatusForbidden, "forbidden", "contact_admin"},
		{"NotFound", apierrors.NotFound("item", "123"), http.StatusNotFound, "not_found", "none"},
		{"Cooldown", apierrors.Cooldown(30), http.StatusTooManyRequests, "cooldown", "wait"},
		{"Unavailable", apierrors.Unavailable("service"), http.StatusServiceUnavailable, "unavailable", "wait"},
		{"Internal", apierrors.Internal("oops", fmt.Errorf("db error")), http.StatusInternalServerError, "internal", "none"},
		{"Conflict", apierrors.Conflict("duplicate"), http.StatusConflict, "conflict", "wait"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			HandleError(w, nil, r, tt.err)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
			var body ErrorBody
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Error.Code != tt.code {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.code)
			}
			if body.Error.Recovery != tt.recovery {
				t.Errorf("recovery = %q, want %q", body.Error.Recovery, tt.recovery)
			}
		})
	}
}

func TestHandleError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	HandleError(w, nil, r, fmt.Errorf("some random error"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Code != "internal" {
		t.Errorf("code = %q, want %q", body.Error.Code, "internal")
	}
	// Should NOT leak internal error details
	if strings.Contains(body.Error.Message, "some random error") {
		t.Error("generic error message leaked internal details")
	}
}

func TestWriteAPIError_ValidationField(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	apiErr := apierrors.Validation("cpu_threshold", "must be 0-100")
	WriteAPIError(w, nil, r, apiErr)

	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Field != "cpu_threshold" {
		t.Errorf("field = %q, want %q", body.Error.Field, "cpu_threshold")
	}
}

func TestWriteAPIError_RequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("X-Request-ID", "abc-123-def")
	apiErr := apierrors.NotFound("item", "42")
	WriteAPIError(w, nil, r, apiErr)

	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.RequestID != "abc-123-def" {
		t.Errorf("request_id = %q, want %q", body.Error.RequestID, "abc-123-def")
	}
}

func TestHandleError_GenericWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("X-Request-ID", "req-555")
	HandleError(w, nil, r, fmt.Errorf("some error"))

	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.RequestID != "req-555" {
		t.Errorf("request_id = %q, want %q", body.Error.RequestID, "req-555")
	}
}

// ---------------------------------------------------------------------------
// ProtoJSON round-trip (snake_case verification)
// ---------------------------------------------------------------------------

func TestProtoJSON_SnakeCase(t *testing.T) {
	msg := &settingspb.SystemSettings{
		Active:       true,
		CpuThreshold: 85.5,
	}

	w := httptest.NewRecorder()
	if err := ProtoJSON(w, msg); err != nil {
		t.Fatalf("ProtoJSON returned error: %v", err)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	// Verify snake_case field names are used (proto names), not camelCase.
	if !strings.Contains(body, "cpu_threshold") {
		t.Errorf("expected snake_case key cpu_threshold in body: %s", body)
	}
	if strings.Contains(body, "cpuThreshold") {
		t.Errorf("unexpected camelCase key cpuThreshold in body: %s", body)
	}
}

// ---------------------------------------------------------------------------
// DecodeProtoJSON
// ---------------------------------------------------------------------------

func TestDecodeProtoJSON(t *testing.T) {
	jsonBody := `{"active":true,"cpu_threshold":90.0,"memory_threshold":75.5}`
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(jsonBody)))

	var msg settingspb.SystemSettings
	if err := DecodeProtoJSON(r, &msg); err != nil {
		t.Fatalf("DecodeProtoJSON returned error: %v", err)
	}

	if !msg.GetActive() {
		t.Error("expected Active = true")
	}
	if msg.GetCpuThreshold() != 90.0 {
		t.Errorf("CpuThreshold = %f, want 90.0", msg.GetCpuThreshold())
	}
	if msg.GetMemoryThreshold() != 75.5 {
		t.Errorf("MemoryThreshold = %f, want 75.5", msg.GetMemoryThreshold())
	}
}

// ---------------------------------------------------------------------------
// ProtoJSONWithStatus
// ---------------------------------------------------------------------------

func TestProtoJSONWithStatus(t *testing.T) {
	msg := &settingspb.SystemSettings{
		Active:        false,
		DiskThreshold: 95.0,
	}

	w := httptest.NewRecorder()
	if err := ProtoJSONWithStatus(w, http.StatusAccepted, msg); err != nil {
		t.Fatalf("ProtoJSONWithStatus returned error: %v", err)
	}

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "disk_threshold") {
		t.Errorf("expected disk_threshold in body: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Context cancellation / deadline handling
// ---------------------------------------------------------------------------

func TestHandleError_ContextCanceled(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	HandleError(w, nil, r, context.Canceled)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Code != "unavailable" {
		t.Errorf("code = %q, want %q", body.Error.Code, "unavailable")
	}
	if body.Error.Recovery != "wait" {
		t.Errorf("recovery = %q, want %q", body.Error.Recovery, "wait")
	}
	if !body.Error.Retryable {
		t.Error("expected retryable = true for 503")
	}
}

func TestHandleError_DeadlineExceeded(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	HandleError(w, nil, r, context.DeadlineExceeded)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Code != "unavailable" {
		t.Errorf("code = %q, want %q", body.Error.Code, "unavailable")
	}
	if !body.Error.Retryable {
		t.Error("expected retryable = true")
	}
}

func TestHandleError_WrappedContextError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	wrapped := fmt.Errorf("query: %w", context.Canceled)
	HandleError(w, nil, r, wrapped)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	var body ErrorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Code != "unavailable" {
		t.Errorf("code = %q, want %q", body.Error.Code, "unavailable")
	}
}

// ---------------------------------------------------------------------------
// WriteError retryable and recovery fields
// ---------------------------------------------------------------------------

func TestWriteError_RetryableAndRecoveryFields(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		wantRetry    bool
		wantRecovery string
	}{
		{"503", http.StatusServiceUnavailable, true, "wait"},
		{"429", http.StatusTooManyRequests, true, "wait"},
		{"409", http.StatusConflict, true, "wait"},
		{"500", http.StatusInternalServerError, false, "none"},
		{"400", http.StatusBadRequest, false, "fix_input"},
		{"401", http.StatusUnauthorized, false, "authenticate"},
		{"403", http.StatusForbidden, false, "contact_admin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			WriteError(w, nil, r, tt.status, "test", "test message", "")

			var body ErrorBody
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Error.Retryable != tt.wantRetry {
				t.Errorf("retryable = %v, want %v", body.Error.Retryable, tt.wantRetry)
			}
			if body.Error.Recovery != tt.wantRecovery {
				t.Errorf("recovery = %q, want %q", body.Error.Recovery, tt.wantRecovery)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Retry-After header
// ---------------------------------------------------------------------------

func TestWriteAPIError_RetryAfterHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	apiErr := apierrors.Cooldown(60)
	WriteAPIError(w, nil, r, apiErr)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	ra := w.Header().Get("Retry-After")
	if ra != "60" {
		t.Errorf("Retry-After = %q, want %q", ra, "60")
	}
}

func TestWriteAPIError_NoRetryAfterFor400(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	apiErr := apierrors.Validation("name", "required")
	WriteAPIError(w, nil, r, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	ra := w.Header().Get("Retry-After")
	if ra != "" {
		t.Errorf("Retry-After = %q, want empty", ra)
	}
}
