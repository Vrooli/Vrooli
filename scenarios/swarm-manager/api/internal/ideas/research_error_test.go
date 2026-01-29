package ideas

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

func (m *mockAgentErrorService) IsEnabled() bool                              { return true }
func (m *mockAgentErrorService) IsAvailable(_ context.Context) bool           { return false }
func (m *mockAgentErrorService) ResolveURL(_ context.Context) (string, error) { return "", m.err }
func (m *mockAgentErrorService) GetProfileID() string                         { return "" }

func (m *mockAgentErrorService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, m.err
}

func (m *mockAgentErrorService) SpawnRecommendation(_ context.Context, _ agentmanager.RecommendationSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, m.err
}

func TestResearch_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ideas/missing/research", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "missing"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatusNotFound(t, w)
}

func TestResearch_InvalidJSON(t *testing.T) {
	_, ideasDir := setupTestHandler(t)
	createTestIdea(t, ideasDir, Idea{
		Name:    "idea-1",
		Title:   "Idea",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	})

	h := NewHandlerWithClients(ideasDir, nil, &mockAgentErrorService{err: agentmanager.ErrRequestFailed})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ideas/idea-1/research", bytes.NewBufferString("{"))
	req = mux.SetURLVars(req, map[string]string{"name": "idea-1"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatusBadRequest(t, w)
}

func TestResearch_AgentNotAvailable(t *testing.T) {
	_, ideasDir := setupTestHandler(t)
	createTestIdea(t, ideasDir, Idea{
		Name:    "idea-2",
		Title:   "Idea",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	})

	h := NewHandlerWithClients(ideasDir, nil, &mockAgentErrorService{err: agentmanager.ErrNotAvailable})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ideas/idea-2/research", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "idea-2"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatus(t, w, http.StatusServiceUnavailable)
}

func TestResearch_AgentFailure(t *testing.T) {
	_, ideasDir := setupTestHandler(t)
	createTestIdea(t, ideasDir, Idea{
		Name:    "idea-3",
		Title:   "Idea",
		Status:  StatusBacklog,
		Created: "2026-01-28T00:00:00Z",
		Updated: "2026-01-28T00:00:00Z",
	})

	h := NewHandlerWithClients(ideasDir, nil, &mockAgentErrorService{err: errors.New("boom")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ideas/idea-3/research", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "idea-3"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatus(t, w, http.StatusInternalServerError)
}
