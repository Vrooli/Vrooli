package execution

import (
	"testing"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

func TestBaselineSetStateTransitions(t *testing.T) {
	statuses := []BaselineSetStatus{
		BaselineSetStatusRequired,
		BaselineSetStatusScopeRepairRequired,
		BaselineSetStatusComplete,
		BaselineSetStatusPartial,
		BaselineSetStatusDegraded,
	}
	allowed := map[BaselineSetStatus]map[BaselineSetStatus]bool{
		BaselineSetStatusRequired: {
			BaselineSetStatusRequired: true, BaselineSetStatusScopeRepairRequired: true,
			BaselineSetStatusComplete: true, BaselineSetStatusPartial: true, BaselineSetStatusDegraded: true,
		},
		BaselineSetStatusScopeRepairRequired: {
			BaselineSetStatusScopeRepairRequired: true, BaselineSetStatusRequired: true, BaselineSetStatusDegraded: true,
		},
		BaselineSetStatusComplete: {BaselineSetStatusComplete: true},
		BaselineSetStatusPartial: {
			BaselineSetStatusPartial: true, BaselineSetStatusDegraded: true,
		},
		BaselineSetStatusDegraded: {BaselineSetStatusDegraded: true},
	}
	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				state := BaselineSetState{Status: from}
				want := allowed[from][to]
				if state.CanTransitionTo(to) != want {
					t.Fatalf("CanTransitionTo(%q) from %q = %v, want %v", to, from, state.CanTransitionTo(to), want)
				}
				err := state.TransitionTo(to)
				if want {
					if err != nil || state.Status != to {
						t.Fatalf("valid transition error=%v status=%q", err, state.Status)
					}
					return
				}
				if err == nil || state.Status != from {
					t.Fatalf("rejected transition error=%v mutated status to %q", err, state.Status)
				}
			})
		}
	}
}

func TestBaselineSetStateRejectsUnknownAndNilTransitions(t *testing.T) {
	state := BaselineSetState{Status: "future"}
	if state.CanTransitionTo(BaselineSetStatusRequired) {
		t.Fatal("unknown source status accepted")
	}
	if err := state.TransitionTo(BaselineSetStatusRequired); err == nil || state.Status != "future" {
		t.Fatalf("unknown source transition error=%v status=%q", err, state.Status)
	}
	state = BaselineSetState{Status: BaselineSetStatusRequired}
	if state.CanTransitionTo("future") {
		t.Fatal("unknown target status accepted")
	}
	if err := state.TransitionTo("future"); err == nil || state.Status != BaselineSetStatusRequired {
		t.Fatalf("unknown target transition error=%v status=%q", err, state.Status)
	}
	var nilState *BaselineSetState
	if err := nilState.TransitionTo(BaselineSetStatusRequired); err == nil {
		t.Fatal("nil state transition accepted")
	}
}

func TestBaselineSetStateValidateRejectsUnknownDurableStatus(t *testing.T) {
	if err := (BaselineSetState{Status: "future"}).Validate(); err == nil {
		t.Fatal("unknown persisted status accepted")
	}
	if err := (BaselineSetState{}).Validate(); err != nil {
		t.Fatalf("legacy empty status should remain readable: %v", err)
	}
}

func TestReceiptProjectionCannotReviveTerminalBaseline(t *testing.T) {
	state := Execution{BaselineSet: BaselineSetState{
		Status: BaselineSetStatusComplete, Required: 2, Ready: 2, Detail: "original terminal result",
	}}
	receipt := &validationv1.ValidationReceipt{State: validationv1.ReceiptState_RECEIPT_STATE_FAILED, Detail: "stale provider projection"}
	if err := (&service{}).projectBaselineReceipt(&state, receipt); err == nil {
		t.Fatal("terminal receipt was allowed to transition to partial")
	}
	if state.BaselineSet.Status != BaselineSetStatusComplete || state.BaselineSet.Required != 2 || state.BaselineSet.Ready != 2 || state.BaselineSet.Detail != "original terminal result" {
		t.Fatalf("rejected projection mutated terminal receipt: %+v", state.BaselineSet)
	}
}
