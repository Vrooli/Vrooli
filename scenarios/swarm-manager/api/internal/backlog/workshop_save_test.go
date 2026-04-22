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

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"
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
	NextMode  *string `json:"next_mode,omitempty"`
}

func makeWorkshopSaveBody(roundNumber int, round workshop.Round) []byte {
	content, _ := json.Marshal(round)
	body := map[string]any{
		"round_number": roundNumber,
		"content":      string(content),
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

// enableAutoAdvanceSettings writes settings that enable auto-advance but disable
// auto-initialize (so no agent spawn on item creation, only on workshop save).
func enableAutoAdvanceSettings(t *testing.T, rootDir string) {
	t.Helper()
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                      "dark",
		"default_mode":               "manual",
		"max_auto_rounds":            10,
		"auto_initialize_workshop":   false,
		"auto_advance_workshop":      true,
		"auto_cascade_workshop":      false,
		"auto_advance_delay_seconds": 0,
		"agent_max_turns":            60,
		"agent_timeout_seconds":      900,
		"agent_requires_approval":    true,
		"search_debounce_ms":         300,
		"toast_duration_ms":          5000,
		"delete_confirmation":        map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})
}

func TestWorkshopSave_ValidRound_WritesFile(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-finalize", TaskID: "task-finalize"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)
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

	body := makeWorkshopSaveBody(1, round)
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
	if !resp.AutoAdvance.Triggered {
		t.Fatal("expected finalize auto-advance to trigger")
	}
	if resp.AutoAdvance.Reason != "finalizing" {
		t.Errorf("expected reason 'finalizing', got %q", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "finalize" {
		t.Errorf("expected next mode 'finalize', got %v", resp.AutoAdvance.NextMode)
	}

	data, err := os.ReadFile(roundPath)
	if err != nil {
		t.Fatalf("failed to read saved round: %v", err)
	}
	var saved workshop.Round
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to unmarshal saved round: %v", err)
	}
	if !saved.PendingSynthesis {
		t.Error("expected saved round to be marked pending_synthesis=true")
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
	body := makeWorkshopSaveBody(1, round)
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

	// Enable auto-advance for this test.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                      "dark",
		"default_mode":               "manual",
		"max_auto_rounds":            10,
		"auto_initialize_workshop":   false,
		"auto_advance_workshop":      true,
		"auto_cascade_workshop":      false,
		"auto_advance_delay_seconds": 0,
		"agent_max_turns":            60,
		"agent_timeout_seconds":      900,
		"agent_requires_approval":    true,
		"search_debounce_ms":         300,
		"toast_duration_ms":          5000,
		"delete_confirmation":        map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

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

	body := makeWorkshopSaveBody(1, round)
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
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "workshop" {
		t.Errorf("expected next mode 'workshop', got %v", resp.AutoAdvance.NextMode)
	}
	if agent.lastReq == nil {
		t.Fatal("expected agent spawn to be called")
	}
}

func TestWorkshopSave_Ready_AutoFinalizes(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)
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

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-ready", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp.AutoAdvance.Triggered {
		t.Error("expected auto-finalize to be triggered when ready")
	}
	if resp.AutoAdvance.Reason != "finalizing" {
		t.Errorf("expected reason 'finalizing', got %q", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "finalize" {
		t.Errorf("expected next mode 'finalize', got %v", resp.AutoAdvance.NextMode)
	}
	if agent.lastReq == nil {
		t.Fatal("expected finalize agent spawn when item is ready")
	}
	if agent.lastReq.Title != "Finalize: WS Ready" {
		t.Errorf("expected finalize title, got %q", agent.lastReq.Title)
	}
}

func TestWorkshopSave_MaxRounds_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)
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

	body := makeWorkshopSaveBody(10, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-maxrounds", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered at max rounds")
	}
	if resp.AutoAdvance.Reason != "max_rounds" {
		t.Errorf("expected reason 'max_rounds', got %q", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "workshop" {
		t.Errorf("expected next mode 'workshop', got %v", resp.AutoAdvance.NextMode)
	}
}

func TestWorkshopSave_PendingDecisions_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)
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

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-pending", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered with pending decisions")
	}
	if resp.AutoAdvance.Reason != "pending_decisions" {
		t.Errorf("expected reason 'pending_decisions', got %q", resp.AutoAdvance.Reason)
	}
}

