package backlog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"
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
		{KindChore, "chore"},
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
	got, err := parseResearchMode("initialize")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ResearchModeInitialize {
		t.Errorf("expected ResearchModeInitialize, got %q", got)
	}
}

func TestParseResearchMode_Finalize(t *testing.T) {
	got, err := parseResearchMode("finalize")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ResearchModeFinalize {
		t.Errorf("expected ResearchModeFinalize, got %q", got)
	}
}

func TestResearchSkillID_Initialize(t *testing.T) {
	kinds := []BacklogKind{KindIdea, KindFix, KindExecute, KindResearch, KindChore}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			entry, ok := promptcatalog.ResolveBacklogSkill(string(ResearchModeInitialize), string(kind))
			if !ok {
				t.Fatalf("expected prompt catalog entry for initialize/%s", kind)
			}
			got := entry.SkillID
			want := "swarm-manager-initialize-backlog"
			if got != want {
				t.Errorf("ResolveBacklogSkill(initialize, %s) = %q, want %q", kind, got, want)
			}
		})
	}
}

func TestResearchSkillID_Finalize(t *testing.T) {
	kinds := []struct {
		kind BacklogKind
		want string
	}{
		{KindIdea, "swarm-manager-workshop-finalize"},
		{KindFix, "swarm-manager-workshop-finalize"},
		{KindExecute, "swarm-manager-workshop-finalize"},
		{KindResearch, "swarm-manager-workshop-research-finalize"},
		{KindChore, "swarm-manager-workshop-finalize"},
	}
	for _, tt := range kinds {
		t.Run(string(tt.kind), func(t *testing.T) {
			entry, ok := promptcatalog.ResolveBacklogSkill(string(ResearchModeFinalize), string(tt.kind))
			if !ok {
				t.Fatalf("expected prompt catalog entry for finalize/%s", tt.kind)
			}
			got := entry.SkillID
			if got != tt.want {
				t.Errorf("ResolveBacklogSkill(finalize, %s) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestResearch_FinalizeMode_RejectsWithoutPendingSynthesis(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "finalize-no-pending",
		Title:    "Finalize No Pending",
		Status:   StatusResearching,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", item.Name, "workshop", "round-001.json"), workshop.Round{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:     []workshop.Item{{ID: "i1", Type: "info", Text: "already current"}},
	})

	body := `{"mode":"finalize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/finalize-no-pending/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": item.Name})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResearch_FinalizeMode_RejectsWhenNotReady(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "finalize-not-ready",
		Title:    "Finalize Not Ready",
		Status:   StatusResearching,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", item.Name, "workshop", "round-001.json"), workshop.Round{
		RoundNum:         1,
		PendingSynthesis: true,
		Readiness:        map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:            []workshop.Item{{ID: "d1", Type: "decision", Selected: strPtr("A")}},
	})

	body := `{"mode":"finalize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/finalize-not-ready/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": item.Name})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResearch_FinalizeMode_SpawnsAgent(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task-finalize", RunID: "run-finalize", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "finalize-ok",
		Title:    "Finalize Ok",
		Status:   StatusResearching,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindIdea, item)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", item.Name, "workshop", "round-001.json"), workshop.Round{
		RoundNum:         1,
		PendingSynthesis: true,
		Readiness:        map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:            []workshop.Item{{ID: "d1", Type: "decision", Selected: strPtr("A")}},
	})

	body := `{"mode":"finalize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/finalize-ok/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": item.Name})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if agent.lastReq == nil {
		t.Fatal("expected SpawnBacklog to be called")
	}
	if agent.lastReq.Title != "Finalize: Finalize Ok" {
		t.Errorf("expected finalize title, got %q", agent.lastReq.Title)
	}
}

func TestResearch_FinalizeMode_AllowsLegacyAnsweredRound(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task-legacy", RunID: "run-legacy", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	item := BacklogItem{
		Name:     "legacy-finalize-ok",
		Title:    "Legacy Finalize Ok",
		Status:   StatusResearching,
		Priority: 3,
		Tags:     []string{},
		Created:  "2026-03-09T00:00:00Z",
		Updated:  "2026-03-09T00:00:00Z",
	}
	createTestItem(t, rootDir, KindResearch, item)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "research", item.Name, "workshop", "round-001.json"), workshop.Round{
		RoundNum:  1,
		Readiness: map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items: []workshop.Item{
			{ID: "d1", Type: "decision", Selected: strPtr("A")},
			{ID: "i1", Type: "info", Text: "legacy"},
		},
	})

	body := `{"mode":"finalize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/research/legacy-finalize-ok/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "research", "name": item.Name})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if agent.lastReq == nil {
		t.Fatal("expected SpawnBacklog to be called")
	}
	if agent.lastReq.Title != "Finalize: Legacy Finalize Ok" {
		t.Errorf("expected finalize title, got %q", agent.lastReq.Title)
	}
}

