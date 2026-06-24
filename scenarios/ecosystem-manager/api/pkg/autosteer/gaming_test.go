package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer/gameguard"
)

func newGamingOrch() *ExecutionOrchestrator {
	return &ExecutionOrchestrator{traceStore: NewTraceStore(nil)}
}

func TestRecordRealized_StampsGamingCause(t *testing.T) {
	o := newGamingOrch()
	state := &ProfileExecutionState{TaskID: "t1", Trace: []DecisionTraceEntry{{Iteration: 1}}}

	gamed := gameguard.Result{Gamed: true, Causes: []gameguard.Cause{gameguard.CauseLedgerDeletion}}
	o.recordRealized(state, 5, 1, gamed)

	if got := state.Trace[0].GamingCause; got != "gamed:ledger-deletion" {
		t.Errorf("GamingCause = %q, want gamed:ledger-deletion", got)
	}
}

func TestRecordRealized_FlaggedForReview(t *testing.T) {
	o := newGamingOrch()
	state := &ProfileExecutionState{TaskID: "t1", Trace: []DecisionTraceEntry{{Iteration: 1}}}

	flagged := gameguard.Result{FlaggedForReview: true}
	o.recordRealized(state, 5, 1, flagged)

	if got := state.Trace[0].GamingCause; got != "flagged-for-review" {
		t.Errorf("GamingCause = %q, want flagged-for-review", got)
	}
}

func TestRecordRealized_CleanNoCause(t *testing.T) {
	o := newGamingOrch()
	state := &ProfileExecutionState{TaskID: "t1", Trace: []DecisionTraceEntry{{Iteration: 1}}}

	o.recordRealized(state, 5, 1, gameguard.Result{})

	if got := state.Trace[0].GamingCause; got != "" {
		t.Errorf("clean iteration GamingCause = %q, want empty", got)
	}
}

// RunGamed is the promote-safety gate: a gamed iteration anywhere in a run's
// durable trace blocks the shadow→live promote; a clean run is promotable.
func TestRunGamed(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	defer cleanup()
	store := NewTraceStore(pg.db)
	o := &ExecutionOrchestrator{traceStore: store}

	gamed := DecisionTraceEntry{Iteration: 1, ChosenSkill: "refactor"}
	if err := store.Append("t-gamed", "demo", "demo", gamed); err != nil {
		t.Fatalf("Append: %v", err)
	}
	gamed.GamingCause = "gamed:test-weakening"
	if err := store.SetRealized("t-gamed", gamed); err != nil {
		t.Fatalf("SetRealized: %v", err)
	}
	if blocked, err := o.RunGamed("t-gamed"); err != nil || !blocked {
		t.Fatalf("RunGamed(gamed) = %v, %v; want true, nil", blocked, err)
	}

	clean := DecisionTraceEntry{Iteration: 1, ChosenSkill: "refactor"}
	if err := store.Append("t-clean", "demo", "demo", clean); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if blocked, err := o.RunGamed("t-clean"); err != nil || blocked {
		t.Fatalf("RunGamed(clean) = %v, %v; want false, nil", blocked, err)
	}
}
