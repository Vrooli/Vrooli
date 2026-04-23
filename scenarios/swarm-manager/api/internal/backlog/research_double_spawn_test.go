package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"
)

// countingAgentService counts spawn calls for concurrency assertions.
type countingAgentService struct {
	spawnCount atomic.Int32
	result     agentmanager.RunResult
	err        error
}

func (c *countingAgentService) IsEnabled() bool                    { return true }
func (c *countingAgentService) IsAvailable(_ context.Context) bool { return true }
func (c *countingAgentService) ResolveURL(_ context.Context) (string, error) {
	return "http://agent", nil
}
func (c *countingAgentService) GetProfileID() string { return "" }
func (c *countingAgentService) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	c.spawnCount.Add(1)
	return c.result, c.err
}

func (c *countingAgentService) ContinueRun(_ context.Context, _ string, _ string) error {
	return nil
}

func (c *countingAgentService) SpawnResearch(_ context.Context, _ agentmanager.ResearchSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, nil
}

func (c *countingAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}

func (c *countingAgentService) GetRunDiff(_ context.Context, runID string) (agentmanager.RunDiff, error) {
	return agentmanager.RunDiff{RunID: runID}, nil
}
func (c *countingAgentService) StopRun(_ context.Context, _ string) error { return nil }

// writeReadyRound writes a workshop round with all decisions answered and high
// readiness scores so auto-advance would trigger.
func writeReadyRound(t *testing.T, rootDir string, kind BacklogKind, name string, roundNum int) {
	t.Helper()
	round := workshop.Round{
		RoundNum:    roundNum,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 1, "risk_awareness": 2},
		Items:       []workshop.Item{{ID: "d1", Type: "decision", Selected: strPtr("A")}},
	}
	content, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	itemDir := filepath.Join(rootDir, backlogKindDirs[kind], name)
	workshopDir := filepath.Join(itemDir, "workshop")
	testutil.MakeDir(t, workshopDir)
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"), string(content))
}

// enableAutoAdvanceWithDelay writes settings enabling auto-advance with the given delay.
func enableAutoAdvanceWithDelay(t *testing.T, rootDir string, delaySec int) {
	t.Helper()
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                      "dark",
		"default_mode":               "manual",
		"max_auto_rounds":            10,
		"auto_initialize_workshop":   false,
		"auto_advance_workshop":      true,
		"auto_cascade_workshop":      false,
		"auto_advance_delay_seconds": delaySec,
		"agent_max_turns":            600,
		"agent_timeout_seconds":      900,
		"agent_requires_approval":    true,
		"search_debounce_ms":         300,
		"toast_duration_ms":          5000,
		"delete_confirmation":        map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})
}

func researchRequest(kind, name, mode string) *http.Request {
	body := map[string]any{"mode": mode, "confirm": true}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/backlog/"+kind+"/"+name+"/research", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"kind": kind, "name": name})
	return req
}

// TestResearch_CancelsPendingAdvance verifies that when a deferred auto-advance
// is pending and the user manually triggers "Next Round" (Research with mode=workshop),
// the pending advance is cancelled and only one agent is spawned.
func TestResearch_CancelsPendingAdvance(t *testing.T) {
	agent := &countingAgentService{
		result: agentmanager.RunResult{RunID: "run-manual", TaskID: "task-manual"},
	}
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	enableAutoAdvanceWithDelay(t, rootDir, 30)

	h := NewHandlerWithClients(rootDir, agent, &promptmanager.MockClient{Result: "test prompt"})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "double-spawn", Title: "Double Spawn", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})
	writeReadyRound(t, rootDir, KindIdea, "double-spawn", 1)

	// Simulate a pending auto-advance (as if WorkshopSave had fired from auto-save).
	itemDir := filepath.Join(rootDir, "ideas", "double-spawn")
	pa := PendingAdvance{
		CreatedAt:  time.Now().UTC(),
		AdvanceAt:  time.Now().UTC().Add(30 * time.Second),
		NextMode:   "workshop",
		RoundCount: 1,
		Kind:       "idea",
		Name:       "double-spawn",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatal(err)
	}

	// User clicks "Next Round" → Research with mode=workshop.
	w := httptest.NewRecorder()
	h.Research(w, researchRequest("idea", "double-spawn", "workshop"))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Pending advance should be cancelled.
	if hasPendingAdvance(itemDir) {
		t.Error("expected pending advance to be cancelled after manual Research spawn")
	}

	// Only one agent should have been spawned.
	if count := agent.spawnCount.Load(); count != 1 {
		t.Errorf("expected exactly 1 spawn, got %d", count)
	}
}

