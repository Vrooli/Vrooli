package scenarioruntime

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func TestDemandAcquisitionAndStopClaimHaveOneWinner(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path})
	})
	for i := 0; i < 20; i++ {
		scenario := fmt.Sprintf("concurrent-%d", i)
		_, err := store.CreateInstance(ctx, Instance{InstanceID: scenario, Scenario: scenario, Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand})
		if err != nil {
			t.Fatal(err)
		}
		ready := make(chan struct{})
		acquired := make(chan error, 1)
		claimed := make(chan error, 1)
		go func() {
			<-ready
			_, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: scenario, Scenario: scenario, ConsumerID: "concurrent", Kind: DemandLeaseProgram}, time.Minute)
			acquired <- err
		}()
		go func() { <-ready; _, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "test"); claimed <- err }()
		close(ready)
		acquireErr, claimErr := <-acquired, <-claimed
		if claimErr != nil {
			t.Fatal(claimErr)
		}
		instance, err := store.GetInstance(ctx, scenario)
		if err != nil {
			t.Fatal(err)
		}
		if acquireErr == nil {
			if instance.Status != StatusRunning {
				t.Fatalf("acquired hold on stopping instance: %+v", instance)
			}
		} else if !errors.Is(acquireErr, ErrDemandLeaseConflict) || instance.Status != StatusStopping {
			t.Fatalf("unexpected arbitration: status=%s acquire=%v", instance.Status, acquireErr)
		}
	}
}

func TestDemandRejectsStoppingVariantUntilRestored(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path}) })
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "stopping", Scenario: "target", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand})
	if err != nil {
		t.Fatal(err)
	}
	lease := DemandLease{LeaseID: "live", Scenario: "target", ConsumerID: "worker", Kind: DemandLeaseProgram}
	if _, err := store.AcquireDemandLease(ctx, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, StatusStopping, "stop"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RenewDemandLease(ctx, lease.LeaseID, time.Minute); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("renew during stop: %v", err)
	}
	if _, err := store.AcquireDemandLease(ctx, lease, time.Minute); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("acquire during stop: %v", err)
	}
	shadow := lease
	shadow.LeaseID, shadow.Variant = "shadow", "shadow"
	if _, err := store.AcquireDemandLease(ctx, shadow, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RestoreDemandStopCandidate(ctx, instance.InstanceID, instance.Generation, instance.Phase); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RenewDemandLease(ctx, lease.LeaseID, time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestDemandLeaseAcquireRenewAndDemandQuery(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	lease, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "lease-1", Scenario: "ai-gateway", ConsumerID: "program-1", Kind: DemandLeaseProgram}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Variant != DefaultVariant || !lease.ExpiresAt.Equal(clk.Now().Add(time.Minute)) {
		t.Fatalf("lease = %+v", lease)
	}
	has, err := store.HasActiveDemand(ctx, "ai-gateway", "")
	if err != nil || !has {
		t.Fatalf("HasActiveDemand = %v, %v", has, err)
	}
	clk.Advance(20 * time.Second)
	renewed, err := store.RenewDemandLease(ctx, "lease-1", 2*time.Minute)
	if err != nil || !renewed.ExpiresAt.Equal(clk.Now().Add(2*time.Minute)) {
		t.Fatalf("renewed = %+v, %v", renewed, err)
	}
	if renewed.CreatedAt != lease.CreatedAt {
		t.Fatal("renew changed lease identity timestamp")
	}
	reacquired, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "lease-1", Scenario: "ai-gateway", ConsumerID: "program-1", Kind: DemandLeaseProgram}, time.Minute)
	if err != nil || reacquired.LeaseID != lease.LeaseID {
		t.Fatalf("reacquire = %+v, %v", reacquired, err)
	}
}

func TestDemandLeaseExpiryReleaseAndConflict(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "unknown-kind", Scenario: "visited-tracker", ConsumerID: "job-1", Kind: "guess"}, time.Minute); err == nil {
		t.Fatal("accepted unsupported demand lease kind")
	}
	if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "too-long", Scenario: "visited-tracker", ConsumerID: "job-1", Kind: DemandLeaseJob}, MaxDemandLeaseTTL+time.Second); err == nil {
		t.Fatal("accepted unbounded demand lease ttl")
	}
	if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "lease-1", Scenario: "visited-tracker", ConsumerID: "job-1", Kind: DemandLeaseJob}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "lease-1", Scenario: "other", ConsumerID: "job-1", Kind: DemandLeaseJob}, time.Minute); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("identity conflict = %v", err)
	}
	clk.Advance(time.Minute + time.Second)
	expired, err := store.ExpireDemandLeases(ctx, clk.Now())
	if err != nil || len(expired) != 1 || expired[0].Status != DemandLeaseExpired {
		t.Fatalf("expired = %+v, %v", expired, err)
	}
	if _, err := store.RenewDemandLease(ctx, "lease-1", time.Minute); !errors.Is(err, ErrDemandLeaseExpired) {
		t.Fatalf("renew expired = %v", err)
	}
	if _, err := store.ReleaseDemandLease(ctx, "lease-1", "done"); err != nil {
		t.Fatalf("release expired should be idempotent read: %v", err)
	}
	if has, err := store.HasActiveDemand(ctx, "visited-tracker", "live"); err != nil || has {
		t.Fatalf("expired demand remains active: %v, %v", has, err)
	}
}

