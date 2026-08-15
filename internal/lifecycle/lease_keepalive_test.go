package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func newKeepAliveSession(t *testing.T, ttl time.Duration) (*runtimeRegistrySession, *scenarioruntime.SQLiteStore) {
	t.Helper()
	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ownerPID := 4242
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		Scenario:   "web-console",
		Status:     scenarioruntime.StatusStarting,
		Phase:      "setup",
		OwnerPID:   &ownerPID,
		HostBootID: "boot-1",
	}, ttl)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	return &runtimeRegistrySession{
		enabled:  true,
		store:    store,
		instance: instance,
		claims:   map[string]scenarioruntime.PortClaim{},
	}, store
}

// A cold setup phase runs far longer than the lease TTL. The lease must be
// renewed while the phase runs, otherwise it is already past due the moment the
// phase ends and the post-phase heartbeat fails.
func TestKeepLeaseAliveRenewsAcrossALongPhase(t *testing.T) {
	ctx := context.Background()
	session, store := newKeepAliveSession(t, 60*time.Millisecond)
	originalDeadline := *session.instance.HeartbeatDeadlineAt

	restore := leaseHeartbeatInterval
	leaseHeartbeatInterval = 5 * time.Millisecond
	t.Cleanup(func() { leaseHeartbeatInterval = restore })

	phaseRan := false
	err := session.keepLeaseAlive(ctx, func(err error) {
		t.Errorf("unexpected renewal failure: %v", err)
	}, func() error {
		// Outlive the TTL, the way a real UI build outlives the 30s default.
		time.Sleep(150 * time.Millisecond)
		phaseRan = true
		return nil
	})
	if err != nil {
		t.Fatalf("keepLeaseAlive() error = %v", err)
	}
	if !phaseRan {
		t.Fatal("phase body did not run")
	}

	after, err := store.GetInstance(ctx, session.instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if after.HeartbeatDeadlineAt == nil {
		t.Fatal("HeartbeatDeadlineAt = nil, want a renewed deadline")
	}
	if !after.HeartbeatDeadlineAt.After(originalDeadline) {
		t.Fatalf("deadline = %s, want later than the creation deadline %s", after.HeartbeatDeadlineAt, originalDeadline)
	}
	if after.Status != scenarioruntime.StatusStarting {
		t.Fatalf("status = %q, want %q", after.Status, scenarioruntime.StatusStarting)
	}

	// The post-phase heartbeat that executeStart performs must still succeed.
	if err := session.heartbeat(ctx); err != nil {
		t.Fatalf("post-phase heartbeat error = %v, want the start to survive its own setup", err)
	}
}

// The pump must be joined before keepLeaseAlive returns, so the caller regains
// exclusive access to the session; a surviving goroutine would race every later
// session write.
func TestKeepLeaseAliveStopsPumpBeforeReturning(t *testing.T) {
	ctx := context.Background()
	session, store := newKeepAliveSession(t, time.Minute)

	restore := leaseHeartbeatInterval
	leaseHeartbeatInterval = time.Millisecond
	t.Cleanup(func() { leaseHeartbeatInterval = restore })

	if err := session.keepLeaseAlive(ctx, nil, func() error {
		time.Sleep(20 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatalf("keepLeaseAlive() error = %v", err)
	}

	settled, err := store.GetInstance(ctx, session.instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	// If the pump were still running it would keep writing past this point.
	time.Sleep(20 * time.Millisecond)
	after, err := store.GetInstance(ctx, session.instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if !after.UpdatedAt.Equal(settled.UpdatedAt) {
		t.Fatalf("instance kept changing after keepLeaseAlive returned (%s -> %s); pump outlived the phase", settled.UpdatedAt, after.UpdatedAt)
	}
}

func TestKeepLeaseAlivePropagatesPhaseError(t *testing.T) {
	session, _ := newKeepAliveSession(t, time.Minute)
	want := errors.New("build failed")
	if err := session.keepLeaseAlive(context.Background(), nil, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("keepLeaseAlive() error = %v, want %v", err, want)
	}
}

// A disabled session (registry off) must still run the phase.
func TestKeepLeaseAliveRunsPhaseWhenRegistryDisabled(t *testing.T) {
	session := disabledRuntimeRegistrySession()
	ran := false
	if err := session.keepLeaseAlive(context.Background(), nil, func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatalf("keepLeaseAlive() error = %v", err)
	}
	if !ran {
		t.Fatal("phase body did not run for a disabled session")
	}
}
