package backlog

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
)

func TestResearch_InitializeMode_SpawnsAgent(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task-init", RunID: "run-init", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "init-test",
		Title:    "Test Initialize",
		Status:   StatusBacklog,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	body := `{"mode":"initialize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/init-test/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "init-test"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if agent.lastReq == nil {
		t.Fatal("expected SpawnBacklog to be called")
	}
	if agent.lastReq.Title != "Initialize: Test Initialize" {
		t.Errorf("expected title 'Initialize: Test Initialize', got %q", agent.lastReq.Title)
	}
	if agent.lastReq.Purpose != "research" {
		t.Errorf("expected purpose 'research', got %q", agent.lastReq.Purpose)
	}
}

func TestResearch_InitializeMode_RejectsNonBacklogStatus(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "init-ready",
		Title:    "Ready Item",
		Status:   StatusReady,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	body := `{"mode":"initialize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/init-ready/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "init-ready"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResearch_InitializeMode_AllKinds(t *testing.T) {
	kinds := []struct {
		kind BacklogKind
		dir  string
	}{
		{KindIdea, "idea"},
		{KindFix, "fix"},
		{KindExecute, "execute"},
		{KindResearch, "research"},
	}

	for _, tc := range kinds {
		t.Run(string(tc.kind), func(t *testing.T) {
			agent := &mockAgentService{
				result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
			}
			h, rootDir := setupTestHandlerWithAgent(t, agent)

			item := BacklogItem{
				Name:     "init-" + string(tc.kind),
				Title:    "Test " + string(tc.kind),
				Status:   StatusBacklog,
				Priority: 3,
				Tags:     []string{},
				Created:  "2026-03-09T00:00:00Z",
				Updated:  "2026-03-09T00:00:00Z",
			}
			createTestItem(t, rootDir, tc.kind, item)

			body := `{"mode":"initialize"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/"+tc.dir+"/init-"+string(tc.kind)+"/research", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req = mux.SetURLVars(req, map[string]string{"kind": tc.dir, "name": "init-" + string(tc.kind)})
			w := httptest.NewRecorder()

			h.Research(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
			}
			if agent.lastReq == nil {
				t.Fatal("expected SpawnBacklog to be called")
			}
		})
	}
}

func TestResearch_InitializeMode_DryRun(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "init-dry",
		Title:    "Dry Run Test",
		Status:   StatusBacklog,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	body := `{"mode":"initialize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/init-dry/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "init-dry"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatusOK(t, w)
	if agent.lastReq != nil {
		t.Fatal("expected SpawnBacklog NOT to be called for dry run")
	}
}

func TestResearch_InitializeMode_AgentUnavailable(t *testing.T) {
	h, rootDir := setupTestHandlerWithAgent(t, &mockAgentErrorService{err: agentmanager.ErrNotAvailable})

	item := BacklogItem{
		Name:     "init-unavail",
		Title:    "Unavailable Test",
		Status:   StatusBacklog,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)

	body := `{"mode":"initialize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/init-unavail/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "init-unavail"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestParseResearchMode_Initialize(t *testing.T) {
	got := parseResearchMode("initialize")
	if got != ResearchModeInitialize {
		t.Errorf("expected ResearchModeInitialize, got %q", got)
	}
}

func TestResearchSkillID_Initialize(t *testing.T) {
	kinds := []BacklogKind{KindIdea, KindFix, KindExecute, KindResearch}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			got := researchSkillID(ResearchModeInitialize, kind)
			want := "swarm-manager-initialize-backlog"
			if got != want {
				t.Errorf("researchSkillID(initialize, %s) = %q, want %q", kind, got, want)
			}
		})
	}
}

// mockInitRecordingAgent records calls and verifies skill ID.
type mockInitRecordingAgent struct {
	lastReq *agentmanager.BacklogSpawnRequest
	result  agentmanager.RunResult
}

func (m *mockInitRecordingAgent) IsEnabled() bool                    { return true }
func (m *mockInitRecordingAgent) IsAvailable(_ context.Context) bool { return true }
func (m *mockInitRecordingAgent) ResolveURL(_ context.Context) (string, error) {
	return "http://agent", nil
}
func (m *mockInitRecordingAgent) GetProfileID() string { return "" }
func (m *mockInitRecordingAgent) SpawnBacklog(_ context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	m.lastReq = &req
	return m.result, nil
}

func (m *mockInitRecordingAgent) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

func (m *mockInitRecordingAgent) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}
func (m *mockInitRecordingAgent) StopRun(_ context.Context, _ string) error { return nil }
