package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"
)

// workshopSaveResponse mirrors the JSON shape of WorkshopSaveResponse.
type workshopSaveResponse struct {
	File        BacklogFile     `json:"file"`
	AutoAdvance workshopAutoAdv `json:"auto_advance"`
}

type workshopAutoAdv struct {
	Triggered bool    `json:"triggered"`
	RunID     *string `json:"run_id,omitempty"`
	TaskID    *string `json:"task_id,omitempty"`
	Reason    string  `json:"reason"`
}

func makeWorkshopSaveBody(roundNumber int, round workshop.Round, autoWorkshop *bool) []byte {
	content, _ := json.Marshal(round)
	body := map[string]any{
		"round_number": roundNumber,
		"content":      string(content),
	}
	if autoWorkshop != nil {
		body["auto_workshop"] = *autoWorkshop
	}
	data, _ := json.Marshal(body)
	return data
}

func workshopSaveRequest(kind, name string, body []byte) *http.Request {
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/backlog/%s/%s/workshop/save", kind, name), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": kind, "name": name})
	return req
}

func TestWorkshopSave_ValidRound_WritesFile(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-test", Title: "WS Test", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	// Auto-workshop disabled since we don't have a mock agent.
	falseVal := false
	body := makeWorkshopSaveBody(1, round, &falseVal)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-test", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was written.
	roundPath := filepath.Join(rootDir, "ideas", "ws-test", "workshop", "round-001.json")
	testutil.AssertFileExists(t, roundPath)

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.File.Name != "round-001.json" {
		t.Errorf("expected file name 'round-001.json', got %q", resp.File.Name)
	}
	if resp.AutoAdvance.Reason != "opt_out" {
		t.Errorf("expected reason 'opt_out', got %q", resp.AutoAdvance.Reason)
	}
}

func TestWorkshopSave_InvalidJSON_Returns400(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-bad", Title: "WS Bad", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	body, _ := json.Marshal(map[string]any{
		"round_number": 1,
		"content":      "not valid json {{{",
	})
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-bad", body))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkshopSave_ItemNotFound_Returns404(t *testing.T) {
	h, _ := setupTestHandler(t)

	round := workshop.Round{RoundNum: 1, Readiness: map[string]int{}}
	body := makeWorkshopSaveBody(1, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "nonexistent", body))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWorkshopSave_AutoAdvance_Triggers(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-123", TaskID: "task-456"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-advance", Title: "WS Advance", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Low scores → not ready → should auto-advance.
	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-advance", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.AutoAdvance.Triggered {
		t.Errorf("expected auto-advance to be triggered, reason=%s", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.Reason != "not_ready" {
		t.Errorf("expected reason 'not_ready', got %q", resp.AutoAdvance.Reason)
	}
	if agent.lastReq == nil {
		t.Fatal("expected agent spawn to be called")
	}
}

func TestWorkshopSave_Ready_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-ready", Title: "WS Ready", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// All scores 3 → ready.
	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-ready", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT to be triggered when ready")
	}
	if resp.AutoAdvance.Reason != "ready" {
		t.Errorf("expected reason 'ready', got %q", resp.AutoAdvance.Reason)
	}
	if agent.lastReq != nil {
		t.Error("expected no agent spawn when item is ready")
	}
}

func TestWorkshopSave_MaxRounds_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-maxrounds", Title: "WS Max", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Write 9 existing rounds, then save round 10.
	workshopDir := filepath.Join(rootDir, "ideas", "ws-maxrounds", "workshop")
	testutil.MakeDir(t, workshopDir)
	for i := 1; i <= 9; i++ {
		r := workshop.Round{
			RoundNum:  i,
			Readiness: map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
			Items:     []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
		}
		data, _ := json.Marshal(r)
		testutil.WriteFile(t, filepath.Join(workshopDir, fmt.Sprintf("round-%03d.json", i)), string(data))
	}

	round := workshop.Round{
		RoundNum:    10,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(10, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-maxrounds", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered at max rounds")
	}
	if resp.AutoAdvance.Reason != "max_rounds" {
		t.Errorf("expected reason 'max_rounds', got %q", resp.AutoAdvance.Reason)
	}
}

func TestWorkshopSave_PendingDecisions_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-pending", Title: "WS Pending", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Unanswered decision.
	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision"}}, // No Selected
	}

	body := makeWorkshopSaveBody(1, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-pending", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered with pending decisions")
	}
	if resp.AutoAdvance.Reason != "pending_decisions" {
		t.Errorf("expected reason 'pending_decisions', got %q", resp.AutoAdvance.Reason)
	}
}

func TestWorkshopSave_OptOut_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-optout", Title: "WS OptOut", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	falseVal := false
	body := makeWorkshopSaveBody(1, round, &falseVal)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-optout", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered with opt-out")
	}
	if resp.AutoAdvance.Reason != "opt_out" {
		t.Errorf("expected reason 'opt_out', got %q", resp.AutoAdvance.Reason)
	}
	if agent.lastReq != nil {
		t.Error("expected no agent spawn with opt-out")
	}
}

func TestWorkshopSave_AgentDown_StillSaves(t *testing.T) {
	agent := &mockAgentService{
		err: fmt.Errorf("agent unavailable"),
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-agentdown", Title: "WS Agent Down", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-agentdown", body))

	// Save should still succeed.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// File should be written.
	roundPath := filepath.Join(rootDir, "ideas", "ws-agentdown", "workshop", "round-001.json")
	testutil.AssertFileExists(t, roundPath)

	var resp workshopSaveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered when agent is down")
	}
	if resp.AutoAdvance.Reason != "error" {
		t.Errorf("expected reason 'error', got %q", resp.AutoAdvance.Reason)
	}
}

func TestWorkshopSave_ConcurrentSaves_LockPreventsDouble(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-lock", Title: "WS Lock", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Manually create a lock file to simulate a concurrent spawn.
	itemDir := filepath.Join(rootDir, "ideas", "ws-lock")
	lockPath := filepath.Join(itemDir, ".workshop-lock")
	testutil.WriteFile(t, lockPath, "2026-01-01T00:00:00Z")

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round, nil)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-lock", body))

	// Save succeeds.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	// Auto-advance should fail due to lock → reported as error.
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered when lock is held")
	}
	if resp.AutoAdvance.Reason != "error" {
		t.Errorf("expected reason 'error', got %q", resp.AutoAdvance.Reason)
	}
	// Agent should NOT have been called.
	if agent.lastReq != nil {
		t.Error("expected no agent spawn when lock is held")
	}

	// Clean up lock.
	os.Remove(lockPath)
}

func strPtr(s string) *string { return &s }
