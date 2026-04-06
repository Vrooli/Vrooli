package backlog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"
)

func TestWritePendingAdvance_CreatesFile(t *testing.T) {
	itemDir := t.TempDir()
	pa := PendingAdvance{
		CreatedAt:  time.Now().UTC(),
		AdvanceAt:  time.Now().UTC().Add(10 * time.Second),
		NextMode:   "workshop",
		RoundCount: 2,
		Kind:       "idea",
		Name:       "test-item",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatalf("writePendingAdvance failed: %v", err)
	}
	testutil.AssertFileExists(t, filepath.Join(itemDir, pendingAdvanceFile))
}

func TestReadPendingAdvance_ReadsFile(t *testing.T) {
	itemDir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	pa := PendingAdvance{
		CreatedAt:  now,
		AdvanceAt:  now.Add(10 * time.Second),
		NextMode:   "finalize",
		RoundCount: 5,
		Kind:       "fix",
		Name:       "bug-item",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatal(err)
	}

	got, err := readPendingAdvance(itemDir)
	if err != nil {
		t.Fatalf("readPendingAdvance failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil PendingAdvance")
	}
	if got.NextMode != "finalize" {
		t.Errorf("expected next_mode 'finalize', got %q", got.NextMode)
	}
	if got.Kind != "fix" {
		t.Errorf("expected kind 'fix', got %q", got.Kind)
	}
}

func TestReadPendingAdvance_NotFound(t *testing.T) {
	itemDir := t.TempDir()
	got, err := readPendingAdvance(itemDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestDeletePendingAdvance_RemovesFile(t *testing.T) {
	itemDir := t.TempDir()
	pa := PendingAdvance{
		CreatedAt: time.Now().UTC(),
		AdvanceAt: time.Now().UTC().Add(10 * time.Second),
		NextMode:  "workshop",
		Kind:      "idea",
		Name:      "test",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatal(err)
	}
	if !deletePendingAdvance(itemDir) {
		t.Error("expected deletePendingAdvance to return true")
	}
	if hasPendingAdvance(itemDir) {
		t.Error("expected hasPendingAdvance to return false after delete")
	}
}

func TestDeletePendingAdvance_NoFile(t *testing.T) {
	itemDir := t.TempDir()
	if deletePendingAdvance(itemDir) {
		t.Error("expected deletePendingAdvance to return false when no file exists")
	}
}

func TestWorkshopSave_WithDelay_CreatesPendingFile(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-delayed", TaskID: "task-delayed"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Enable auto-advance with a 10-second delay.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, ".vrooli", "settings.json"), map[string]any{
		"theme":                       "dark",
		"default_mode":                "manual",
		"max_auto_rounds":             10,
		"auto_initialize_workshop":    false,
		"auto_advance_workshop":       true,
		"auto_cascade_workshop":       false,
		"auto_advance_delay_seconds":  10,
		"agent_max_turns":             60,
		"agent_timeout_seconds":       900,
		"agent_requires_approval":     true,
		"search_debounce_ms":          300,
		"toast_duration_ms":           5000,
		"confirm_destructive_actions": true,
	})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-delayed", Title: "WS Delayed", Status: StatusBacklog,
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
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-delayed", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp workshopSaveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should NOT be triggered immediately.
	if resp.AutoAdvance.Triggered {
		t.Error("expected auto-advance NOT triggered immediately when delay > 0")
	}

	// Agent should NOT have been called.
	if agent.lastReq != nil {
		t.Error("expected no agent spawn when delay > 0")
	}

	// Pending advance file should exist.
	itemDir := filepath.Join(rootDir, "ideas", "ws-delayed")
	if !hasPendingAdvance(itemDir) {
		t.Error("expected pending advance file to exist")
	}
}

func TestWorkshopCancelPendingAdvance_Cancels(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-cancel", Title: "WS Cancel", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Write a pending advance file.
	itemDir := filepath.Join(rootDir, "ideas", "ws-cancel")
	pa := PendingAdvance{
		CreatedAt: time.Now().UTC(),
		AdvanceAt: time.Now().UTC().Add(10 * time.Second),
		NextMode:  "workshop",
		Kind:      "idea",
		Name:      "ws-cancel",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-cancel/workshop/pending-advance", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-cancel"})
	w := httptest.NewRecorder()
	h.WorkshopCancelPendingAdvance(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Cancelled bool `json:"cancelled"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Cancelled {
		t.Error("expected cancelled=true")
	}
	if hasPendingAdvance(itemDir) {
		t.Error("expected pending advance file to be deleted")
	}
}

func TestWorkshopCancelPendingAdvance_NoPending(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-no-pending", Title: "WS No Pending", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	req := httptest.NewRequest("DELETE", "/api/v1/backlog/idea/ws-no-pending/workshop/pending-advance", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "idea", "name": "ws-no-pending"})
	w := httptest.NewRecorder()
	h.WorkshopCancelPendingAdvance(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Cancelled bool `json:"cancelled"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Cancelled {
		t.Error("expected cancelled=false when no pending advance exists")
	}
}

func TestWorkshopSave_NewSaveReplacesPendingAdvance(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-replace", TaskID: "task-replace"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	testutil.WriteJSONFile(t, filepath.Join(rootDir, ".vrooli", "settings.json"), map[string]any{
		"theme":                       "dark",
		"default_mode":                "manual",
		"max_auto_rounds":             10,
		"auto_initialize_workshop":    false,
		"auto_advance_workshop":       true,
		"auto_cascade_workshop":       false,
		"auto_advance_delay_seconds":  30,
		"agent_max_turns":             60,
		"agent_timeout_seconds":       900,
		"agent_requires_approval":     true,
		"search_debounce_ms":          300,
		"toast_duration_ms":           5000,
		"confirm_destructive_actions": true,
	})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-replace", Title: "WS Replace", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Write an old pending advance.
	itemDir := filepath.Join(rootDir, "ideas", "ws-replace")
	oldPa := PendingAdvance{
		CreatedAt: time.Now().UTC().Add(-20 * time.Second),
		AdvanceAt: time.Now().UTC().Add(10 * time.Second),
		NextMode:  "workshop",
		Kind:      "idea",
		Name:      "ws-replace",
	}
	if err := writePendingAdvance(itemDir, oldPa); err != nil {
		t.Fatal(err)
	}

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}

	body := makeWorkshopSaveBody(1, round)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-replace", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify pending advance file was replaced (new advance_at should be ~30s from now).
	pa, err := readPendingAdvance(itemDir)
	if err != nil {
		t.Fatalf("failed to read pending advance: %v", err)
	}
	if pa == nil {
		t.Fatal("expected pending advance file to exist")
	}
	// The new advance_at should be further out than the old one.
	if !pa.AdvanceAt.After(time.Now().UTC().Add(25 * time.Second)) {
		t.Errorf("expected new advance_at to be ~30s from now, got %v", pa.AdvanceAt)
	}
}