// TestResearch_RejectsWhenAgentActive verifies that the Research handler
// returns 409 Conflict when the agent service returns ErrBacklogItemBusy
// (another agent is already working on this backlog item).
func TestResearch_RejectsWhenAgentActive(t *testing.T) {
	agent := &countingAgentService{
		err: agentactivity.ErrBacklogItemBusy,
	}
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)

	h := NewHandlerWithClients(rootDir, agent, &promptmanager.MockClient{Result: "test prompt"})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "busy-item", Title: "Busy Item", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})
	writeReadyRound(t, rootDir, KindIdea, "busy-item", 1)

	w := httptest.NewRecorder()
	h.Research(w, researchRequest("idea", "busy-item", "workshop"))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when agent is active, got %d: %s", w.Code, w.Body.String())
	}
}

// TestResearch_NoPendingCancelForNonWorkshopModes verifies that Research does
// NOT cancel pending advances for modes other than workshop/finalize (e.g.,
// initialize, clarify). Only workshop/finalize modes represent "the user is
// manually advancing" and should supersede auto-advance.
func TestResearch_NoPendingCancelForNonWorkshopModes(t *testing.T) {
	agent := &countingAgentService{
		result: agentmanager.RunResult{RunID: "run-init", TaskID: "task-init"},
	}
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	enableAutoAdvanceWithDelay(t, rootDir, 30)

	h := NewHandlerWithClients(rootDir, agent, &promptmanager.MockClient{Result: "test prompt"})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "init-item", Title: "Init Item", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Write a pending advance.
	itemDir := filepath.Join(rootDir, "ideas", "init-item")
	pa := PendingAdvance{
		CreatedAt:  time.Now().UTC(),
		AdvanceAt:  time.Now().UTC().Add(30 * time.Second),
		NextMode:   "workshop",
		RoundCount: 1,
		Kind:       "idea",
		Name:       "init-item",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatal(err)
	}

	// Research with mode=initialize should NOT touch the pending advance.
	w := httptest.NewRecorder()
	h.Research(w, researchRequest("idea", "init-item", "initialize"))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Pending advance should still exist.
	if !hasPendingAdvance(itemDir) {
		t.Error("expected pending advance to remain for non-workshop mode")
	}
}

// TestResearch_FinalizeCancelsPendingAdvance verifies that finalize mode also
// cancels pending advances, since finalization supersedes auto-advance.
func TestResearch_FinalizeCancelsPendingAdvance(t *testing.T) {
	agent := &countingAgentService{
		result: agentmanager.RunResult{RunID: "run-final", TaskID: "task-final"},
	}
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	enableAutoAdvanceWithDelay(t, rootDir, 30)

	h := NewHandlerWithClients(rootDir, agent, &promptmanager.MockClient{Result: "test prompt"})

	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "final-item", Title: "Final Item", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Write a ready round with high scores so finalize is allowed.
	round := workshop.Round{
		RoundNum:         1,
		GeneratedAt:      "2026-01-01T00:00:00Z",
		Readiness:        map[string]int{"problem_clarity": 5, "scope_defined": 5, "approach_solid": 5, "testable": 5, "risk_awareness": 5},
		Items:            []workshop.Item{{ID: "d1", Type: "decision", Selected: strPtr("A")}},
		PendingSynthesis: true,
	}
	content, _ := json.MarshalIndent(round, "", "  ")
	itemDir := filepath.Join(rootDir, "ideas", "final-item")
	workshopDir := filepath.Join(itemDir, "workshop")
	testutil.MakeDir(t, workshopDir)
	testutil.WriteFile(t, filepath.Join(workshopDir, "round-001.json"), string(content))

	// Write a pending advance.
	pa := PendingAdvance{
		CreatedAt:  time.Now().UTC(),
		AdvanceAt:  time.Now().UTC().Add(30 * time.Second),
		NextMode:   "finalize",
		RoundCount: 1,
		Kind:       "idea",
		Name:       "final-item",
	}
	if err := writePendingAdvance(itemDir, pa); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	h.Research(w, researchRequest("idea", "final-item", "finalize"))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Pending advance should be cancelled.
	if hasPendingAdvance(itemDir) {
		t.Error("expected pending advance to be cancelled after finalize")
	}

	// Only one spawn.
	if count := agent.spawnCount.Load(); count != 1 {
		t.Errorf("expected 1 spawn, got %d", count)
	}
}
