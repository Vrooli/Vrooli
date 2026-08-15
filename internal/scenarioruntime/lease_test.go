package scenarioruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSQLiteStoreLeaseHeartbeatAndStop(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateLease(ctx, Instance{
		InstanceID: "inst-alpha",
		Scenario:   "alpha",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	if created.LastHeartbeatAt == nil || !created.LastHeartbeatAt.Equal(clk.Now()) {
		t.Fatalf("created.LastHeartbeatAt = %#v, want %s", created.LastHeartbeatAt, clk.Now())
	}
	if created.HeartbeatDeadlineAt == nil || !created.HeartbeatDeadlineAt.Equal(clk.Now().Add(time.Minute)) {
		t.Fatalf("created.HeartbeatDeadlineAt = %#v, want %s", created.HeartbeatDeadlineAt, clk.Now().Add(time.Minute))
	}

	clk.Advance(10 * time.Second)
	heartbeat, err := store.HeartbeatLease(ctx, created.InstanceID, created.Generation, 2*time.Minute)
	if err != nil {
		t.Fatalf("HeartbeatLease() error = %v", err)
	}
	if heartbeat.Status != StatusStarting {
		t.Fatalf("heartbeat.Status = %q, want %q", heartbeat.Status, StatusStarting)
	}
	if heartbeat.LastHeartbeatAt == nil || !heartbeat.LastHeartbeatAt.Equal(clk.Now()) {
		t.Fatalf("heartbeat.LastHeartbeatAt = %#v, want %s", heartbeat.LastHeartbeatAt, clk.Now())
	}
	if heartbeat.HeartbeatDeadlineAt == nil || !heartbeat.HeartbeatDeadlineAt.Equal(clk.Now().Add(2*time.Minute)) {
		t.Fatalf("heartbeat.HeartbeatDeadlineAt = %#v, want %s", heartbeat.HeartbeatDeadlineAt, clk.Now().Add(2*time.Minute))
	}

	clk.Advance(time.Second)
	stopped, err := store.StopLease(ctx, created.InstanceID, created.Generation, "operator stop")
	if err != nil {
		t.Fatalf("StopLease() error = %v", err)
	}
	if stopped.Status != StatusStopped {
		t.Fatalf("stopped.Status = %q, want %q", stopped.Status, StatusStopped)
	}
	if stopped.StoppedAt == nil || !stopped.StoppedAt.Equal(clk.Now()) {
		t.Fatalf("stopped.StoppedAt = %#v, want %s", stopped.StoppedAt, clk.Now())
	}
	if stopped.StopReason != "operator stop" {
		t.Fatalf("stopped.StopReason = %q, want operator stop", stopped.StopReason)
	}
}

func TestSQLiteStoreFreshLeaseActiveWithUnknownHealth(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	listed, err := store.ListInstances(ctx, InstanceFilter{Statuses: []string{StatusStarting, StatusRunning}})
	if err != nil {
		t.Fatalf("ListInstances() error = %v", err)
	}
	if len(listed) != 1 || listed[0].InstanceID != created.InstanceID {
		t.Fatalf("active leases = %#v, want fresh lease", listed)
	}
	if _, err := store.GetHealthSnapshot(ctx, created.InstanceID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetHealthSnapshot() error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStoreExpireStaleLeasesDoesNotInspectPorts(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	instance, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	expiresAt := clk.Now().Add(10 * time.Minute)
	if _, err := store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "ALPHA_API_PORT",
		Port:       15080,
		Status:     ClaimStatusBound,
		ExpiresAt:  &expiresAt,
	}); err != nil {
		t.Fatalf("AcquirePortClaim() error = %v", err)
	}

	expired, err := store.ExpireStaleLeases(ctx, clk.Now().Add(time.Minute+time.Nanosecond))
	if err != nil {
		t.Fatalf("ExpireStaleLeases() error = %v", err)
	}
	if len(expired) != 1 || expired[0].InstanceID != instance.InstanceID {
		t.Fatalf("expired = %#v, want inst-alpha", expired)
	}
	if expired[0].Status != StatusExpired {
		t.Fatalf("expired[0].Status = %q, want %q", expired[0].Status, StatusExpired)
	}

	claims, err := store.ListPortClaims(ctx, PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].Status != ClaimStatusBound {
		t.Fatalf("claims = %#v, want bound claim preserved as stale diagnostic", claims)
	}
}

func TestSQLiteStoreExpireStaleStartingLeasesLeavesRunningLeasesForSupervisorMigration(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	starting, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease(starting) error = %v", err)
	}
	running, err := store.CreateLease(ctx, Instance{InstanceID: "inst-beta", Scenario: "beta", Status: StatusRunning}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease(running) error = %v", err)
	}

	expired, err := store.ExpireStaleStartingLeases(ctx, clk.Now().Add(time.Minute+time.Nanosecond), StartingLeaseGuard{})
	if err != nil {
		t.Fatalf("ExpireStaleStartingLeases() error = %v", err)
	}
	if len(expired) != 1 || expired[0].InstanceID != starting.InstanceID {
		t.Fatalf("expired = %#v, want only starting lease", expired)
	}
	afterRunning, err := store.GetInstance(ctx, running.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance(running) error = %v", err)
	}
	if afterRunning.Status != StatusRunning {
		t.Fatalf("running status = %q, want %q", afterRunning.Status, StatusRunning)
	}
}

// A cold setup phase legitimately outruns the heartbeat TTL. The sweep must
// leave that lease alone, because the owner is alive and still building: reaping
// it makes the owner's next lease write fail and rolls back a healthy start.
func TestSQLiteStoreExpireStaleStartingLeasesSparesLiveOwner(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	livePID := 4242
	starting, err := store.CreateLease(ctx, Instance{
		InstanceID: "inst-slow-setup",
		Scenario:   "web-console",
		Phase:      "setup",
		OwnerPID:   &livePID,
		HostBootID: "boot-1",
	}, 30*time.Second)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}

	guard := StartingLeaseGuard{
		CurrentBootID: "boot-1",
		PIDRunning:    func(pid int) bool { return pid == livePID },
	}
	// Far past the deadline — a 92s UI build against a 30s TTL.
	expired, err := store.ExpireStaleStartingLeases(ctx, clk.Now().Add(92*time.Second), guard)
	if err != nil {
		t.Fatalf("ExpireStaleStartingLeases() error = %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expired = %#v, want live starter left untouched", expired)
	}

	after, err := store.GetInstance(ctx, starting.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if after.Status != StatusStarting {
		t.Fatalf("status = %q, want %q so the owner can still heartbeat", after.Status, StatusStarting)
	}

	// The owner must still be able to renew after the sweep ran.
	if _, err := store.HeartbeatLease(ctx, after.InstanceID, after.Generation, 30*time.Second); err != nil {
		t.Fatalf("HeartbeatLease() after sweep error = %v, want the live start to keep its lease", err)
	}
}

func TestSQLiteStoreExpireStaleStartingLeasesReapsAbandonedOwners(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))

	deadPID := 999001
	otherBootPID := 999002
	cases := []struct {
		name        string
		instance    Instance
		wantTrigger string
	}{
		{
			name:        "owner pid dead",
			instance:    Instance{InstanceID: "inst-dead", Scenario: "alpha", OwnerPID: &deadPID, HostBootID: "boot-1"},
			wantTrigger: "owner_pid_dead",
		},
		{
			name:        "previous host boot",
			instance:    Instance{InstanceID: "inst-reboot", Scenario: "beta", OwnerPID: &otherBootPID, HostBootID: "boot-0"},
			wantTrigger: "boot_id_mismatch",
		},
		{
			name:        "owner pid missing",
			instance:    Instance{InstanceID: "inst-nopid", Scenario: "gamma", HostBootID: "boot-1"},
			wantTrigger: "owner_pid_missing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t, clk)
			if _, err := store.CreateLease(ctx, tc.instance, 30*time.Second); err != nil {
				t.Fatalf("CreateLease() error = %v", err)
			}
			guard := StartingLeaseGuard{
				CurrentBootID: "boot-1",
				PIDRunning:    func(int) bool { return false },
			}
			expired, err := store.ExpireStaleStartingLeases(ctx, clk.Now().Add(time.Minute), guard)
			if err != nil {
				t.Fatalf("ExpireStaleStartingLeases() error = %v", err)
			}
			if len(expired) != 1 {
				t.Fatalf("expired = %#v, want the abandoned lease reaped", expired)
			}
			if expired[0].Status != StatusExpired {
				t.Fatalf("status = %q, want %q", expired[0].Status, StatusExpired)
			}
			if want := staleStartingStopReason(tc.wantTrigger); expired[0].StopReason != want {
				t.Fatalf("stop_reason = %q, want %q", expired[0].StopReason, want)
			}
		})
	}
}