func TestWorkshopSave_AutoAdvanceDisabledViaSetting(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Disable auto-advance via settings.
	t.Setenv("SCENARIO_ROOT", rootDir)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                    "dark",
		"default_mode":             "manual",
		"max_auto_rounds":          10,
		"auto_initialize_workshop": true,
		"auto_advance_workshop":    false,
		"auto_cascade_workshop":    true,
		"agent_max_turns":          60,
		"agent_timeout_seconds":    900,
		"agent_requires_approval":  true,
		"search_debounce_ms":       300,
		"toast_duration_ms":        5000,
		"delete_confirmation":      map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-disabled", Title: "WS Disabled", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-disabled", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered when disabled via setting")
	}
	if resp.AutoAdvance.Reason != "disabled" {
		t.Errorf("expected reason 'disabled', got %q", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "workshop" {
		t.Errorf("expected next mode 'workshop', got %v", resp.AutoAdvance.NextMode)
	}
	if agent.lastReq != nil {
		t.Error("expected no agent spawn when auto-advance disabled")
	}
}

func TestWorkshopSave_AutoAdvanceDisabled_ReadyRequiresManualFinalize(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	t.Setenv("SCENARIO_ROOT", rootDir)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                    "dark",
		"default_mode":             "manual",
		"max_auto_rounds":          10,
		"auto_initialize_workshop": false,
		"auto_advance_workshop":    false,
		"auto_cascade_workshop":    false,
		"agent_max_turns":          60,
		"agent_timeout_seconds":    900,
		"agent_requires_approval":  true,
		"search_debounce_ms":       300,
		"toast_duration_ms":        5000,
		"delete_confirmation":      map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-disabled-ready", Title: "WS Disabled Ready", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-disabled-ready", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-finalize not to trigger when auto-advance is disabled")
	}
	if resp.AutoAdvance.Reason != "disabled" {
		t.Errorf("expected reason 'disabled', got %q", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "finalize" {
		t.Errorf("expected next mode 'finalize', got %v", resp.AutoAdvance.NextMode)
	}
}

func TestWorkshopSave_AgentDown_StillSaves(t *testing.T) {
	agent := &mockAgentService{
		err: fmt.Errorf("agent unavailable"),
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Enable auto-advance to test agent-down resilience.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                      "dark",
		"default_mode":               "manual",
		"max_auto_rounds":            10,
		"auto_initialize_workshop":   false,
		"auto_advance_workshop":      true,
		"auto_cascade_workshop":      false,
		"auto_advance_delay_seconds": 0,
		"agent_max_turns":            60,
		"agent_timeout_seconds":      900,
		"agent_requires_approval":    true,
		"search_debounce_ms":         300,
		"toast_duration_ms":          5000,
		"delete_confirmation":        map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

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

	body := makeWorkshopSaveBody(1, round)
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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered when agent is down")
	}
	if resp.AutoAdvance.Reason != "error" {
		t.Errorf("expected reason 'error', got %q", resp.AutoAdvance.Reason)
	}
	if resp.AutoAdvance.NextMode == nil || *resp.AutoAdvance.NextMode != "workshop" {
		t.Errorf("expected next mode 'workshop', got %v", resp.AutoAdvance.NextMode)
	}
}

func TestWorkshopSave_ConcurrentSaves_GuardPreventsDouble(t *testing.T) {
	// Simulate the centralized guard rejecting the spawn because another
	// agent is already active for this item.
	agent := &mockAgentService{
		err: agentactivity.ErrBacklogItemBusy,
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Enable auto-advance to trigger the spawn path.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                      "dark",
		"default_mode":               "manual",
		"max_auto_rounds":            10,
		"auto_initialize_workshop":   false,
		"auto_advance_workshop":      true,
		"auto_cascade_workshop":      false,
		"auto_advance_delay_seconds": 0,
		"agent_max_turns":            60,
		"agent_timeout_seconds":      900,
		"agent_requires_approval":    true,
		"search_debounce_ms":         300,
		"toast_duration_ms":          5000,
		"delete_confirmation":        map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-lock", Title: "WS Lock", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-lock", body))

	// Save succeeds even though auto-advance was blocked.
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	// Auto-advance should report agent_active (not error) when the guard blocks.
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered when agent is already active")
	}
	if resp.AutoAdvance.Reason != "agent_active" {
		t.Errorf("expected reason 'agent_active', got %q", resp.AutoAdvance.Reason)
	}
}

func TestWorkshopSave_OtherSelectedEmptyFreeform_NoAutoAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-other", TaskID: "task-other"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-other", Title: "WS Other", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// User selected __other__ but hasn't typed the freeform text yet.
	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items: []workshop.Item{
			{ID: "q1", Type: "decision", Selected: strPtr("A")},
			{ID: "q2", Type: "decision", Selected: strPtr(workshop.OtherKey)}, // no freeform
		},
	}

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-other", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered when __other__ has no freeform")
	}
	if resp.AutoAdvance.Reason != "pending_decisions" {
		t.Errorf("expected reason 'pending_decisions', got %q", resp.AutoAdvance.Reason)
	}
	if agent.lastReq != nil {
		t.Error("expected no agent spawn when __other__ is pending")
	}
}

func TestWorkshopSave_OtherSelectedWithFreeform_AutoAdvances(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-other-ok", TaskID: "task-other-ok"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-other-ok", Title: "WS Other OK", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	freeform := "I want a completely different approach"
	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items: []workshop.Item{
			{ID: "q1", Type: "decision", Selected: strPtr("A")},
			{ID: "q2", Type: "decision", Selected: strPtr(workshop.OtherKey), Freeform: &freeform},
		},
	}

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-other-ok", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.AutoAdvance.Triggered {
		t.Errorf("expected auto-advance to trigger when __other__ has freeform, reason=%s", resp.AutoAdvance.Reason)
	}
	if agent.lastReq == nil {
		t.Fatal("expected agent spawn to be called")
	}
}

func strPtr(s string) *string { return &s }
