package scenarioruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSQLiteStoreSupervisorSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	pid := 4242
	session, err := store.CreateSupervisorSession(ctx, SupervisorSession{
		SupervisorID:  "sup-alpha",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
		Version:       "test",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateSupervisorSession() error = %v", err)
	}
	if session.Status != SupervisorStatusRunning {
		t.Fatalf("session.Status = %q, want %q", session.Status, SupervisorStatusRunning)
	}
	if !session.HeartbeatDeadlineAt.Equal(clk.Now().Add(time.Minute)) {
		t.Fatalf("deadline = %s, want %s", session.HeartbeatDeadlineAt, clk.Now().Add(time.Minute))
	}

	clk.Advance(10 * time.Second)
	heartbeat, err := store.HeartbeatSupervisorSession(ctx, session.SupervisorID, 2*time.Minute)
	if err != nil {
		t.Fatalf("HeartbeatSupervisorSession() error = %v", err)
	}
	if !heartbeat.LastHeartbeatAt.Equal(clk.Now()) {
		t.Fatalf("heartbeat.LastHeartbeatAt = %s, want %s", heartbeat.LastHeartbeatAt, clk.Now())
	}
	if !heartbeat.HeartbeatDeadlineAt.Equal(clk.Now().Add(2 * time.Minute)) {
		t.Fatalf("heartbeat deadline = %s, want %s", heartbeat.HeartbeatDeadlineAt, clk.Now().Add(2*time.Minute))
	}

	active, err := store.ListSupervisorSessions(ctx, SupervisorSessionFilter{Statuses: []string{SupervisorStatusRunning}})
	if err != nil {
		t.Fatalf("ListSupervisorSessions() error = %v", err)
	}
	if len(active) != 1 || active[0].SupervisorID != session.SupervisorID {
		t.Fatalf("active sessions = %#v, want sup-alpha", active)
	}

	stopped, err := store.StopSupervisorSession(ctx, session.SupervisorID, SupervisorStatusStopped, "operator stop")
	if err != nil {
		t.Fatalf("StopSupervisorSession() error = %v", err)
	}
	if stopped.Status != SupervisorStatusStopped || stopped.StopReason != "operator stop" {
		t.Fatalf("stopped = %#v, want stopped/operator stop", stopped)
	}
}

func TestSQLiteStoreClaimSupervisionAndHeartbeatBatch(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	instance, err := store.CreateLease(ctx, Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	claim := SupervisionClaim{InstanceID: instance.InstanceID, Generation: instance.Generation, SupervisorID: "sup-alpha"}
	claimed, err := store.ClaimSupervision(ctx, claim)
	if err != nil {
		t.Fatalf("ClaimSupervision() error = %v", err)
	}
	if claimed.SupervisorID != "sup-alpha" || claimed.OwnerKind != OwnerKindSupervisor || claimed.SupervisedAt == nil {
		t.Fatalf("claimed = %#v, want supervisor ownership", claimed)
	}

	clk.Advance(15 * time.Second)
	renewed, err := store.HeartbeatSupervisedLeaseBatch(ctx, []SupervisionClaim{claim}, 90*time.Second)
	if err != nil {
		t.Fatalf("HeartbeatSupervisedLeaseBatch() error = %v", err)
	}
	if len(renewed) != 1 || renewed[0].InstanceID != instance.InstanceID {
		t.Fatalf("renewed = %#v, want inst-alpha", renewed)
	}
	if renewed[0].LastHeartbeatAt == nil || !renewed[0].LastHeartbeatAt.Equal(clk.Now()) {
		t.Fatalf("renewed heartbeat = %#v, want %s", renewed[0].LastHeartbeatAt, clk.Now())
	}
	if renewed[0].HeartbeatDeadlineAt == nil || !renewed[0].HeartbeatDeadlineAt.Equal(clk.Now().Add(90*time.Second)) {
		t.Fatalf("renewed deadline = %#v, want %s", renewed[0].HeartbeatDeadlineAt, clk.Now().Add(90*time.Second))
	}

	updated, err := store.UpdateInstanceReconciliation(ctx, instance.InstanceID, instance.Generation, string(ReconcileVerifiedRunning), "current")
	if err != nil {
		t.Fatalf("UpdateInstanceReconciliation() error = %v", err)
	}
	if updated.ReconciliationStatus != string(ReconcileVerifiedRunning) || updated.LastReconciledAt == nil {
		t.Fatalf("updated reconciliation = %#v, want verified/current timestamp", updated)
	}
}

func TestSQLiteStoreSupervisorBatchRejectsStaleGeneration(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)))

	instance, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	_, err = store.HeartbeatSupervisedLeaseBatch(ctx, []SupervisionClaim{{
		InstanceID:   instance.InstanceID,
		Generation:   instance.Generation + 1,
		SupervisorID: "sup-alpha",
	}}, time.Minute)
	if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("HeartbeatSupervisedLeaseBatch(stale) error = %v, want ErrStaleGeneration", err)
	}
}
