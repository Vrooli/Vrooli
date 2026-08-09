package worldscale

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/internal/testutil/assertx"
	"prompt-manager/internal/testutil/httpx"
)

func TestHandleGetReturnsDefaultConfigWhenMissing(t *testing.T) {
	rec := httpx.Recorder()

	HandleGet(t.TempDir())(rec, httpx.Request(t, http.MethodGet, "/world-scale", nil, nil))

	httpx.AssertStatus(t, rec, http.StatusOK)
	cfg := httpx.DecodeJSON[Config](t, rec)
	if cfg != defaultConfig {
		t.Fatalf("expected default config, got %+v", cfg)
	}
}

func TestHandleGetRejectsMalformedPersistedConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte("{bad"), 0o644); err != nil {
		t.Fatalf("seed malformed config: %v", err)
	}
	rec := httpx.Recorder()

	HandleGet(dir)(rec, httpx.Request(t, http.MethodGet, "/world-scale", nil, nil))

	httpx.AssertStatus(t, rec, http.StatusInternalServerError)
	assertx.Contains(t, rec.Body.String(), "reading world-scale config", "malformed persisted world-scale config error")
}

func TestHandlePutRejectsOutOfRangeScale(t *testing.T) {
	rec := httpx.Recorder()
	body := strings.NewReader(`{"agent":0.01,"furniture":1,"decoration":1,"overlay":1}`)

	HandlePut(t.TempDir())(rec, httpx.Request(t, http.MethodPut, "/world-scale", body, nil))

	httpx.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlePutRejectsInvalidJSON(t *testing.T) {
	rec := httpx.Recorder()

	HandlePut(t.TempDir())(rec, httpx.Request(t, http.MethodPut, "/world-scale", strings.NewReader("{bad"), nil))

	httpx.AssertStatus(t, rec, http.StatusBadRequest)
	assertx.Contains(t, rec.Body.String(), "invalid request body", "invalid world-scale request body error")
}

func TestHandlePutPersistsConfig(t *testing.T) {
	dir := t.TempDir()
	rec := httpx.Recorder()
	want := Config{Agent: 1.5, Furniture: 0.75, Decoration: 2.25, Overlay: 3}

	HandlePut(dir)(rec, httpx.JSONRequest(t, http.MethodPut, "/world-scale", want, nil))

	httpx.AssertStatus(t, rec, http.StatusOK)
	gotResponse := httpx.DecodeJSON[Config](t, rec)
	if gotResponse != want {
		t.Fatalf("expected response %+v, got %+v", want, gotResponse)
	}
	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	var gotFile Config
	if err := json.Unmarshal(data, &gotFile); err != nil {
		t.Fatalf("decode persisted config: %v", err)
	}
	if gotFile != want {
		t.Fatalf("expected persisted config %+v, got %+v", want, gotFile)
	}
}
