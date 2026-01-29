package backlog

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
)

type mockAgentErrorService struct {
	err error
}

func (m *mockAgentErrorService) IsEnabled() bool                    { return true }
func (m *mockAgentErrorService) IsAvailable(_ context.Context) bool { return true }
func (m *mockAgentErrorService) ResolveURL(_ context.Context) (string, error) {
	return "http://agent", nil
}
func (m *mockAgentErrorService) GetProfileID() string { return "" }
func (m *mockAgentErrorService) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, m.err
}
func (m *mockAgentErrorService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}
func (m *mockAgentErrorService) SpawnRecommendation(_ context.Context, _ agentmanager.RecommendationSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

func TestResearch_InvalidJSON(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentErrorService{err: nil})

	item := BacklogItem{
		Name:        "idea-1",
		Title:       "Idea",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-1/research", bytes.NewBufferString("{"))
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-1"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	testutil.AssertStatusBadRequest(t, w)
}

func TestResearch_AgentUnavailable(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentErrorService{err: agentmanager.ErrNotAvailable})

	item := BacklogItem{
		Name:        "idea-2",
		Title:       "Idea",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-2/research", bytes.NewBufferString(`{"mode":"clarify"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-2"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestResearch_AgentError(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentErrorService{err: errors.New("boom")})

	item := BacklogItem{
		Name:        "idea-3",
		Title:       "Idea",
		Description: "",
		Status:      StatusBacklog,
		Priority:    3,
		Tags:        []string{},
		Created:     "2026-01-28T00:00:00Z",
		Updated:     "2026-01-28T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-3/research", bytes.NewBufferString(`{"mode":"clarify"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-3"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
