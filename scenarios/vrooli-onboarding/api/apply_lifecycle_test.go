package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestApplyRunStatusTransitionMatrix(t *testing.T) {
	states := []applyRunStatus{
		applyRunPending,
		applyRunApplying,
		applyRunAlreadySatisfied,
		applyRunIndeterminate,
		applyRunCancelled,
		applyRunPartiallyApplied,
		applyRunApplied,
		applyRunConfigurationIncomplete,
		applyRunFailed,
		applyRunInterrupted,
	}
	allowed := map[applyRunStatus]map[applyRunStatus]bool{
		applyRunPending: {
			applyRunApplying: true, applyRunAlreadySatisfied: true,
			applyRunFailed: true, applyRunIndeterminate: true,
		},
		applyRunApplying: {
			applyRunCancelled: true, applyRunPartiallyApplied: true,
			applyRunApplied: true, applyRunConfigurationIncomplete: true,
			applyRunIndeterminate: true, applyRunInterrupted: true,
		},
		applyRunAlreadySatisfied:        {applyRunIndeterminate: true},
		applyRunIndeterminate:           {},
		applyRunCancelled:               {applyRunIndeterminate: true},
		applyRunPartiallyApplied:        {applyRunIndeterminate: true},
		applyRunApplied:                 {applyRunIndeterminate: true},
		applyRunConfigurationIncomplete: {applyRunIndeterminate: true},
		applyRunFailed:                  {applyRunIndeterminate: true},
		applyRunInterrupted:             {applyRunIndeterminate: true},
	}

	for _, current := range states {
		for _, next := range states {
			t.Run(string(current)+"_to_"+string(next), func(t *testing.T) {
				run := applyRun{Status: current}
				err := run.TransitionStatus(next)
				wantAllowed := current == next || allowed[current][next]
				if wantAllowed && err != nil {
					t.Fatalf("expected transition to be allowed: %v", err)
				}
				if !wantAllowed && err == nil {
					t.Fatalf("expected transition %q -> %q to be rejected", current, next)
				}
				want := current
				if wantAllowed {
					want = next
				}
				if run.Status != want {
					t.Fatalf("status changed to %q, want %q", run.Status, want)
				}
			})
		}
	}
}

func TestApplyItemOutcomeTransitionMatrix(t *testing.T) {
	states := []applyItemOutcome{
		applyItemPending,
		applyItemApplying,
		applyItemApplied,
		applyItemAlreadySatisfied,
		applyItemBlocked,
		applyItemFailed,
		applyItemTimedOut,
		applyItemNeedsElevation,
		applyItemNotApplicable,
		applyItemSkippedSelf,
	}
	allowed := map[applyItemOutcome]map[applyItemOutcome]bool{
		applyItemPending: {
			applyItemApplying: true, applyItemAlreadySatisfied: true, applyItemBlocked: true,
			applyItemNotApplicable: true, applyItemSkippedSelf: true,
		},
		applyItemApplying: {
			applyItemApplied: true, applyItemFailed: true,
			applyItemTimedOut: true, applyItemNeedsElevation: true,
		},
		applyItemApplied: {}, applyItemAlreadySatisfied: {}, applyItemBlocked: {},
		applyItemFailed: {}, applyItemTimedOut: {}, applyItemNeedsElevation: {},
		applyItemNotApplicable: {}, applyItemSkippedSelf: {},
	}
	for _, current := range states {
		for _, next := range states {
			t.Run(string(current)+"_to_"+string(next), func(t *testing.T) {
				result := applyItemResult{Outcome: current}
				err := result.TransitionOutcome(next)
				wantAllowed := current == next || allowed[current][next]
				if wantAllowed && err != nil {
					t.Fatalf("expected outcome transition to be allowed: %v", err)
				}
				if !wantAllowed && err == nil {
					t.Fatalf("expected outcome transition %q -> %q to be rejected", current, next)
				}
				want := current
				if wantAllowed {
					want = next
				}
				if result.Outcome != want {
					t.Fatalf("outcome changed to %q, want %q", result.Outcome, want)
				}
			})
		}
	}
}

