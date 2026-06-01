package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/google/uuid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestExecutionStateManager_SaveGetRoundTrip(t *testing.T) {
	container, cleanup := SetupTestDatabase(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	manager := NewExecutionStateManager(container.db)
	taskID := uuid.New().String()

	fs := findings.BuildState([]findings.Finding{
		{ID: "f1", Dimension: dimensions.Dimension("standards"), Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR},
	})

	state := manager.InitializeState(taskID, "balanced")
	state.Findings = fs
	state.Iteration = 2
	state.CurrentSkill = "refactor"
	state.CurrentRationale = "heaviest dimension standards"
	state.ScoreHistory = []float64{8, 4}
	state.Trace = []DecisionTraceEntry{{Iteration: 1, ChosenSkill: "refactor", ScoreBefore: 8, ScoreAfter: 4, RealizedDelta: 4}}

	if err := manager.Save(state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := manager.Get(taskID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected execution state, got nil")
	}
	if got.ProfileID != "balanced" {
		t.Errorf("profile id mismatch: %q", got.ProfileID)
	}
	if got.Iteration != 2 || got.CurrentSkill != "refactor" {
		t.Errorf("iteration/skill mismatch: %d/%q", got.Iteration, got.CurrentSkill)
	}
	if len(got.ScoreHistory) != 2 || got.ScoreHistory[1] != 4 {
		t.Errorf("score history mismatch: %v", got.ScoreHistory)
	}
	if len(got.Trace) != 1 || got.Trace[0].RealizedDelta != 4 {
		t.Errorf("trace mismatch: %+v", got.Trace)
	}
	if got.Findings.TotalScore != fs.TotalScore {
		t.Errorf("findings score mismatch: %v vs %v", got.Findings.TotalScore, fs.TotalScore)
	}
}

func TestExecutionStateManager_GetMissing(t *testing.T) {
	container, cleanup := SetupTestDatabase(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	manager := NewExecutionStateManager(container.db)
	state, err := manager.Get(uuid.New().String())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state for missing task, got %+v", state)
	}
}

func TestExecutionStateManager_FinalizeArchivesAndDeletes(t *testing.T) {
	container, cleanup := SetupTestDatabase(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	manager := NewExecutionStateManager(container.db)
	taskID := uuid.New().String()
	state := manager.InitializeState(taskID, "balanced")
	state.Iteration = 3
	state.Trace = []DecisionTraceEntry{
		{Iteration: 1, ChosenSkill: "refactor", RealizedDelta: 2},
		{Iteration: 2, ChosenSkill: "refactor", RealizedDelta: 1},
		{Iteration: 3, ChosenSkill: "test", RealizedDelta: 4},
	}
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if err := manager.FinalizeExecution(state, "demo-scenario"); err != nil {
		t.Fatalf("FinalizeExecution error: %v", err)
	}

	if got, _ := manager.Get(taskID); got != nil {
		t.Fatal("expected state deleted after finalize")
	}

	hist := NewHistoryService(container.db)
	rows, err := hist.GetHistory(HistoryFilters{ProfileID: "balanced"})
	if err != nil {
		t.Fatalf("GetHistory error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 archived execution, got %d", len(rows))
	}
	if rows[0].TotalIterations != 3 {
		t.Errorf("expected 3 total iterations, got %d", rows[0].TotalIterations)
	}
	// Two distinct skills in the trace → two breakdown rows.
	if len(rows[0].PhaseBreakdown) != 2 {
		t.Errorf("expected 2 skill-breakdown rows, got %d", len(rows[0].PhaseBreakdown))
	}
}

// TestExecutionStateManager_NonUUIDTaskID is the regression guard for the live
// P1 failure: real task IDs are strings like
// "scenario-improver-<name>-<timestamp>", not UUIDs. An earlier schema typed
// task_id as UUID, so the controller's very first state query threw
// "pq: invalid input syntax for type uuid" and the closed loop never engaged.
// Every prior test used uuid.New(), masking the bug. This one uses the real
// shape and exercises both task_id-keyed tables (profile_execution_state via
// Save/Get/Delete, profile_executions via FinalizeExecution + GetHistory).
func TestExecutionStateManager_NonUUIDTaskID(t *testing.T) {
	container, cleanup := SetupTestDatabase(t)
	if cleanup == nil {
		return
	}
	defer cleanup()

	manager := NewExecutionStateManager(container.db)
	taskID := "scenario-improver-bookmark-intelligence-hub-20260531-145604"

	state := manager.InitializeState(taskID, "balanced")
	state.Iteration = 3
	state.CurrentSkill = "refactor"
	state.Trace = []DecisionTraceEntry{
		{Iteration: 1, ChosenSkill: "refactor", ScoreBefore: 8, ScoreAfter: 6, RealizedDelta: 2},
		{Iteration: 2, ChosenSkill: "test", ScoreBefore: 6, ScoreAfter: 4, RealizedDelta: 2},
	}

	if err := manager.Save(state); err != nil {
		t.Fatalf("Save with non-UUID task ID returned error: %v", err)
	}

	got, err := manager.Get(taskID)
	if err != nil {
		t.Fatalf("Get with non-UUID task ID returned error: %v", err)
	}
	if got == nil || got.TaskID != taskID {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	if err := manager.FinalizeExecution(state, "bookmark-intelligence-hub"); err != nil {
		t.Fatalf("FinalizeExecution with non-UUID task ID returned error: %v", err)
	}

	hist := NewHistoryService(container.db)
	rows, err := hist.GetHistory(HistoryFilters{ProfileID: "balanced"})
	if err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 archived execution, got %d", len(rows))
	}
}
