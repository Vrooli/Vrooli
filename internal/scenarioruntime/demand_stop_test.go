package scenarioruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func demandStopFixture(t *testing.T) (*SQLiteStore, Instance) {
	t.Helper()
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path})
	})
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "stop-fixture", Scenario: "fixture", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcquirePortClaim(ctx, PortClaim{ClaimID: "port", InstanceID: instance.InstanceID, Scenario: instance.Scenario, PortName: "api", Port: 15042, Status: ClaimStatusBound}); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "expired")
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = %v, %v", claimed, err)
	}
	return store, claimed[0]
}

func TestDemandStopResumptionKeepsConsumersOut(t *testing.T) {
	ctx := context.Background()
	store, instance := demandStopFixture(t)
	if err := store.BeginDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RestoreDemandStopCandidate(ctx, instance.InstanceID, instance.Generation, ""); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("restore after termination began = %v", err)
	}
	if _, err := store.AcquireDemandLease(ctx, DemandLease{LeaseID: "new-consumer", Scenario: instance.Scenario, ConsumerID: "new", Kind: DemandLeaseJob}, time.Minute); !errors.Is(err, ErrDemandLeaseConflict) {
		t.Fatalf("admit consumer during partial teardown = %v", err)
	}
	// A newly opened registry sees the same intent after the old owner exits.
	reopened, err := NewSQLiteStore(ctx, Config{DBPath: store.path})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	pending, err := reopened.PendingDemandStops(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %v, %v", pending, err)
	}
	if started, err := reopened.DemandStopStarted(ctx, instance.InstanceID, instance.Generation); err != nil || !started {
		t.Fatalf("resumed boundary = %v, %v", started, err)
	}
	if err := reopened.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		t.Fatal(err)
	}
	if pending, err := reopened.PendingDemandStops(ctx); err != nil || len(pending) != 0 {
		t.Fatalf("completed intent remains = %v, %v", pending, err)
	}
	if _, err := store.DemandStopStarted(ctx, instance.InstanceID, instance.Generation); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale candidate remains authorized = %v", err)
	}
}