func TestApplyItemOutcomeRejectsUnknownWithoutMutation(t *testing.T) {
	unknownCurrent := applyItemResult{Outcome: "future_outcome"}
	if err := unknownCurrent.TransitionOutcome(applyItemApplied); err == nil {
		t.Fatal("expected unknown current outcome to be rejected")
	}
	if unknownCurrent.Outcome != "future_outcome" {
		t.Fatalf("unknown current outcome changed to %q", unknownCurrent.Outcome)
	}
	validCurrent := applyItemResult{Outcome: applyItemPending}
	if err := validCurrent.TransitionOutcome("future_outcome"); err == nil {
		t.Fatal("expected unknown next outcome to be rejected")
	}
	if validCurrent.Outcome != applyItemPending {
		t.Fatalf("invalid transition mutated outcome to %q", validCurrent.Outcome)
	}
}

func TestApplyRunStatusRejectsUnknownAndNilWithoutMutation(t *testing.T) {
	unknownCurrent := applyRun{Status: "unknown"}
	if err := unknownCurrent.TransitionStatus(applyRunApplying); err == nil {
		t.Fatal("expected an unknown current status to be rejected")
	}
	if unknownCurrent.Status != "unknown" {
		t.Fatalf("unknown current status changed to %q", unknownCurrent.Status)
	}

	validCurrent := applyRun{Status: applyRunPending}
	if err := validCurrent.TransitionStatus("unknown"); err == nil {
		t.Fatal("expected an unknown next status to be rejected")
	}
	if validCurrent.Status != applyRunPending {
		t.Fatalf("invalid transition mutated status to %q", validCurrent.Status)
	}
	if err := (*applyRun)(nil).TransitionStatus(applyRunApplying); err == nil {
		t.Fatal("expected a nil apply run to be rejected")
	}
}

func TestApplyRunStatusKeepsPersistedJSONStringContract(t *testing.T) {
	var run applyRun
	if err := json.Unmarshal([]byte(`{"run_id":"apply-legacy","status":"applying","items":[{"outcome":"applying"}]}`), &run); err != nil {
		t.Fatalf("decode legacy run: %v", err)
	}
	if run.Status != applyRunApplying || run.Items[0].Outcome != applyItemApplying {
		t.Fatalf("decoded status = %q, want %q", run.Status, applyRunApplying)
	}
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("encode run: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode encoded JSON: %v", err)
	}
	if decoded["status"] != "applying" {
		t.Fatalf("serialized status = %#v, want legacy string applying", decoded["status"])
	}
	items := decoded["items"].([]any)
	if items[0].(map[string]any)["outcome"] != "applying" {
		t.Fatalf("serialized outcome = %#v, want legacy string applying", items[0].(map[string]any)["outcome"])
	}
}

func TestUnknownApplyRunStatusIsNotTerminal(t *testing.T) {
	if isTerminalApplyStatus("future_status") {
		t.Fatal("unknown status must not be treated as terminal and eligible for cleanup")
	}
	if !isTerminalApplyStatus("indeterminate") {
		t.Fatal("indeterminate status remains a terminal outcome")
	}
}

func TestPersistApplyRunRejectsUnknownAndProjectedStatuses(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	for _, status := range []applyRunStatus{"future_status", applyRunInterrupted} {
		t.Run(string(status), func(t *testing.T) {
			run := applyRun{ID: "invalid-status", Status: status}
			path, err := applyRunPath(run.ID)
			if err != nil {
				t.Fatalf("resolve run path: %v", err)
			}
			if err := persistApplyRun(run); err == nil {
				t.Fatalf("persistApplyRun accepted status %q", status)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("invalid status created persisted run file, stat error=%v", err)
			}
		})
	}
}

func TestPersistApplyRunRejectsUnknownItemOutcome(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	run := applyRun{ID: "invalid-item-outcome", Status: applyRunPending, Items: []applyItemResult{{Outcome: "future_outcome"}}}
	if err := persistApplyRun(run); err == nil {
		t.Fatal("persistApplyRun accepted an unknown item outcome")
	}
}