// ---------------------------------------------------------------------------
// Research dependency blocking tests
// ---------------------------------------------------------------------------

type researchBlockingResponse struct {
	DryRun          bool   `json:"dry_run"`
	Started         bool   `json:"started"`
	Message         string `json:"message"`
	BlockingReasons []struct {
		Message   string `json:"message"`
		Forceable bool   `json:"forceable"`
	} `json:"blocking_reasons"`
}

func TestResearch_BlocksOnUnmetDeps_DryRun(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task", RunID: "run", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Create dep in "backlog" status.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "research-dep", Title: "Dep", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-03-09T00:00:00Z", Updated: "2026-03-09T00:00:00Z",
	})
	// Create item that depends on it.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "research-child", Title: "Child", Status: StatusBacklog, Priority: 3,
		Tags: []string{}, Created: "2026-03-09T00:00:00Z", Updated: "2026-03-09T00:00:00Z",
		DependsOn: []string{"idea/research-dep"},
	})

	// No confirm → dry_run with blocking reasons.
	body := `{"mode":"initialize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/research-child/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "research-child"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	testutil.AssertStatusOK(t, w)
	resp := testutil.DecodeJSON[researchBlockingResponse](t, w)
	if !resp.DryRun {
		t.Error("expected dry_run=true")
	}
	if resp.Started {
		t.Error("expected started=false")
	}
	if len(resp.BlockingReasons) == 0 {
		t.Fatal("expected blocking reasons")
	}
	if agent.lastReq != nil {
		t.Error("expected no agent spawn when blocked")
	}
}

func TestResearch_BlocksOnUnmetDeps_ForceOverrides(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task-forced", RunID: "run-forced", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Create dep in "backlog" status.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "force-dep", Title: "Dep", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-03-09T00:00:00Z", Updated: "2026-03-09T00:00:00Z",
	})
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "force-child", Title: "Child", Status: StatusBacklog, Priority: 3,
		Tags: []string{}, Created: "2026-03-09T00:00:00Z", Updated: "2026-03-09T00:00:00Z",
		DependsOn: []string{"idea/force-dep"},
	})

	// confirm=true, force=true → should proceed and spawn.
	body := `{"mode":"initialize","confirm":true,"force":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/force-child/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "force-child"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if agent.lastReq == nil {
		t.Fatal("expected SpawnBacklog to be called with force override")
	}
}

func TestResearch_FinalizeMode_SkipsDepsCheck(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{TaskID: "task-fin", RunID: "run-fin", BaseURL: "http://agent", CreatedAt: "now"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Create dep in "backlog" status — would block if checked.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "fin-dep", Title: "Dep", Status: StatusBacklog, Priority: 5,
		Tags: []string{}, Created: "2026-03-09T00:00:00Z", Updated: "2026-03-09T00:00:00Z",
	})
	// Create item in "researching" with unmet dep and a ready workshop round.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "fin-child", Title: "Finalize Child", Status: StatusResearching, Priority: 3,
		Tags: []string{}, Created: "2026-03-09T00:00:00Z", Updated: "2026-03-09T00:00:00Z",
		DependsOn: []string{"idea/fin-dep"},
	})
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "ideas", "fin-child", "workshop", "round-001.json"), workshop.Round{
		RoundNum:         1,
		PendingSynthesis: true,
		Readiness:        map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:            []workshop.Item{{ID: "d1", Type: "decision", Selected: strPtr("A")}},
	})

	body := `{"mode":"finalize"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/idea/fin-child/research", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "fin-child"})
	w := httptest.NewRecorder()

	h.Research(w, req)

	// Finalize should proceed despite unmet dep — it skips dep checks.
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if agent.lastReq == nil {
		t.Fatal("expected SpawnBacklog to be called for finalize despite unmet dep")
	}
}
