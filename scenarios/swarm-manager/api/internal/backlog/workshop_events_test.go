package backlog

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/workshop"
)

// captureEventLogger is the minimum EventLogger surface used by the backlog
// package. Each emit method records its call so tests can assert on the
// stream without wiring a real event-log repo.
type captureEventLogger struct {
	mu                     sync.Mutex
	workshopRoundCompleted []workshopRoundEmit
	statusChanges          []statusChangeEmit
}

type workshopRoundEmit struct {
	EntityID string
	Payload  eventlog.WorkshopRoundPayload
}

type statusChangeEmit struct {
	EntityID, From, To string
}

func (l *captureEventLogger) EmitBacklogCreated(_, _, _ string, _ int, _, _ string) {}
func (l *captureEventLogger) EmitBacklogCreatedFromSource(_, _, _ string, _ int, _, _, _, _ string) {
	// no-op
}

func (l *captureEventLogger) EmitBacklogStatusChanged(entityID, from, to string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.statusChanges = append(l.statusChanges, statusChangeEmit{EntityID: entityID, From: from, To: to})
}
func (l *captureEventLogger) EmitBacklogPriorityChanged(_ string, _, _ int)              {}
func (l *captureEventLogger) EmitBacklogEffortChanged(_, _, _ string)                    {}
func (l *captureEventLogger) EmitBacklogDependencyAdded(_, _ string)                     {}
func (l *captureEventLogger) EmitBacklogDependencyRemoved(_, _ string)                   {}
func (l *captureEventLogger) EmitBacklogInitiativeChanged(_, _, _ string)                {}
func (l *captureEventLogger) EmitBacklogArchived(_, _, _ string)                         {}
func (l *captureEventLogger) EmitBacklogUnarchived(_, _ string)                          {}
func (l *captureEventLogger) EmitBacklogDeleted(_ string)                                {}
func (l *captureEventLogger) EmitBacklogViewed(_, _ string)                              {}
func (l *captureEventLogger) EmitClarificationStarted(_ string, _ int, _ string, _ bool) {}
func (l *captureEventLogger) EmitClarificationResolved(_ string, _ int, _ string, _ int, _ string) {
}
func (l *captureEventLogger) EmitClarificationAction(_ string, _ int, _ string, _ string) {}
func (l *captureEventLogger) EmitWorkshopRoundCompleted(entityID string, payload eventlog.WorkshopRoundPayload) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.workshopRoundCompleted = append(l.workshopRoundCompleted, workshopRoundEmit{
		EntityID: entityID, Payload: payload,
	})
}

// TestWorkshopSaveEmitsRoundCompleted is the Phase 3 regression guard.
// Before the fix, EmitWorkshopRoundCompleted existed on the interface but
// was never called from production code, so the Agent tab's workshop-rounds
// metric was permanently zero.
func TestWorkshopSaveEmitsRoundCompleted(t *testing.T) {
	agent := &mockAgentService{result: agentmanager.RunResult{RunID: "r", TaskID: "t"}}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "ws-emit", Title: "WS Emit", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	logger := &captureEventLogger{}
	h.SetEventLogger(logger)

	round := workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 3, "scope_defined": 3, "approach_solid": 3, "testable": 3, "risk_awareness": 3},
		Items:       []workshop.Item{{ID: "q1", Type: "decision", Selected: strPtr("A")}},
	}
	body := makeWorkshopSaveBody(1, round)

	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "ws-emit", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(logger.workshopRoundCompleted) != 1 {
		t.Fatalf("expected 1 workshop-round-completed emit, got %d", len(logger.workshopRoundCompleted))
	}
	got := logger.workshopRoundCompleted[0]
	if got.EntityID != "idea/ws-emit" || got.Payload.RoundNumber != 1 {
		t.Errorf("expected entity=idea/ws-emit round=1, got entity=%q round=%d", got.EntityID, got.Payload.RoundNumber)
	}
	if got.Payload.Kind != "idea" {
		t.Errorf("expected kind=idea, got kind=%q", got.Payload.Kind)
	}
	if got.Payload.ItemsTotal != 1 || got.Payload.ItemsAnswered != 1 {
		t.Errorf("expected ItemsTotal=1 ItemsAnswered=1, got %+v", got.Payload)
	}
}

// TestBacklogFailedToCompletedTriggersManualAccept is the integration test
// for Phase 2 wiring: the user transitions a backlog item from failed to
// completed, and the backlog handler invokes the ExecutionQueuer to flip
// the latest failed execution into a manual-accept.
func TestBacklogFailedToCompletedTriggersManualAccept(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	// Seed a backlog item in the failed state.
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "accept-me", Title: "Accept", Status: StatusFailed,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	// Override the executionQueuer with a mock that records calls.
	eq := &mockExecutionQueuer{
		manuallyAcceptedID: "exec-xyz", manuallyAcceptedOK: true,
	}
	h.SetExecutionQueuer(eq)

	w := doUpdate(t, h, "idea", "accept-me", map[string]any{"status": "completed"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(eq.manuallyAcceptedCalls) != 1 {
		t.Fatalf("expected 1 manual-accept call, got %d", len(eq.manuallyAcceptedCalls))
	}
	call := eq.manuallyAcceptedCalls[0]
	if call.Kind != "idea" || call.Name != "accept-me" {
		t.Errorf("expected idea/accept-me, got %s/%s", call.Kind, call.Name)
	}
}

// TestBacklogOtherTransitionDoesNotTriggerManualAccept guards the inverse:
// only failed→completed should trigger manual-accept. Other transitions
// (e.g., backlog→completed for research items that complete without an
// agent run) must be a no-op against executions.
func TestBacklogOtherTransitionDoesNotTriggerManualAccept(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindIdea, BacklogItem{
		Name: "skip-me", Title: "Skip", Status: StatusBacklog,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	})

	eq := &mockExecutionQueuer{}
	h.SetExecutionQueuer(eq)

	w := doUpdate(t, h, "idea", "skip-me", map[string]any{"status": "completed"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(eq.manuallyAcceptedCalls) != 0 {
		t.Fatalf("expected no manual-accept calls, got %d", len(eq.manuallyAcceptedCalls))
	}
}

var _ = httptest.NewRecorder // retain import for future expansion