// Without a liveness probe the sweep cannot prove any owner dead, so it must not
// condemn a lease on elapsed time alone.
func TestStaleStartingTriggerZeroGuardProtectsLeasesWithOwners(t *testing.T) {
	pid := 1234
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(-time.Minute)
	instance := Instance{
		Status:              StatusStarting,
		OwnerPID:            &pid,
		HostBootID:          "boot-1",
		HeartbeatDeadlineAt: &deadline,
	}
	if trigger, ok := StaleStartingTrigger(instance, StartingLeaseGuard{}, now); ok {
		t.Fatalf("StaleStartingTrigger() = %q, true; want protected under a zero guard", trigger)
	}
}

func TestStaleStartingTriggerIgnoresLeasesBeforeDeadline(t *testing.T) {
	pid := 1234
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(time.Minute)
	instance := Instance{
		Status:              StatusStarting,
		OwnerPID:            &pid,
		HeartbeatDeadlineAt: &deadline,
	}
	guard := StartingLeaseGuard{PIDRunning: func(int) bool { return false }}
	if trigger, ok := StaleStartingTrigger(instance, guard, now); ok {
		t.Fatalf("StaleStartingTrigger() = %q, true; want no reap before the deadline", trigger)
	}
}

func TestSQLiteStoreHeartbeatLeaseRejectsStaleGeneration(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)))
	instance, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}

	_, err = store.HeartbeatLease(ctx, instance.InstanceID, instance.Generation+1, time.Minute)
	if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("HeartbeatLease(stale generation) error = %v, want ErrStaleGeneration", err)
	}
}
