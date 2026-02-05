package recommendations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

type mockAgentService struct {
	result agentmanager.RunResult
	err    error
}

func (m *mockAgentService) IsEnabled() bool                              { return true }
func (m *mockAgentService) IsAvailable(_ context.Context) bool           { return m.err == nil }
func (m *mockAgentService) ResolveURL(_ context.Context) (string, error) { return "", nil }
func (m *mockAgentService) GetProfileID() string                         { return "" }

func (m *mockAgentService) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, m.err
}

func (m *mockAgentService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, m.err
}

func (m *mockAgentService) SpawnRecommendation(_ context.Context, _ agentmanager.RecommendationSpawnRequest) (agentmanager.RunResult, error) {
	if m.err != nil {
		return agentmanager.RunResult{}, m.err
	}
	return m.result, nil
}

func TestHandler_StartRecommendationSuccess(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "recs.json")
	settingsPath := filepath.Join(dir, "settings.json")

	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"
	testutil.WriteJSONFile(t, settingsPath, cfg)

	rec := Recommendation{
		ID:          "rec-1",
		Scenario:    "demo",
		Type:        TypeFeature,
		Description: "Add feature",
		Status:      StatusPending,
		Priority:    3,
		Created:     "2026-01-28T00:00:00Z",
		Source:      "generated",
	}
	if err := NewStore(storePath).Save([]Recommendation{rec}); err != nil {
		t.Fatalf("seed recommendations: %v", err)
	}

	handler := NewHandlerWithServices(
		NewStore(storePath),
		newTestEngine(dir, nil),
		settings.NewStore(settingsPath),
		&mockAgentService{result: agentmanager.RunResult{TaskID: "task-1", RunID: "run-1", CreatedAt: "2026-01-28T12:00:00Z"}},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec-1/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rec-1"})
	recorder := httptest.NewRecorder()

	handler.Start(recorder, req)

	testutil.AssertStatusCreated(t, recorder)

	response := testutil.DecodeProtoJSON(t, recorder, &apipb.RecommendationResponse{})
	if response.Recommendation.GetTaskId() != "task-1" || response.Recommendation.GetRunId() != "run-1" {
		t.Fatalf("expected run identifiers to be set")
	}
	if response.Recommendation.GetStartedAt() == "" {
		t.Fatalf("expected startedAt to be set")
	}

	updated := testutil.ReadJSONFile[[]Recommendation](t, storePath)
	if len(updated) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(updated))
	}
	if updated[0].TaskID == "" || updated[0].RunID == "" {
		t.Fatalf("expected stored run identifiers")
	}
}

func TestHandler_StartRecommendationAgentUnavailable(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "recs.json")
	settingsPath := filepath.Join(dir, "settings.json")

	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"
	testutil.WriteJSONFile(t, settingsPath, cfg)

	rec := Recommendation{
		ID:          "rec-2",
		Scenario:    "demo",
		Type:        TypeFeature,
		Description: "Add feature",
		Status:      StatusPending,
		Priority:    3,
		Created:     "2026-01-28T00:00:00Z",
		Source:      "generated",
	}
	if err := NewStore(storePath).Save([]Recommendation{rec}); err != nil {
		t.Fatalf("seed recommendations: %v", err)
	}

	handler := NewHandlerWithServices(
		NewStore(storePath),
		newTestEngine(dir, nil),
		settings.NewStore(settingsPath),
		&mockAgentService{err: agentmanager.ErrNotAvailable},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec-2/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rec-2"})
	recorder := httptest.NewRecorder()

	handler.Start(recorder, req)

	testutil.AssertStatus(t, recorder, http.StatusServiceUnavailable)
}

func TestHandler_StartRecommendationAlreadyStarted(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "recs.json")
	settingsPath := filepath.Join(dir, "settings.json")

	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"
	testutil.WriteJSONFile(t, settingsPath, cfg)

	rec := Recommendation{
		ID:          "rec-3",
		Scenario:    "demo",
		Type:        TypeFeature,
		Description: "Add feature",
		Status:      StatusPending,
		Priority:    3,
		Created:     "2026-01-28T00:00:00Z",
		Source:      "generated",
		TaskID:      "task-existing",
	}
	if err := NewStore(storePath).Save([]Recommendation{rec}); err != nil {
		t.Fatalf("seed recommendations: %v", err)
	}

	handler := NewHandlerWithServices(
		NewStore(storePath),
		newTestEngine(dir, nil),
		settings.NewStore(settingsPath),
		&mockAgentService{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec-3/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rec-3"})
	recorder := httptest.NewRecorder()

	handler.Start(recorder, req)

	testutil.AssertStatus(t, recorder, http.StatusConflict)
}

func TestHandler_StartRecommendationModeOff(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "recs.json")
	settingsPath := filepath.Join(dir, "settings.json")

	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "off"
	testutil.WriteJSONFile(t, settingsPath, cfg)

	rec := Recommendation{
		ID:          "rec-4",
		Scenario:    "demo",
		Type:        TypeFeature,
		Description: "Add feature",
		Status:      StatusPending,
		Priority:    3,
		Created:     "2026-01-28T00:00:00Z",
		Source:      "generated",
	}
	if err := NewStore(storePath).Save([]Recommendation{rec}); err != nil {
		t.Fatalf("seed recommendations: %v", err)
	}

	handler := NewHandlerWithServices(
		NewStore(storePath),
		newTestEngine(dir, nil),
		settings.NewStore(settingsPath),
		&mockAgentService{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec-4/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rec-4"})
	recorder := httptest.NewRecorder()

	handler.Start(recorder, req)

	testutil.AssertStatus(t, recorder, http.StatusConflict)
}

func TestHandler_StartRecommendationYoloAutoApprove(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "recs.json")
	settingsPath := filepath.Join(dir, "settings.json")

	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "yolo"
	testutil.WriteJSONFile(t, settingsPath, cfg)

	rec := Recommendation{
		ID:          "rec-5",
		Scenario:    "demo",
		Type:        TypeFeature,
		Description: "Add feature",
		Status:      StatusPending,
		Priority:    3,
		Created:     "2026-01-28T00:00:00Z",
		Source:      "generated",
	}
	if err := NewStore(storePath).Save([]Recommendation{rec}); err != nil {
		t.Fatalf("seed recommendations: %v", err)
	}

	handler := NewHandlerWithServices(
		NewStore(storePath),
		newTestEngine(dir, nil),
		settings.NewStore(settingsPath),
		&mockAgentService{result: agentmanager.RunResult{TaskID: "task-5", RunID: "run-5"}},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/rec-5/start", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "rec-5"})
	recorder := httptest.NewRecorder()

	handler.Start(recorder, req)

	testutil.AssertStatusCreated(t, recorder)

	response := testutil.DecodeProtoJSON(t, recorder, &apipb.RecommendationResponse{})
	if response.Recommendation.GetStatus() != string(StatusApproved) {
		t.Fatalf("expected status approved, got %s", response.Recommendation.GetStatus())
	}
	if !response.Recommendation.GetAutoApproved() {
		t.Fatalf("expected autoApproved true")
	}
	if response.Recommendation.GetStartedBy() != "yolo" {
		t.Fatalf("expected startedBy yolo, got %q", response.Recommendation.GetStartedBy())
	}
}
