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
