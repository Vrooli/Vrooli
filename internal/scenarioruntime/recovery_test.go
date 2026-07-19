package scenarioruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRecoveryPolicyIsExplicitDurableAndFailsClosed(t *testing.T) {
	ctx := context.Background()
	clock := newFixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clock)

	if _, err := store.GetRecoveryPolicy(ctx, "critical-api", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetRecoveryPolicy without declaration error = %v, want ErrNotFound", err)
	}
	policy, err := store.UpsertRecoveryPolicy(ctx, RecoveryPolicy{
		Scenario: "critical-api", Critical: true, DependencyTier: 2,
		Enabled: true, RetryBudget: 3,
	})
	if err != nil {
		t.Fatalf("UpsertRecoveryPolicy: %v", err)
	}
	if policy.Variant != DefaultVariant || !policy.Critical || !policy.Enabled || policy.OptOut {
		t.Fatalf("policy = %#v, want explicit enabled critical live policy", policy)
	}
	clock.Advance(time.Minute)
	policy.OptOut = true
	policy.Enabled = false
	updated, err := store.UpsertRecoveryPolicy(ctx, policy)
	if err != nil {
		t.Fatalf("UpsertRecoveryPolicy(update): %v", err)
	}
	if !updated.UpdatedAt.After(policy.UpdatedAt) {
		t.Fatalf("updated timestamp = %s, want after %s", updated.UpdatedAt, policy.UpdatedAt)
	}
	got, err := store.GetRecoveryPolicy(ctx, "critical-api", "live")
	if err != nil {
		t.Fatalf("GetRecoveryPolicy: %v", err)
	}
	if !got.OptOut || got.Enabled || got.RetryBudget != 3 {
		t.Fatalf("durable policy = %#v, want updated opt-out policy", got)
	}
}

func TestPressureEpochAndRecoveryDecisionAreRestartSafeAndIdempotent(t *testing.T) {
	ctx := context.Background()
	clock := newFixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clock)
	epoch, err := store.CreatePressureEpoch(ctx, PressureEpoch{EpochID: "epoch-1", Source: "system-monitor"})
	if err != nil {
		t.Fatalf("CreatePressureEpoch: %v", err)
	}
	if epoch.Status != PressureEpochDetected {
		t.Fatalf("epoch status = %q, want detected", epoch.Status)
	}
	clock.Advance(30 * time.Second)
	cleared := clock.Now()
	epoch.Status = PressureEpochCleared
	epoch.ClearedAt = &cleared
	if _, err := store.UpdatePressureEpoch(ctx, epoch); err != nil {
		t.Fatalf("UpdatePressureEpoch: %v", err)
	}
	first, err := store.RecordRecoveryDecision(ctx, RecoveryDecision{
		EpochID: "epoch-1", Scenario: "critical-api", State: RecoveryDecisionQueued,
		Attempt: 1, IdempotencyKey: "epoch-1/critical-api/live/attempt-1",
	})
	if err != nil {
		t.Fatalf("RecordRecoveryDecision(first): %v", err)
	}
	clock.Advance(time.Minute)
	replay, err := store.RecordRecoveryDecision(ctx, RecoveryDecision{
		EpochID: "epoch-1", Scenario: "critical-api", State: RecoveryDecisionRestored,
		Attempt: 1, IdempotencyKey: "epoch-1/critical-api/live/attempt-1",
	})
	if err != nil {
		t.Fatalf("RecordRecoveryDecision(replay): %v", err)
	}
	if replay.DecisionID != first.DecisionID || replay.State != RecoveryDecisionQueued {
		t.Fatalf("replayed decision = %#v, want original %#v", replay, first)
	}
	decisions, err := store.ListRecoveryDecisions(ctx, RecoveryDecisionFilter{EpochID: "epoch-1"})
	if err != nil {
		t.Fatalf("ListRecoveryDecisions: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("decisions = %#v, want one idempotent decision", decisions)
	}
}

func TestRecoveryPolicyRejectsUnsafeValues(t *testing.T) {
	store := newTestStore(t, newFixedClock(time.Now()))
	if _, err := store.UpsertRecoveryPolicy(context.Background(), RecoveryPolicy{Scenario: "api", DependencyTier: -1}); err == nil {
		t.Fatal("UpsertRecoveryPolicy accepted negative dependency tier")
	}
	if _, err := store.UpsertRecoveryPolicy(context.Background(), RecoveryPolicy{Scenario: "api", RetryBudget: -1}); err == nil {
		t.Fatal("UpsertRecoveryPolicy accepted negative retry budget")
	}
}
