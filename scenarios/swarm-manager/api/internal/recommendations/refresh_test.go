package recommendations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

func TestHandler_RefreshGeneratesRecommendations(t *testing.T) {
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

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[ListResponse](t, rec)
	if len(resp.Recommendations) == 0 {
		t.Fatalf("expected recommendations, got none")
	}
}
