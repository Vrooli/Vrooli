package stagedmessages

import (
	"context"
	"testing"
	"time"

	"git-control-tower/internal/reviewjobs"
)

func TestMetricsAggregateDurableLifecycleEvidence(t *testing.T) {
	_, store, _, _ := serviceFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	created := now.Add(-10 * time.Minute)
	create := func(job reviewjobs.Job) {
		t.Helper()
		job.CreatedAt = created
		if _, _, err := store.Create(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	create(reviewjobs.Job{ID: "failed", IdempotencyKey: "failed", InputDigest: "f", Operation: operation, RepositoryID: "repo", State: reviewjobs.Failed, ErrorCode: "provider_admission_maintenance", UpdatedAt: now.Add(-8 * time.Minute), ExecutionStartedKnown: true, ExecutionStarted: false, TokensKnown: true, ChargeKnown: true})
	create(reviewjobs.Job{ID: "dispatch", IdempotencyKey: "dispatch", InputDigest: "d", Operation: operation, RepositoryID: "repo", State: reviewjobs.Running, UpdatedAt: now.Add(-7 * time.Minute)})
	create(reviewjobs.Job{ID: "reconcile", IdempotencyKey: "reconcile", InputDigest: "r", Operation: operation, RepositoryID: "repo", State: reviewjobs.Running, ExecutionID: "owner-1", UpdatedAt: now.Add(-6 * time.Minute)})
	create(reviewjobs.Job{ID: "success", IdempotencyKey: "success", InputDigest: "s", Operation: operation, RepositoryID: "repo", State: reviewjobs.Succeeded, UpdatedAt: now.Add(-5 * time.Minute)})
	create(reviewjobs.Job{ID: "other-op", IdempotencyKey: "other-op", InputDigest: "o", Operation: "review", RepositoryID: "repo", State: reviewjobs.Failed, UpdatedAt: now})
	_, _, err := store.Create(ctx, reviewjobs.Job{ID: "retry", IdempotencyKey: "failed", InputDigest: "different", Operation: operation, RepositoryID: "repo"})
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{store: store, now: func() time.Time { return now }}
	got, err := svc.Metrics(ctx, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got.Jobs != 4 || got.TerminalJobs != 1 || got.AwaitingProviderJobs != 1 || got.FailedJobs != 0 || got.FailureRate == nil || *got.FailureRate != 0 {
		t.Fatalf("unexpected lifecycle totals: %+v", got)
	}
	if got.FailuresByCode["provider_admission_maintenance"] != 1 || got.ZeroWorkRefusals != 1 {
		t.Fatalf("failure/zero-work classification incorrect: %+v", got)
	}
	if got.DuplicateRetries != 1 || got.StuckDispatch != 1 || got.StuckReconciliation != 1 {
		t.Fatalf("retry or stuck counts incorrect: %+v", got)
	}
	if got.TerminalDurationCount != 1 || got.TerminalDurationMillis != 0 {
		t.Fatalf("terminal duration incorrect: %+v", got)
	}
}

func TestMetricsTracksDispatchingUntilOwnerHandleIsSaved(t *testing.T) {
	_, store, _, _ := serviceFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	staleAt := now.Add(-10 * time.Minute)
	dispatching, _, err := store.Create(ctx, reviewjobs.Job{
		ID: "dispatching-no-handle", IdempotencyKey: "dispatching-no-handle", InputDigest: "d",
		Operation: operation, RepositoryID: "repo", State: reviewjobs.Dispatching,
		CreatedAt: staleAt, UpdatedAt: staleAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{store: store, now: func() time.Time { return now }}
	before, err := svc.Metrics(ctx, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if before.StuckDispatch != 1 || before.StuckReconciliation != 0 {
		t.Fatalf("stale dispatch without an owner handle was not counted correctly: %+v", before)
	}

	updated := dispatching
	if err := updated.Transition(reviewjobs.Running); err != nil {
		t.Fatal(err)
	}
	updated.ExecutionID = "owner-execution"
	changed, err := store.UpdateAdvisory(ctx, dispatching, updated)
	if err != nil || !changed {
		t.Fatalf("save owner acknowledgement: changed=%t err=%v", changed, err)
	}
	afterAck, err := svc.Metrics(ctx, time.Now().Add(time.Minute), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if afterAck.StuckDispatch != 0 || afterAck.StuckReconciliation != 0 {
		t.Fatalf("owner acknowledgement did not clear the stale-dispatch signal: %+v", afterAck)
	}
	afterReconciliationDelay, err := svc.Metrics(ctx, time.Now().Add(6*time.Minute), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if afterReconciliationDelay.StuckDispatch != 0 || afterReconciliationDelay.StuckReconciliation != 1 {
		t.Fatalf("owner-backed job did not move to reconciliation monitoring: %+v", afterReconciliationDelay)
	}
}

func TestMetricsTracksQueuedAdmissionsAwaitingRecovery(t *testing.T) {
	_, store, _, _ := serviceFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	staleAt := now.Add(-10 * time.Minute)
	if _, _, err := store.Create(ctx, reviewjobs.Job{
		ID: "queued-before-dispatch", IdempotencyKey: "queued-before-dispatch", InputDigest: "q",
		Operation: operation, RepositoryID: "repo", State: reviewjobs.Queued,
		CreatedAt: staleAt, UpdatedAt: staleAt,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := (&Service{store: store, now: func() time.Time { return now }}).Metrics(ctx, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got.StuckDispatch != 1 {
		t.Fatalf("old queued admission was invisible to stuck-dispatch monitoring: %+v", got)
	}
}

func TestMetricsCountsProviderWaitWithoutCallingItFailedOrStuck(t *testing.T) {
	_, store, _, _ := serviceFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	old := now.Add(-20 * time.Minute)
	_, _, err := store.Create(ctx, reviewjobs.Job{
		ID: "awaiting-provider", IdempotencyKey: "awaiting-provider", InputDigest: "p",
		Operation: operation, RepositoryID: "repo", State: reviewjobs.AwaitingProvider,
		ErrorCode: "provider_admission_maintenance", ExecutionStartedKnown: true,
		ExecutionStarted: false, TokensKnown: true, ChargeKnown: true,
		CreatedAt: old, UpdatedAt: old,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := (&Service{store: store, now: func() time.Time { return now }}).Metrics(ctx, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got.AwaitingProviderJobs != 1 || got.TerminalJobs != 0 || got.FailedJobs != 0 || got.FailuresByCode["provider_admission_maintenance"] != 1 || got.ZeroWorkRefusals != 1 || got.StuckDispatch != 0 || got.StuckReconciliation != 0 {
		t.Fatalf("provider wait should be visible, non-failed, zero-work, and not stuck: %+v", got)
	}
}

func TestMetricsClassifiesOnlyProvenLegacyMaintenanceRowsAsProviderWait(t *testing.T) {
	_, store, _, _ := serviceFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	old := now.Add(-20 * time.Minute)
	for _, job := range []reviewjobs.Job{
		{
			ID: "legacy-proven-wait", IdempotencyKey: "legacy-proven-wait", InputDigest: "p",
			Operation: operation, RepositoryID: "repo", State: reviewjobs.Failed,
			ErrorCode: "provider_admission_maintenance", ExecutionStartedKnown: true,
			ExecutionStarted: false, TokensKnown: true, TokensUsed: 0, ChargeKnown: false,
			CreatedAt: old, UpdatedAt: old,
		},
		{
			ID: "legacy-unknown-failure", IdempotencyKey: "legacy-unknown-failure", InputDigest: "u",
			Operation: operation, RepositoryID: "repo", State: reviewjobs.Failed,
			CreatedAt: old, UpdatedAt: old,
		},
	} {
		if _, _, err := store.Create(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	got, err := (&Service{store: store, now: func() time.Time { return now }}).Metrics(ctx, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got.Jobs != 2 || got.AwaitingProviderJobs != 1 || got.TerminalJobs != 1 || got.FailedJobs != 1 || got.FailuresByCode["provider_admission_maintenance"] != 1 || got.FailuresByCode["unclassified"] != 1 || got.ZeroWorkRefusals != 0 || got.StuckDispatch != 0 || got.StuckReconciliation != 0 {
		t.Fatalf("legacy projection metrics lost proof, charged certainty, or unknown history: %+v", got)
	}
}
