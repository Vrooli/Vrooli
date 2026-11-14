package repairbudget

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestDispatchReserveAcknowledgeAndReplacementRefusal(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("op-1", "fp-1", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}

	reserved, err := ledger.Dispatch("op-1")
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if reserved.Status != DispatchReserved || !reserved.ReplacementAllowed {
		t.Fatalf("expected a reserved, replaceable operation, got %+v", reserved)
	}

	if _, err := ledger.Acknowledge("op-1", "swarm-manager/api", "run-100"); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	// Replaying the same acknowledgement is idempotent.
	if _, err := ledger.Acknowledge("op-1", "swarm-manager/api", "run-100"); err != nil {
		t.Fatalf("idempotent acknowledge: %v", err)
	}
	// A different run is a duplicate executor and must be refused.
	if _, err := ledger.Acknowledge("op-1", "swarm-manager/api", "run-999"); !errors.Is(err, ErrExecutorActive) {
		t.Fatalf("expected active-executor refusal, got %v", err)
	}
	active, err := ledger.Dispatch("op-1")
	if err != nil {
		t.Fatalf("dispatch after ack: %v", err)
	}
	if active.Status != DispatchActive || active.OwnerRun != "run-100" || active.ReplacementAllowed {
		t.Fatalf("expected the original executor retained, got %+v", active)
	}
	if _, err := ledger.Fallback("op-1", "swarm-manager/cli", "run-200"); !errors.Is(err, ErrExecutorActive) {
		t.Fatalf("fallback must not duplicate an active executor, got %v", err)
	}
}

