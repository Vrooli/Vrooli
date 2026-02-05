package recommendations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

func TestHandler_NewHandlerDefaults(t *testing.T) {
	handler := NewHandler("")
	if handler == nil {
		t.Fatalf("expected handler")
	}
	if handler.store == nil || handler.engine == nil || handler.settingsStore == nil {
		t.Fatalf("expected default dependencies to be set")
	}
}

func TestHandler_ListRefreshMergesRecommendations(t *testing.T) {
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
	manual := []Recommendation{{
		ID:          "manual-1",
		Scenario:    "demo",
		Type:        TypeDocs,
		Description: "Manual",
		Status:      StatusApproved,
		Priority:    2,
		Created:     "2024-01-01T00:00:00Z",
		Source:      "manual",
	}}
	if err := NewStore(storePath).Save(manual); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	handler := &Handler{
		store: NewStore(storePath),
		engine: newTestEngine(scenariosDir, []scenarios.ScenarioSource{
			makeScenarioSource("demo", "Demo", scenarioPath, "demo"),
		}),
		settingsStore: settings.NewStore(settingsPath),
	}

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?refresh=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeProtoJSON(t, rec, &apipb.ListRecommendationsResponse{})
	if len(resp.Recommendations) < 2 {
		t.Fatalf("expected merged recommendations, got %d", len(resp.Recommendations))
	}
	if !containsRecommendationID(resp.Recommendations, "manual-1") {
		t.Fatalf("expected manual recommendation to be preserved")
	}
}

func TestHandler_ListLoadError(t *testing.T) {
	root := t.TempDir()
	settingsPath := filepath.Join(root, "settings.json")
	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"
	settingsData, _ := json.Marshal(cfg)
	if err := os.WriteFile(settingsPath, settingsData, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	badPath := filepath.Join(root, "recs.json")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	handler := &Handler{
		store:         NewStore(badPath),
		engine:        newTestEngine(root, nil),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}

func containsRecommendationID(items []*domainpb.Recommendation, id string) bool {
	for _, item := range items {
		if item.GetId() == id {
			return true
		}
	}
	return false
}
