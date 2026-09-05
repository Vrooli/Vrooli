package scenarioruntime

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func TestSQLiteStoreSupervisorSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

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
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	launcherPID := 4242
	instance, err := store.CreateLease(ctx, Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        StatusRunning,
		OwnerPID:      &launcherPID,
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
	if claimed.OwnerPID != nil {
		t.Fatalf("claimed.OwnerPID = %v, want nil after supervisor claim", *claimed.OwnerPID)
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
	if renewed[0].OwnerPID != nil {
		t.Fatalf("renewed.OwnerPID = %v, want nil after supervised heartbeat", *renewed[0].OwnerPID)
	}

	updated, err := store.UpdateInstanceReconciliation(ctx, instance.InstanceID, instance.Generation, string(ReconcileVerifiedRunning), "current")
	if err != nil {
		t.Fatalf("UpdateInstanceReconciliation() error = %v", err)
	}
	if updated.ReconciliationStatus != string(ReconcileVerifiedRunning) || updated.LastReconciledAt == nil {
		t.Fatalf("updated reconciliation = %#v, want verified/current timestamp", updated)
	}
}

// A claim the supervisor no longer owns is skipped, not fatal. This used to
// return ErrStaleGeneration, which aborted the transaction for the WHOLE batch:
// one scenario stopped mid-tick meant every other scenario's lease went
// unrenewed, and the error propagated out of the supervisor's Run loop and
// exited the process. One instance changing state must never cost the rest of
// the fleet its leases.
func TestSQLiteStoreSupervisorBatchSkipsClaimsItNoLongerOwns(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))})
	})

	stale, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease(alpha) error = %v", err)
	}
	live, err := store.CreateLease(ctx, Instance{
		InstanceID: "inst-beta", Scenario: "beta", Status: StatusRunning, SupervisorID: "sup-alpha",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease(beta) error = %v", err)
	}

	// The stale claim is listed FIRST so a regression that aborts the batch
	// cannot pass by luck of ordering.
	renewed, err := store.HeartbeatSupervisedLeaseBatch(ctx, []SupervisionClaim{
		{InstanceID: stale.InstanceID, Generation: stale.Generation + 1, SupervisorID: "sup-alpha"},
		{InstanceID: live.InstanceID, Generation: live.Generation, SupervisorID: "sup-alpha"},
	}, time.Minute)
	if err != nil {
		t.Fatalf("HeartbeatSupervisedLeaseBatch() error = %v, want nil", err)
	}
	if len(renewed) != 1 || renewed[0].InstanceID != live.InstanceID {
		t.Fatalf("renewed = %+v, want only %s", renewed, live.InstanceID)
	}
}

func TestStaleSupervisorTriggerRequiresPositiveEvidence(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	livePID, deadPID := 111, 222
	guard := StartingLeaseGuard{
		CurrentBootID: "boot-current",
		PIDRunning:    func(pid int) bool { return pid == livePID },
	}
	base := SupervisorSession{
		Status:              SupervisorStatusRunning,
		HostBootID:          "boot-current",
		HeartbeatDeadlineAt: now.Add(-time.Minute),
		PID:                 &deadPID,
	}
	withSession := func(mutate func(*SupervisorSession)) SupervisorSession {
		session := base
		mutate(&session)
		return session
	}

	cases := map[string]struct {
		session     SupervisorSession
		wantTrigger string
		wantStale   bool
	}{
		"dead pid past deadline": {session: base, wantTrigger: "owner_pid_dead", wantStale: true},
		"missing pid past deadline": {
			session:     withSession(func(s *SupervisorSession) { s.PID = nil }),
			wantTrigger: "owner_pid_missing", wantStale: true,
		},
		"previous boot": {
			session:     withSession(func(s *SupervisorSession) { s.HostBootID = "boot-previous" }),
			wantTrigger: "boot_id_mismatch", wantStale: true,
		},
		// A supervisor whose deadline has passed but whose process is alive is
		// slow or briefly wedged, not dead — and it is still the fleet's owner.
		"live pid past deadline": {
			session: withSession(func(s *SupervisorSession) { s.PID = &livePID }),
		},
		"deadline in the future": {
			session: withSession(func(s *SupervisorSession) { s.HeartbeatDeadlineAt = now.Add(time.Minute) }),
		},
		"already terminal": {
			session: withSession(func(s *SupervisorSession) { s.Status = SupervisorStatusFailed }),
		},
		// A zero guard proves nothing, so it must condemn nothing that still
		// carries a PID.
		"unprovable without a guard": {
			session: base,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			activeGuard := guard
			if name == "unprovable without a guard" {
				activeGuard = StartingLeaseGuard{}
			}
			trigger, stale := StaleSupervisorTrigger(tc.session, activeGuard, now)
			if stale != tc.wantStale || trigger != tc.wantTrigger {
				t.Fatalf("StaleSupervisorTrigger() = (%q, %v), want (%q, %v)", trigger, stale, tc.wantTrigger, tc.wantStale)
			}
		})
	}
}