func TestDemandStopFinalizationIsAtomicAndGenerationScoped(t *testing.T) {
	ctx := context.Background()
	store, instance := demandStopFixture(t)
	if err := store.BeginDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		t.Fatal(err)
	}
	// Fail the instance update after the port update to prove transaction rollback.
	if _, err := store.db.ExecContext(ctx, `CREATE TRIGGER fail_stop BEFORE UPDATE OF status ON runtime_instances WHEN NEW.status = 'stopped' BEGIN SELECT RAISE(ABORT, 'injected persistence failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation); err == nil {
		t.Fatal("injected failure was ignored")
	}
	claims, err := store.ListPortClaims(ctx, PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil || len(claims) != 1 || claims[0].Status != ClaimStatusBound {
		t.Fatalf("failed finalization released ports = %v, %v", claims, err)
	}
	if _, err := store.db.ExecContext(ctx, `DROP TRIGGER fail_stop`); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation+1); !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("stale generation finalization = %v", err)
	}
	if err := store.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		t.Fatal(err)
	}
	claims, err = store.ListPortClaims(ctx, PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil || len(claims) != 1 || claims[0].Status != ClaimStatusReleased {
		t.Fatalf("successful finalization retained ports = %v, %v", claims, err)
	}
}

func TestDemandStopLockExcludesOtherHandlesAndAliases(t *testing.T) {
	ctx := context.Background()
	store, _ := demandStopFixture(t)
	unlock, err := store.AcquireDemandStopLock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	alias := filepath.Join(t.TempDir(), "alias.db")
	if err := os.Symlink(store.path, alias); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	other, err := NewSQLiteStore(ctx, Config{DBPath: alias})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	blocked, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	if release, err := other.AcquireDemandStopLock(blocked); !errors.Is(err, context.DeadlineExceeded) {
		if release != nil {
			release()
		}
		t.Fatalf("overlapping ownership via alias: %v", err)
	}
	unlock()
	unlock() // Cleanup must be idempotent.
	next, cancelNext := context.WithTimeout(ctx, time.Second)
	defer cancelNext()
	release, err := other.AcquireDemandStopLock(next)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestRuntimeVersion11UpgradePreservesUnownedStops(t *testing.T) {
	ctx := context.Background()
	store, instance := demandStopFixture(t)
	if _, err := store.db.ExecContext(ctx, `DROP TABLE runtime_demand_stops; PRAGMA user_version = 11`); err != nil {
		t.Fatal(err)
	}
	if err := store.ensureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if version, err := readSchemaVersion(ctx, store.db); err != nil || version != SchemaVersion {
		t.Fatalf("version = %d, %v", version, err)
	}
	if pending, err := store.PendingDemandStops(ctx); err != nil || len(pending) != 0 {
		t.Fatalf("upgrade guessed legacy stop ownership = %v, %v", pending, err)
	}
	after, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil || after.Status != StatusStopping {
		t.Fatalf("upgrade changed legacy instance = %v, %v", after, err)
	}
	claims, err := store.ListPortClaims(ctx, PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil || len(claims) != 1 || claims[0].Status != ClaimStatusBound {
		t.Fatalf("upgrade changed legacy claims = %v, %v", claims, err)
	}
}

func TestDemandStopLockOwnerHelper(t *testing.T) {
	path := os.Getenv("VROOLI_DEMAND_LOCK_TEST_DB")
	if path == "" {
		return
	}
	store, err := NewSQLiteStore(context.Background(), Config{DBPath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	unlock, err := store.AcquireDemandStopLock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	fmt.Println("locked")
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestDemandStopLockRecoversAfterOwnerProcessDies(t *testing.T) {
	store, _ := demandStopFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDemandStopLockOwnerHelper$")
	cmd.Env = append(os.Environ(), "VROOLI_DEMAND_LOCK_TEST_DB="+store.path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stdin.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() })
	if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatalf("owner readiness = %q, %v", line, err)
	}
	blocked, cancelBlocked := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancelBlocked()
	if unlock, err := store.AcquireDemandStopLock(blocked); !errors.Is(err, context.DeadlineExceeded) {
		if unlock != nil {
			unlock()
		}
		t.Fatalf("live owner lost exclusivity: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	unlock, err := store.AcquireDemandStopLock(ctx)
	if err != nil {
		t.Fatalf("dead owner stranded lock: %v", err)
	}
	unlock()
}

func TestGenericCleanupCannotBypassDemandStop(t *testing.T) {
	ctx := context.Background()
	store, instance := demandStopFixture(t)
	operations := []struct {
		name string
		run  func() error
	}{
		{"release-one", func() error { _, err := store.ReleasePortClaim(ctx, "port"); return err }},
		{"expire-one", func() error { _, err := store.ExpirePortClaim(ctx, "port"); return err }},
		{"release-all", func() error { _, err := store.ReleaseActivePortClaimsForInstance(ctx, instance.InstanceID); return err }},
		{"stop-lease", func() error {
			_, err := store.StopLease(ctx, instance.InstanceID, instance.Generation, "generic cleanup")
			return err
		}},
		{"stuck-reaper", func() error { return FinalizeStuckInstance(ctx, store, instance, "dead-owner", time.Now()) }},
	}
	for _, started := range []bool{false, true} {
		if started {
			if err := store.BeginDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
				t.Fatal(err)
			}
		}
		for _, op := range operations {
			if err := op.run(); !errors.Is(err, ErrDemandStopOwned) {
				t.Fatalf("%s (signaling=%v) bypassed owner: %v", op.name, started, err)
			}
		}
	}
	claims, err := store.ListPortClaims(ctx, PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil || len(claims) != 1 || claims[0].Status != ClaimStatusBound {
		t.Fatalf("generic cleanup changed claims: %v, %v", claims, err)
	}
	if err := store.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		t.Fatalf("exclusive finalizer was prevented from completing: %v", err)
	}
}