func TestClaimDemandStopCandidatesIsPolicyAndVariantScoped(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	managed, err := store.CreateInstance(ctx, Instance{InstanceID: "demand-live", Scenario: "ai-gateway", Variant: "live", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand})
	if err != nil {
		t.Fatal(err)
	}
	protected, err := store.CreateInstance(ctx, Instance{InstanceID: "demand-shadow", Scenario: "ai-gateway", Variant: "shadow", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand})
	if err != nil {
		t.Fatal(err)
	}
	manual, err := store.CreateInstance(ctx, Instance{InstanceID: "manual", Scenario: "visited-tracker", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyManaged})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "shadow-hold", Scenario: protected.Scenario, Variant: protected.Variant, ConsumerID: "program", Kind: DemandLeaseProgram}, time.Minute); err != nil {
		t.Fatal(err)
	}
	candidates, err := store.ClaimDemandStopCandidates(ctx, clk.Now(), "expired")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].InstanceID != managed.InstanceID || candidates[0].Status != StatusStopping {
		t.Fatalf("candidates = %+v", candidates)
	}
	restored, err := store.RestoreDemandStopCandidate(ctx, managed.InstanceID, managed.Generation, managed.Phase)
	if err != nil {
		t.Fatalf("RestoreDemandStopCandidate: %v", err)
	}
	if restored.Status != StatusRunning || restored.StopReason != "" {
		t.Fatalf("restored = %+v", restored)
	}
	for _, id := range []string{managed.InstanceID, protected.InstanceID, manual.InstanceID} {
		instance, getErr := store.GetInstance(ctx, id)
		if getErr != nil {
			t.Fatal(getErr)
		}
		switch id {
		case managed.InstanceID:
			if instance.Status != StatusRunning {
				t.Fatalf("managed status = %q", instance.Status)
			}
		case protected.InstanceID, manual.InstanceID:
			if instance.Status != StatusRunning {
				t.Fatalf("protected instance %s status = %q", id, instance.Status)
			}
		}
	}
}

func TestExplicitStartRetentionWinsOverDemandButNotStopping(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path}) })
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "operator", Scenario: "target", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand})
	if err != nil {
		t.Fatal(err)
	}
	retained, err := store.RetainExplicitInstance(ctx, instance.InstanceID, instance.Generation)
	if err != nil || retained.SupervisionPolicy != SupervisionPolicyManaged {
		t.Fatalf("retained: %+v %v", retained, err)
	}
	candidates, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "idle")
	if err != nil || len(candidates) != 0 {
		t.Fatalf("explicit start reaped: %+v %v", candidates, err)
	}
	if _, err := store.RetainExplicitInstance(ctx, instance.InstanceID, instance.Generation+1); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("stale promotion: %v", err)
	}
	if _, err := store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, StatusStopping, "stop"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RetainExplicitInstance(ctx, instance.InstanceID, instance.Generation); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("revived stopping instance: %v", err)
	}
}

func TestTransientReleaseKeepsOnlyABoundedSlidingIdleWindow(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
	if _, err := store.CreateInstance(ctx, Instance{InstanceID: "warm", Scenario: "target", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand}); err != nil {
		t.Fatal(err)
	}
	acquire := func(id string) {
		t.Helper()
		if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: id, Scenario: "target", ConsumerID: "worker", Kind: DemandLeaseProgram}, time.Minute); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReleaseDemandLease(ctx, id, "completed"); err != nil {
			t.Fatal(err)
		}
	}
	acquire("first")
	if active, err := store.HasActiveDemand(ctx, "target", ""); err != nil || active {
		t.Fatalf("release retained caller ownership: %v %v", active, err)
	}
	clk.Advance(9 * time.Minute)
	acquire("second")
	clk.Advance(9 * time.Minute)
	if _, err := store.ReleaseDemandLease(ctx, "second", "retry"); err != nil {
		t.Fatal(err)
	}
	candidates, err := store.ClaimDemandStopCandidates(ctx, clk.Now(), "idle")
	if err != nil || len(candidates) != 0 {
		t.Fatalf("warm instance reaped: %+v %v", candidates, err)
	}
	clk.Advance(time.Minute)
	candidates, err = store.ClaimDemandStopCandidates(ctx, clk.Now(), "idle")
	if err != nil || len(candidates) != 1 {
		t.Fatalf("idle window accumulated credit or replay extended it: %+v %v", candidates, err)
	}
}

func TestRuntimeVersion10UpgradePreservesDemandAndAddsIdleWindows(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path}) })
	lease := DemandLease{LeaseID: "migration", Scenario: "target", ConsumerID: "worker", Kind: DemandLeaseProgram}
	if _, err := store.AcquireDemandLease(ctx, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "DROP TABLE runtime_demand_idle_windows"); err != nil {
		t.Fatal(err)
	}
	// Exercise the real version-stamped upgrade, then prove the additive step is replay-safe.
	if _, err := store.db.ExecContext(ctx, "PRAGMA user_version = 10"); err != nil {
		t.Fatal(err)
	}
	if err := store.ensureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := schemaMigrations[10](ctx, store.db); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseDemandLease(ctx, lease.LeaseID, "migrated"); err != nil {
		t.Fatal(err)
	}
	var rows int
	if err := store.db.QueryRowContext(ctx, "SELECT count(*) FROM runtime_demand_idle_windows").Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("idle migration: %d %v", rows, err)
	}
}
