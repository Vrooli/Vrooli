package recommendations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

func TestHandler_ListGeneratesWhenEmpty(t *testing.T) {
	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatalf("create scenarios dir: %v", err)
	}

	scenarioPath := filepath.Join(scenariosDir, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatalf("create scenario dir: %v", err)
	}
	serviceJSON := []byte(`{"profile":{"name":"Demo","description":"Demo","tags":["demo"]}}`)
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), serviceJSON, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioPath, "docs"), 0o755); err != nil {
		t.Fatalf("create docs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "docs", "PROBLEMS.md"), []byte("| TD-001 | Bug |"), 0o644); err != nil {
		t.Fatalf("write problems: %v", err)
	}

	settingsPath := filepath.Join(root, "settings.json")
	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"
	settingsData, _ := json.Marshal(cfg)
	if err := os.WriteFile(settingsPath, settingsData, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	storePath := filepath.Join(root, "recs.json")
	handler := &Handler{
		store:         NewStore(storePath),
		engine:        newTestEngine(scenariosDir, []scenarios.ScenarioSource{
			makeScenarioSource("demo", "Demo", scenarioPath, "demo"),
		}),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[ListResponse](t, rec)
	if len(resp.Recommendations) == 0 {
		t.Fatalf("expected recommendations, got none")
	}
	if _, err := os.Stat(storePath); err != nil {
		t.Fatalf("expected recommendations to be persisted: %v", err)
	}
}

func TestHandler_ListSettingsLoadError(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid settings: %v", err)
	}

	handler := &Handler{
		store:         NewStore(filepath.Join(dir, "recs.json")),
		engine:        newTestEngine(dir, nil),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}
