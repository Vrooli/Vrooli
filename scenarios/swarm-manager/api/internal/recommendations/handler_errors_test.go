package recommendations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

func TestHandler_CreateInvalidRequests(t *testing.T) {
	dir := t.TempDir()
	handler := &Handler{
		store:         NewStore(filepath.Join(dir, "recs.json")),
		engine:        NewEngine(dir),
		settingsStore: settings.NewStore(filepath.Join(dir, "settings.json")),
	}

	invalidBodies := []string{
		"{",
		`{"scenarioName":"","type":"docs","description":"desc","priority":3}`,
		`{"scenarioName":"demo","type":"invalid","description":"desc","priority":3}`,
		`{"scenarioName":"demo","type":"docs","description":"","priority":3}`,
		`{"scenarioName":"demo","type":"docs","description":"desc","priority":0}`,
	}

	for _, body := range invalidBodies {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		handler.Create(rec, req)
		testutil.AssertStatusBadRequest(t, rec)
	}
}

func TestHandler_UpdateErrorCases(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "recs.json")
	if err := NewStore(storePath).Save([]Recommendation{}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	handler := &Handler{
		store:         NewStore(storePath),
		engine:        NewEngine(dir),
		settingsStore: settings.NewStore(filepath.Join(dir, "settings.json")),
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/recommendations/", bytes.NewBufferString(`{"status":"approved"}`))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)
	testutil.AssertStatusBadRequest(t, rec)

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/recommendations/abc", bytes.NewBufferString("{"))
	rec = httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	handler.Update(rec, req)
	testutil.AssertStatusBadRequest(t, rec)

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/recommendations/abc", bytes.NewBufferString(`{"status":"invalid"}`))
	rec = httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	handler.Update(rec, req)
	testutil.AssertStatusBadRequest(t, rec)

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/recommendations/missing", bytes.NewBufferString(`{"status":"approved"}`))
	rec = httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	handler.Update(rec, req)
	testutil.AssertStatusNotFound(t, rec)
}

func TestHandler_RefreshModeOff(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "off"
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	handler := &Handler{
		store:         NewStore(filepath.Join(dir, "recs.json")),
		engine:        NewEngine(dir),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	rec := httptest.NewRecorder()
	handler.Refresh(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[ListResponse](t, rec)
	if len(resp.Recommendations) != 0 {
		t.Fatalf("expected empty list, got %d", len(resp.Recommendations))
	}
}
