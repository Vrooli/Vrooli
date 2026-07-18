package backlog

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
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
func (m *mockAgentErrorService) ApproveRun(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockAgentErrorService) ContinueRun(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockAgentErrorService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}

func (m *mockAgentErrorService) GetRunDiff(_ context.Context, runID string) (agentmanager.RunDiff, error) {
	return agentmanager.RunDiff{RunID: runID}, nil
}
func (m *mockAgentErrorService) StopRun(_ context.Context, _ string) error { return nil }

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
	h.SetWorkshopWorkflow(&fakeWorkshopWorkflow{err: agentmanager.ErrNotAvailable})

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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-2/research", bytes.NewBufferString(`{"mode":"workshop"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-2"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestResearch_AgentError(t *testing.T) {
	// A runner is wired, but the live phase-start seam fails. A non-typed start
	// error maps to invokeInternal -> 500, matching the legacy spawn-error contract.
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-err")
	phase.err = errors.New("boom")

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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/idea-3/research", bytes.NewBufferString(`{"mode":"workshop"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "idea-3"})
	w := httptest.NewRecorder()

	h.Research(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
