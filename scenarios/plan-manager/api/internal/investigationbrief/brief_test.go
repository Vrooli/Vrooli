package investigationbrief

import (
	"context"
	"errors"
	"testing"
	"time"

	internalexecution "plan-manager/internal/execution"
	planmodel "plan-manager/internal/planmodel"
	internalvalidation "plan-manager/internal/validation"

	"github.com/vrooli/api-core/scheduletest"
)

type fakePlans struct{ plan planmodel.Plan }

func (f fakePlans) GetPlan(context.Context, string) (planmodel.Plan, error) { return f.plan, nil }

type fakeExecutions struct {
	internalexecution.Repository
	execution internalexecution.Execution
}

func (f fakeExecutions) GetExecution(context.Context, string) (internalexecution.Execution, bool, error) {
	return f.execution, true, nil
}

type fakeOperations struct {
	internalvalidation.OperationStore
	operation internalvalidation.ValidationOperation
}

func (f fakeOperations) GetOperation(context.Context, string) (internalvalidation.ValidationOperation, bool, error) {
	return f.operation, true, nil
}

func (fakeOperations) ListNonTerminalOperations(context.Context) ([]internalvalidation.ValidationOperation, error) {
	return nil, nil
}

func testProvider(op internalvalidation.ValidationOperation) Provider {
	return NewProvider(
		fakePlans{plan: planmodel.Plan{ID: "plan-1", ContentHash: "sha256:plan", TargetOutcome: "trusted diagnosis", Phases: []planmodel.Phase{{ID: "phase-1", Acceptance: "receipt is linked"}}}},
		fakeExecutions{execution: internalexecution.Execution{ID: "exec-1", PlanID: "plan-1", RunID: "run-1", CurrentPhaseID: "phase-1", StartedAt: "2026-09-06T12:50:00Z", PhaseValidationGenerations: map[string]int{"phase-1": 3}}},
		fakeOperations{operation: op},
		scheduletest.New(time.Date(2026, 9, 6, 13, 0, 0, 0, time.UTC)),
	)
}

func TestBriefReportsExplicitProducerWaitAndAuthoritativeIdentity(t *testing.T) {
	brief, err := testProvider(internalvalidation.ValidationOperation{
		ID: "op-1", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "exec-1", Status: internalvalidation.OperationQueued,
		QueuedAt: "2026-09-06T12:59:00Z", QueueReason: "producer capacity", QueueBudgetSeconds: 90,
		ExecutionBudgetSeconds: 600, ProducerWaitArgv: []string{"gct", "wait", "op-1"}, SyncArgv: []string{"plan-manager", "validation", "sync", "op-1"},
	}).Get(context.Background(), Request{ExecutionID: "exec-1", PhaseID: "phase-1", RunIDs: []string{"run-1"}, ValidationOperationID: "op-1"})
	if err != nil {
		t.Fatal(err)
	}
	if brief.Status != "ready" || brief.RunIDs[0] != "run-1" || brief.PhaseGeneration != 3 {
		t.Fatalf("unexpected identity: %+v", brief)
	}
	if brief.ProducerWait == nil || !brief.ProducerWait.Explicit || brief.KnownWaitSeconds != 60 {
		t.Fatalf("expected explicit wait: %+v", brief)
	}
	if brief.Budget.QueueSeconds != 90 || !brief.Budget.Known {
		t.Fatalf("expected budget: %+v", brief.Budget)
	}
	if brief.ActiveWorkSeconds != brief.WallTimeSeconds-brief.KnownWaitSeconds {
		t.Fatalf("wait must be separated from active work: %+v", brief)
	}
}

func TestBriefRejectsCallerRunSetMismatchAndStalePhase(t *testing.T) {
	provider := testProvider(internalvalidation.ValidationOperation{})
	_, err := provider.Get(context.Background(), Request{ExecutionID: "exec-1", PhaseID: "phase-1", RunIDs: []string{"other-run"}})
	if !errors.Is(err, ErrRunSetMismatch) {
		t.Fatalf("expected run-set mismatch, got %v", err)
	}
	_, err = provider.Get(context.Background(), Request{ExecutionID: "exec-1", PhaseID: "old-phase"})
	if !errors.Is(err, ErrPhaseStale) {
		t.Fatalf("expected stale phase, got %v", err)
	}
}

func TestBriefDoesNotInventProgressWithoutValidationEvidence(t *testing.T) {
	brief, err := testProvider(internalvalidation.ValidationOperation{}).Get(context.Background(), Request{ExecutionID: "exec-1", PhaseID: "phase-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.MaterialProgress) != 0 {
		t.Fatalf("unrelated changes must not become progress: %+v", brief.MaterialProgress)
	}
	if len(brief.OmittedReasons) == 0 {
		t.Fatal("expected explicit omission reason")
	}
}

func TestBriefUsesTerminalValidationReceiptAsProgressEvidence(t *testing.T) {
	brief, err := testProvider(internalvalidation.ValidationOperation{
		ID: "op-2", PlanID: "plan-1", PhaseID: "phase-1", ExecutionID: "exec-1", Status: internalvalidation.OperationTerminal,
		Result: &internalvalidation.Result{ID: "receipt-2", Detail: "scope verified", RanAt: "2026-09-06T12:59:30Z"},
	}).Get(context.Background(), Request{ExecutionID: "exec-1", PhaseID: "phase-1", ValidationOperationID: "op-2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.MaterialProgress) != 1 || brief.MaterialProgress[0].EvidenceRef != "validation:receipt-2" {
		t.Fatalf("expected receipt evidence: %+v", brief.MaterialProgress)
	}
}
