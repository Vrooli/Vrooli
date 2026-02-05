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
	"google.golang.org/protobuf/encoding/protojson"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

func TestEngine_GenerateProblemsRecommendation(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("create scenario dir: %v", err)
	}

	serviceJSON := []byte(`{"profile":{"name":"Demo","description":"Demo scenario","tags":["demo"]}}`)
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), serviceJSON, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(scenarioDir, "docs"), 0o755); err != nil {
		t.Fatalf("create docs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "docs", "PROBLEMS.md"), []byte("| TD-001 | Bug |"), 0o644); err != nil {
		t.Fatalf("write problems: %v", err)
	}

	sources := []scenarios.ScenarioSource{
		makeScenarioSource("demo", "Demo scenario", scenarioDir, "demo"),
	}
	engine := newTestEngine(dir, sources)
	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"

	recs, err := engine.Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(recs) == 0 {
		t.Fatalf("expected recommendations, got none")
	}
	if recs[0].Type != TypeRefactor {
		t.Errorf("expected refactor type, got %s", recs[0].Type)
	}
}

func TestHandler_ListModeOffReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	recPath := filepath.Join(dir, "recs.json")

	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "off"
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	handler := &Handler{
		store:         NewStore(recPath),
		engine:        newTestEngine(dir, nil),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations", nil)
	recorder := httptest.NewRecorder()
	handler.List(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	response := testutil.DecodeProtoJSON(t, recorder, &apipb.ListRecommendationsResponse{})
	if len(response.Recommendations) != 0 {
		t.Fatalf("expected empty list, got %d", len(response.Recommendations))
	}
}

func TestHandler_CreateAndUpdate(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "recs.json")

	handler := &Handler{
		store:         NewStore(recPath),
		engine:        newTestEngine(dir, nil),
		settingsStore: settings.NewStore(filepath.Join(dir, "settings.json")),
	}

	createPayload := &apipb.CreateRecommendationRequest{
		ScenarioName: "demo",
		Type:         "docs",
		Description:  "Add documentation",
		Priority:     3,
	}
	body, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.Create(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

	created := testutil.DecodeProtoJSON(t, recorder, &apipb.RecommendationResponse{})

	status := "approved"
	patch := &apipb.UpdateRecommendationRequest{Status: &status}
	patchBody, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(patch)

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/recommendations/"+created.Recommendation.Id, bytes.NewReader(patchBody))
	updateRec := httptest.NewRecorder()
	updateReq = muxSetVars(updateReq, map[string]string{"id": created.Recommendation.Id})
	handler.Update(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", updateRec.Code)
	}
}

// muxSetVars is a helper to inject mux vars in unit tests.
func muxSetVars(req *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(req, vars)
}