// A SIGKILLed supervisor never runs its graceful shutdown, so without a reaper
// its row claims status='running' forever. This host had accumulated 2,362 such
// rows, every one of them indistinguishable from the live supervisor.
func TestExpireStaleSupervisorSessionsRetiresOnlyProvableCorpses(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	deadPID, livePID := 4242, 4243
	for _, session := range []SupervisorSession{
		{SupervisorID: "sup-dead", HostBootID: "boot-current", HostSessionID: "s", PID: &deadPID},
		{SupervisorID: "sup-live", HostBootID: "boot-current", HostSessionID: "s", PID: &livePID},
	} {
		if _, err := store.CreateSupervisorSession(ctx, session, time.Minute); err != nil {
			t.Fatalf("CreateSupervisorSession(%s) error = %v", session.SupervisorID, err)
		}
	}
	clk.Advance(2 * time.Minute) // both deadlines lapse

	expired, err := store.ExpireStaleSupervisorSessions(ctx, clk.Now(), StartingLeaseGuard{
		CurrentBootID: "boot-current",
		PIDRunning:    func(pid int) bool { return pid == livePID },
	})
	if err != nil {
		t.Fatalf("ExpireStaleSupervisorSessions() error = %v", err)
	}
	if len(expired) != 1 || expired[0].SupervisorID != "sup-dead" {
		t.Fatalf("expired = %+v, want only sup-dead", expired)
	}
	if expired[0].Status != SupervisorStatusFailed {
		t.Fatalf("expired status = %q, want %q", expired[0].Status, SupervisorStatusFailed)
	}

	running, err := store.ListSupervisorSessions(ctx, SupervisorSessionFilter{Statuses: []string{SupervisorStatusRunning}})
	if err != nil {
		t.Fatalf("ListSupervisorSessions() error = %v", err)
	}
	if len(running) != 1 || running[0].SupervisorID != "sup-live" {
		t.Fatalf("still running = %+v, want only sup-live", running)
	}
}

// The handover that keeps lifecycle ownership from outliving the command that
// created it.
func TestAttachLiveSupervisionTransfersOwnershipAndRefreshesLease(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	ownerPID := 99
	instance, err := store.CreateLease(ctx, Instance{
		InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning,
		OwnerKind: OwnerKindLifecycle, OwnerPID: &ownerPID,
	}, DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	supervisorPID := 4242
	if _, err := store.CreateSupervisorSession(ctx, SupervisorSession{
		SupervisorID: "sup-live", HostBootID: "boot", HostSessionID: "s", PID: &supervisorPID,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession() error = %v", err)
	}

	attached, ok, err := store.AttachLiveSupervision(ctx, instance.InstanceID, instance.Generation, DefaultSupervisedLeaseTTL)
	if err != nil || !ok {
		t.Fatalf("AttachLiveSupervision() = (ok=%v, err=%v), want (true, nil)", ok, err)
	}
	if attached.OwnerKind != OwnerKindSupervisor {
		t.Fatalf("owner kind = %q, want %q", attached.OwnerKind, OwnerKindSupervisor)
	}
	// The PID must be cleared: leaving the exiting CLI's PID on a
	// supervisor-owned row is what made the orphan-squat guard condemn it.
	if attached.OwnerPID != nil {
		t.Fatalf("owner pid = %v, want nil", *attached.OwnerPID)
	}
	if attached.SupervisorID != "sup-live" {
		t.Fatalf("supervisor id = %q, want sup-live", attached.SupervisorID)
	}
	wantDeadline := clk.Now().Add(DefaultSupervisedLeaseTTL)
	if attached.HeartbeatDeadlineAt == nil || !attached.HeartbeatDeadlineAt.Equal(wantDeadline) {
		t.Fatalf("deadline = %v, want %v — the handover must start with a full window", attached.HeartbeatDeadlineAt, wantDeadline)
	}
}

// Handing an instance to a dead supervisor would be worse than keeping
// lifecycle ownership: nothing would ever renew it.
func TestAttachLiveSupervisionRefusesWhenNoSessionIsLive(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	ownerPID := 99
	instance, err := store.CreateLease(ctx, Instance{
		InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning,
		OwnerKind: OwnerKindLifecycle, OwnerPID: &ownerPID,
	}, DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	supervisorPID := 4242
	if _, err := store.CreateSupervisorSession(ctx, SupervisorSession{
		SupervisorID: "sup-lapsed", HostBootID: "boot", HostSessionID: "s", PID: &supervisorPID,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession() error = %v", err)
	}
	clk.Advance(2 * time.Minute) // the only session's deadline lapses

	kept, ok, err := store.AttachLiveSupervision(ctx, instance.InstanceID, instance.Generation, DefaultSupervisedLeaseTTL)
	if err != nil {
		t.Fatalf("AttachLiveSupervision() error = %v", err)
	}
	if ok {
		t.Fatal("AttachLiveSupervision() attached to a session whose deadline had lapsed")
	}
	if kept.OwnerKind != OwnerKindLifecycle || kept.OwnerPID == nil {
		t.Fatalf("instance = %+v, want untouched lifecycle ownership", kept)
	}
}
