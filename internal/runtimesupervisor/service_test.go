package runtimesupervisor

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFixedClock(t time.Time) *fixedClock {
	return &fixedClock{now: t.UTC()}
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type fakeHostProvider struct {
	snapshot hostsession.Snapshot
}

func (p fakeHostProvider) Current(context.Context, string) (hostsession.Snapshot, error) {
	return p.snapshot, nil
}

func TestModeFromEnvDefaultsToAuto(t *testing.T) {
	t.Setenv(ModeEnv, "")
	if got := ModeFromEnv(); got != ModeAuto {
		t.Fatalf("ModeFromEnv() = %q, want %q", got, ModeAuto)
	}
	t.Setenv(ModeEnv, ModeOff)
	if got := ModeFromEnv(); got != ModeOff {
		t.Fatalf("ModeFromEnv(off) = %q, want %q", got, ModeOff)
	}
}

func TestServiceTickAdoptsAndRenewsRunningInstance(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	clk.Advance(10 * time.Second)
	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Renewed != 1 || report.Unverified != 0 || report.Expired != 0 {
		t.Fatalf("report = %#v, want one renewed", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	after, err := check.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.SupervisorID != "sup-alpha" || after.OwnerKind != scenarioruntime.OwnerKindSupervisor {
		t.Fatalf("after supervision = %#v, want sup-alpha owner", after)
	}
	if after.HeartbeatDeadlineAt == nil || !after.HeartbeatDeadlineAt.Equal(clk.Now().Add(90*time.Second)) {
		t.Fatalf("deadline = %#v, want %s", after.HeartbeatDeadlineAt, clk.Now().Add(90*time.Second))
	}
	if after.ReconciliationStatus != string(scenarioruntime.ReconcileVerifiedRunning) {
		t.Fatalf("reconciliation = %q, want verified", after.ReconciliationStatus)
	}
}

func TestServiceTickPersistsListenerEvidenceForBoundClaims(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       18080,
		Status:     scenarioruntime.ClaimStatusBound,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	listenerPID := 2468
	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PortListener: func(port int) scenarioruntime.ListenerEvidence {
			if port != 18080 {
				t.Fatalf("inspected port = %d, want 18080", port)
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: true, PID: &listenerPID, ProcessLabel: "alpha-api"}
		},
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Renewed != 1 || report.Unverified != 0 || report.Expired != 0 {
		t.Fatalf("report = %#v, want one renewed", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	claims, err := check.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims = %#v, want one", claims)
	}
	claim := claims[0]
	if claim.ListenerStatus != scenarioruntime.ListenerStatusListening {
		t.Fatalf("ListenerStatus = %q, want listening", claim.ListenerStatus)
	}
	if claim.LastListenerCheckAt == nil || !claim.LastListenerCheckAt.Equal(clk.Now()) {
		t.Fatalf("LastListenerCheckAt = %#v, want %s", claim.LastListenerCheckAt, clk.Now())
	}
	if claim.LastListenerSeenAt == nil || !claim.LastListenerSeenAt.Equal(clk.Now()) {
		t.Fatalf("LastListenerSeenAt = %#v, want %s", claim.LastListenerSeenAt, clk.Now())
	}
	if claim.ListenerPID == nil || *claim.ListenerPID != listenerPID || claim.ListenerProcessLabel != "alpha-api" {
		t.Fatalf("listener identity = pid %#v label %q, want %d alpha-api", claim.ListenerPID, claim.ListenerProcessLabel, listenerPID)
	}
}

func TestServiceTickReconcilesLiveStartingInstance(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-starting",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusStarting,
		Phase:         "develop",
		ScopePath:     t.TempDir(),
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-starting-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       18081,
		Status:     scenarioruntime.ClaimStatusReserved,
	}); err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	pid := 3579
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "ref-starting-api",
		InstanceID: instance.InstanceID,
		PID:        &pid,
		Step:       "start-api",
		Status:     "running",
		HostBootID: "boot-current",
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	listenerPID := pid
	ready := true
	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		LeaseTTL:     90 * time.Second,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning: func(got int) bool {
			return got == pid
		},
		PortListener: func(port int) scenarioruntime.ListenerEvidence {
			if port != 18081 {
				t.Fatalf("inspected port = %d, want 18081", port)
			}
			return scenarioruntime.ListenerEvidence{Known: true, Listening: true, PID: &listenerPID, ProcessLabel: "alpha-api"}
		},
		HealthProber: func(_ context.Context, got scenarioruntime.Instance, _ []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot {
			if got.InstanceID != instance.InstanceID {
				t.Fatalf("health instance = %s, want %s", got.InstanceID, instance.InstanceID)
			}
			now := clk.Now()
			return scenarioruntime.HealthSnapshot{
				InstanceID: got.InstanceID,
				Scenario:   got.Scenario,
				Status:     scenarioruntime.HealthStatusHealthy,
				Readiness:  &ready,
				CheckedAt:  &now,
			}
		},
	})
	defer svc.Close()

	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Renewed != 1 || report.Unverified != 0 || report.Expired != 0 {
		t.Fatalf("report = %#v, want reconciled instance renewed", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	after, err := check.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.Status != scenarioruntime.StatusRunning || after.SupervisorID != "sup-alpha" {
		t.Fatalf("after = %#v, want running supervised", after)
	}
	claims, err := check.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusBound {
		t.Fatalf("claims = %#v, want one bound claim", claims)
	}
}

func TestServiceTickExpiresPreviousBootInstance(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID:    "inst-alpha",
		Scenario:      "alpha",
		Status:        scenarioruntime.StatusRunning,
		HostBootID:    "boot-old",
		HostSessionID: "session-old",
		SupervisorID:  "sup-alpha",
	}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{
		DBPath:       dbPath,
		SupervisorID: "sup-alpha",
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.Expired != 1 || report.Renewed != 0 {
		t.Fatalf("report = %#v, want one expired", report)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	after, err := check.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if after.Status != scenarioruntime.StatusExpired {
		t.Fatalf("after.Status = %q, want expired", after.Status)
	}
}

func TestServiceTickRunsHealthProbesWithBoundedConcurrency(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := store.CreateLease(ctx, scenarioruntime.Instance{
			InstanceID:    "",
			Scenario:      "alpha",
			Status:        scenarioruntime.StatusRunning,
			HostBootID:    "boot-current",
			HostSessionID: "session-current",
		}, time.Minute); err != nil {
			t.Fatalf("CreateLease(%d): %v", i, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var mu sync.Mutex
	var inFlight int
	var maxInFlight int
	prober := func(_ context.Context, instance scenarioruntime.Instance, _ []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
		now := clk.Now()
		ready := true
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusHealthy,
			Readiness:  &ready,
			CheckedAt:  &now,
		}
	}

	svc := New(Config{
		DBPath:               dbPath,
		SupervisorID:         "sup-alpha",
		Clock:                clk,
		HostProvider:         fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		HealthInterval:       time.Minute,
		MaxHealthConcurrency: 2,
		HealthProber:         prober,
	})
	defer svc.Close()
	report, err := svc.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if report.HealthProbeCount != 8 {
		t.Fatalf("HealthProbeCount = %d, want 8", report.HealthProbeCount)
	}
	if maxInFlight > 2 {
		t.Fatalf("max concurrent health probes = %d, want <= 2", maxInFlight)
	}

	check, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer check.Close()
	instances, err := check.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: []string{scenarioruntime.StatusRunning}})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	for _, instance := range instances {
		snapshot, err := check.GetHealthSnapshot(ctx, instance.InstanceID)
		if err != nil {
			t.Fatalf("GetHealthSnapshot(%s): %v", instance.InstanceID, err)
		}
		if snapshot.Status != scenarioruntime.HealthStatusHealthy {
			t.Fatalf("snapshot.Status = %q, want healthy", snapshot.Status)
		}
	}
}

func TestServiceStatusReportsDeadWhenRecordedPIDIsMissing(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	pid := 424242
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  "sup-stale-pid",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning: func(got int) bool {
			if got != pid {
				t.Fatalf("PIDRunning(%d), want %d", got, pid)
			}
			return false
		},
	})
	defer svc.Close()

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Status != StatusDead {
		t.Fatalf("report.Status = %q, want %q; report=%#v", report.Status, StatusDead, report)
	}
	if report.StatusReason == "" {
		t.Fatalf("report.StatusReason is empty, want PID remediation context")
	}
}

func TestServiceStatusReportsStaleWhenHeartbeatExpired(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	pid := 5151
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  "sup-expired",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
	}, 10*time.Second); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	clk.Advance(11 * time.Second)

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning:   func(int) bool { return true },
	})
	defer svc.Close()

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Status != StatusStale {
		t.Fatalf("report.Status = %q, want %q; report=%#v", report.Status, StatusStale, report)
	}
	if report.StatusReason == "" {
		t.Fatalf("report.StatusReason is empty, want heartbeat context")
	}
}

func TestServiceStatusReportsRunningOnlyForFreshLiveSession(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	pid := 6161
	if _, err := store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  "sup-live",
		HostBootID:    "boot-current",
		HostSessionID: "session-current",
		PID:           &pid,
	}, time.Minute); err != nil {
		t.Fatalf("CreateSupervisorSession: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc := New(Config{
		DBPath:       dbPath,
		Clock:        clk,
		HostProvider: fakeHostProvider{snapshot: hostsession.Snapshot{BootID: "boot-current", SessionID: "session-current"}},
		PIDRunning:   func(int) bool { return true },
	})
	defer svc.Close()

	report, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if report.Status != scenarioruntime.SupervisorStatusRunning {
		t.Fatalf("report.Status = %q, want running; report=%#v", report.Status, report)
	}
	if report.StatusReason != "" {
		t.Fatalf("report.StatusReason = %q, want empty", report.StatusReason)
	}
}