func TestDispatchUncertainReconcileThenFallbackSameAllowance(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("op-2", "fp-2", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := ledger.Acknowledge("op-2", "swarm-manager/api", "run-lost"); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if _, err := ledger.MarkUncertain("op-2", "transport reset before start response"); err != nil {
		t.Fatalf("mark uncertain: %v", err)
	}
	if _, err := ledger.MarkUncertain("op-2", "repeat"); err != nil {
		t.Fatalf("idempotent uncertain: %v", err)
	}
	uncertain, err := ledger.Dispatch("op-2")
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if uncertain.Status != DispatchUncertain || uncertain.ReplacementAllowed {
		t.Fatalf("expected uncertain, non-replaceable operation, got %+v", uncertain)
	}
	// Replacement is forbidden while the original start is unresolved.
	if _, err := ledger.Fallback("op-2", "swarm-manager/cli", "run-fallback"); !errors.Is(err, ErrDispatchUncertain) {
		t.Fatalf("expected uncertain-dispatch refusal, got %v", err)
	}

	// Reconciling to no-active-executor makes a fallback safe under the same
	// charged operation; the fallback never charges again.
	reconciled, err := ledger.Reconcile("op-2", false, "", "owner has no run for the operation")
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled.Status != DispatchReconciled || !reconciled.ReplacementAllowed {
		t.Fatalf("expected reconciled, replaceable operation, got %+v", reconciled)
	}
	if _, err := ledger.Fallback("op-2", "swarm-manager/cli", "run-fallback"); err != nil {
		t.Fatalf("fallback after reconcile: %v", err)
	}
	totals, err := ledger.Totals()
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.Effort != 1 {
		t.Fatalf("fallback must debit the same allowance, got %+v", totals)
	}
	after, err := ledger.Dispatch("op-2")
	if err != nil {
		t.Fatalf("dispatch after fallback: %v", err)
	}
	if after.Status != DispatchActive || after.OwnerRun != "run-fallback" {
		t.Fatalf("expected the fallback executor active, got %+v", after)
	}
}

func TestDispatchReconcileActiveKeepsOriginal(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("op-3", "fp-3", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := ledger.MarkUncertain("op-3", "lost response"); err != nil {
		t.Fatalf("mark uncertain: %v", err)
	}
	state, err := ledger.Reconcile("op-3", true, "run-original", "owner reports run active")
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if state.Status != DispatchActive || state.OwnerRun != "run-original" || state.ReplacementAllowed {
		t.Fatalf("expected the reconciled original executor retained, got %+v", state)
	}
	if _, err := ledger.Fallback("op-3", "swarm-manager/cli", "run-second"); !errors.Is(err, ErrExecutorActive) {
		t.Fatalf("expected active-executor refusal, got %v", err)
	}
}

func TestDispatchCancelFencesDelayedStart(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("op-4", "fp-4", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := ledger.Cancel("op-4", "operator cancelled before owner start"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	terminal, err := ledger.Dispatch("op-4")
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if terminal.Status != DispatchFinished || terminal.ReplacementAllowed {
		t.Fatalf("expected a terminal operation, got %+v", terminal)
	}
	// A delayed owner start and a fallback are both fenced by the terminal state.
	if _, err := ledger.Acknowledge("op-4", "swarm-manager/api", "run-late"); !errors.Is(err, ErrAttemptTerminal) {
		t.Fatalf("expected terminal refusal on delayed ack, got %v", err)
	}
	if _, err := ledger.Fallback("op-4", "swarm-manager/cli", "run-late"); !errors.Is(err, ErrAttemptTerminal) {
		t.Fatalf("expected terminal refusal on fallback, got %v", err)
	}
}

func TestDispatchStateSurvivesRestart(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("op-5", "fp-5", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := ledger.Acknowledge("op-5", "swarm-manager/api", "run-500"); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if _, err := ledger.MarkUncertain("op-5", "callback lost after start"); err != nil {
		t.Fatalf("uncertain: %v", err)
	}

	restarted := New("aquila", filepath.Join(filepath.Dir(ledger.path), "repair-ledger.json"))
	restarted.SetClock(ledger.now)
	state, err := restarted.Dispatch("op-5")
	if err != nil {
		t.Fatalf("dispatch after restart: %v", err)
	}
	if state.Status != DispatchUncertain || state.OwnerRun != "run-500" || state.ReplacementAllowed {
		t.Fatalf("dispatch state did not survive restart: %+v", state)
	}
	outstanding, err := restarted.Outstanding()
	if err != nil {
		t.Fatalf("outstanding: %v", err)
	}
	if len(outstanding) != 1 || outstanding[0].AttemptID != "op-5" {
		t.Fatalf("expected one outstanding uncertain operation, got %+v", outstanding)
	}
	if _, err := restarted.Reconcile("op-5", false, "", "owner confirms no run"); err != nil {
		t.Fatalf("reconcile after restart: %v", err)
	}
	outstanding, err = restarted.Outstanding()
	if err != nil {
		t.Fatalf("outstanding after reconcile: %v", err)
	}
	if len(outstanding) != 1 || outstanding[0].Status != DispatchReconciled {
		t.Fatalf("reconciled operation must remain outstanding until terminal, got %+v", outstanding)
	}
}

func TestDispatchConcurrentReplacementOnlyOneExecutor(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Begin(testLimits(), repairEvent("op-6", "fp-6", "swarm-manager", false)); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := ledger.MarkUncertain("op-6", "lost"); err != nil {
		t.Fatalf("uncertain: %v", err)
	}
	if _, err := ledger.Reconcile("op-6", false, "", "no active run"); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if _, err := ledger.Fallback("op-6", "swarm-manager/api", "run-concurrent"); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if succeeded != 1 {
		t.Fatalf("expected exactly one fallback executor, got %d", succeeded)
	}
	state, err := ledger.Dispatch("op-6")
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if state.Status != DispatchActive || state.OwnerRun != "run-concurrent" {
		t.Fatalf("expected one active executor, got %+v", state)
	}
}

func TestDispatchUnknownOperationRefused(t *testing.T) {
	ledger := newLedger(t, "aquila")
	if _, err := ledger.Acknowledge("missing", "owner", "run"); !errors.Is(err, ErrUnknownAttempt) {
		t.Fatalf("expected unknown-attempt refusal, got %v", err)
	}
	if _, err := ledger.MarkUncertain("missing", "lost"); !errors.Is(err, ErrUnknownAttempt) {
		t.Fatalf("expected unknown-attempt refusal, got %v", err)
	}
	if _, err := ledger.Reconcile("missing", false, "", ""); !errors.Is(err, ErrUnknownAttempt) {
		t.Fatalf("expected unknown-attempt refusal, got %v", err)
	}
	if _, err := ledger.Fallback("missing", "owner", "run"); !errors.Is(err, ErrUnknownAttempt) {
		t.Fatalf("expected unknown-attempt refusal, got %v", err)
	}
}
