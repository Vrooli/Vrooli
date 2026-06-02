package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer/gameguard"
	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/ecosystem-manager/api/pkg/findings"
)

// recordingStore captures the last credit event so the gaming-credit assertions
// can inspect what the bandit would have learned.
type recordingStore struct {
	effectiveness.Store
	last effectiveness.CreditEvent
	n    int
}

func (r *recordingStore) Record(ev effectiveness.CreditEvent) error {
	r.last = ev
	r.n++
	return nil
}

func orchWithStore(s effectiveness.Store) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		effectiveness: s,
		traceStore:    NewTraceStore(nil),
	}
}

func sampleDiff() findings.Diff {
	return findings.Diff{
		ClosedByDimension:     map[dimensions.Dimension]int{dimensions.Dimension("standards"): 3},
		IntroducedByDimension: map[dimensions.Dimension]int{dimensions.Dimension("tests"): 1},
	}
}

func TestAssignCredit_GamedZeroesClosed(t *testing.T) {
	store := &recordingStore{}
	o := orchWithStore(store)
	state := &ProfileExecutionState{
		CurrentSkill: "refactor",
		Trace:        []DecisionTraceEntry{{Iteration: 1, HeaviestDimension: "standards"}},
	}

	gamed := gameguard.Result{Gamed: true, Causes: []gameguard.Cause{gameguard.CauseTestWeakening}}
	o.assignCredit(state, sampleDiff(), RunCost{TotalTokens: 1000}, gamed)

	if store.n != 1 {
		t.Fatalf("expected one credit event, got %d", store.n)
	}
	if len(store.last.ClosedByDimension) != 0 {
		t.Errorf("gamed iteration must earn ZERO closed credit, got %v", store.last.ClosedByDimension)
	}
	// Debt (introduced) is preserved so breakage still penalizes the skill.
	if store.last.IntroducedByDimension[dimensions.Dimension("tests")] != 1 {
		t.Errorf("introduced debt must be preserved, got %v", store.last.IntroducedByDimension)
	}
}

func TestAssignCredit_CleanKeepsClosed(t *testing.T) {
	store := &recordingStore{}
	o := orchWithStore(store)
	state := &ProfileExecutionState{
		CurrentSkill: "refactor",
		Trace:        []DecisionTraceEntry{{Iteration: 1, HeaviestDimension: "standards"}},
	}

	o.assignCredit(state, sampleDiff(), RunCost{TotalTokens: 1000}, gameguard.Result{})

	if store.last.ClosedByDimension[dimensions.Dimension("standards")] != 3 {
		t.Errorf("clean iteration must keep closed credit, got %v", store.last.ClosedByDimension)
	}
}

func TestRecordRealized_StampsGamingCauseAndVeto(t *testing.T) {
	o := orchWithStore(&recordingStore{})
	state := &ProfileExecutionState{TaskID: "t1", Trace: []DecisionTraceEntry{{Iteration: 1}}}

	gamed := gameguard.Result{Gamed: true, Causes: []gameguard.Cause{gameguard.CauseLedgerDeletion}}
	o.recordRealized(state, 5, 1, RunCost{}, findings.Diff{}, false, gamed)

	got := state.Trace[0]
	if got.GamingCause != "gamed:ledger-deletion" {
		t.Errorf("GamingCause = %q, want gamed:ledger-deletion", got.GamingCause)
	}
	if !got.VetoApplied {
		t.Error("gamed iteration must set VetoApplied")
	}
}

func TestRecordRealized_FlaggedForReviewNoVeto(t *testing.T) {
	o := orchWithStore(&recordingStore{})
	state := &ProfileExecutionState{TaskID: "t1", Trace: []DecisionTraceEntry{{Iteration: 1}}}

	flagged := gameguard.Result{FlaggedForReview: true}
	o.recordRealized(state, 5, 1, RunCost{}, findings.Diff{}, false, flagged)

	got := state.Trace[0]
	if got.GamingCause != "flagged-for-review" {
		t.Errorf("GamingCause = %q, want flagged-for-review", got.GamingCause)
	}
	if got.VetoApplied {
		t.Error("flagged-for-review must NOT set VetoApplied (no auto-penalty)")
	}
}
