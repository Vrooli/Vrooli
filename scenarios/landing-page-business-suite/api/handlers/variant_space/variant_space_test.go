package variant_space

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetServesVerbatimJSON(t *testing.T) {
	payload := []byte(`{"_metadata":{"preserve":true}}`)
	w := httptest.NewRecorder()
	Get(Dependencies{JSON: func() []byte { return payload }, Log: func(string, map[string]any) {}})(w, httptest.NewRequest(http.MethodGet, "/api/v1/variant-space", nil))
	if got := w.Body.String(); got != string(payload) {
		t.Fatalf("body = %q, want %q", got, payload)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}
