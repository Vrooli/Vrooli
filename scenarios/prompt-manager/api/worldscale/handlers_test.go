package worldscale

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGetReturnsDefaultConfigWhenMissing(t *testing.T) {
	rec := httptest.NewRecorder()

	HandleGet(t.TempDir())(rec, httptest.NewRequest(http.MethodGet, "/world-scale", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var cfg Config
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg != defaultConfig {
		t.Fatalf("expected default config, got %+v", cfg)
	}
}

func TestHandlePutRejectsOutOfRangeScale(t *testing.T) {
	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"agent":0.01,"furniture":1,"decoration":1,"overlay":1}`)

	HandlePut(t.TempDir())(rec, httptest.NewRequest(http.MethodPut, "/world-scale", body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
