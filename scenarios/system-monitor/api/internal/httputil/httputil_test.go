package httputil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
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

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(http.ResponseWriter, string, string)
		status int
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest},
		{"NotFound", NotFound, http.StatusNotFound},
		{"InternalError", InternalError, http.StatusInternalServerError},
		{"Conflict", Conflict, http.StatusConflict},
		{"ServiceUnavailable", ServiceUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.fn(w, "test", "something went wrong")

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
			body := strings.TrimSpace(w.Body.String())
			if body != "something went wrong" {
				t.Errorf("body = %q, want %q", body, "something went wrong")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ProtoJSON round-trip (snake_case verification)
// ---------------------------------------------------------------------------

func TestProtoJSON_SnakeCase(t *testing.T) {
	msg := &domain.SystemSettings{
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

	var msg domain.SystemSettings
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
	msg := &domain.SystemSettings{
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
