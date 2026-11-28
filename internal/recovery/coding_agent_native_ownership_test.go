package recovery

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeOwnershipSurvivesReadOnlyBrokerReplacement(t *testing.T) {
	broker, _ := nativeRecoveryFixture(t)
	reopened, err := NewCodingAgentBroker(broker.path, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := reopened.NativeOwnershipSnapshot(t.Context())
	if err != nil || len(snapshot) != 1 || !snapshot[0].NativePending {
		t.Fatalf("observer replacement lost durable native ownership: %+v %v", snapshot, err)
	}
}

func TestNativeOwnershipReadOnlyRetainedLedgerProbe(t *testing.T) {
	path := os.Getenv("VROOLI_CODING_OWNERSHIP_PROBE_LEDGER")
	if path == "" {
		t.Skip("requires explicit read-only retained ledger probe")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broker, err := NewCodingAgentBroker(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := broker.NativeOwnershipSnapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	records, err := broker.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	unknown, pending := 0, 0
	for _, record := range records {
		if record.Outcome == "unconfirmed" {
			unknown++
		}
	}
	for _, item := range snapshot {
		if item.NativePending {
			pending++
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("retained ledger changed during the read-only probe; preserve concurrent attribution")
	}
	t.Logf("retained ledger unchanged: records=%d unconfirmed=%d native_sidecars=%d native_pending=%d sha256=%x", len(records), unknown, len(snapshot), pending, sha256.Sum256(after))
}

func TestNativeOwnershipRequiresExactExitRatherThanReceiptCompletion(t *testing.T) {
	broker, execution := nativeRecoveryFixture(t)
	broker.records[0].Outcome = "succeeded"
	if err := broker.persistLocked(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := broker.NativeOwnershipSnapshot(t.Context())
	if err != nil || len(snapshot) != 1 || !snapshot[0].NativePending {
		t.Fatalf("completed receipt hid a native execution: %+v %v", snapshot, err)
	}
	execution.ExitProven = true
	execution.Fence = "revoked"
	if err := broker.writeExecution(execution); err != nil {
		t.Fatal(err)
	}
	snapshot, err = broker.NativeOwnershipSnapshot(t.Context())
	if err != nil || len(snapshot) != 1 || snapshot[0].NativePending {
		t.Fatalf("exact native exit did not release ownership: %+v %v", snapshot, err)
	}
}

func TestLegacyUnknownReceiptsDoNotInventNativeOwnership(t *testing.T) {
	broker, _ := nativeRecoveryFixture(t)
	broker.records = []CodingAgentRecord{{ID: "00000000-0000-4000-8000-000000000003", Scenario: "agent-manager", Requester: "legacy", Outcome: "unconfirmed", CreatedAt: time.Now().Add(-24 * time.Hour)}}
	if err := broker.persistLocked(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := broker.NativeOwnershipSnapshot(t.Context())
	if err != nil || len(snapshot) != 0 {
		t.Fatalf("historical receipt became a physical native writer: %+v %v", snapshot, err)
	}
	records, err := broker.Snapshot()
	if err != nil || len(records) != 1 || records[0].Outcome != "unconfirmed" {
		t.Fatal("read-only ownership projection changed the retained budget receipt")
	}
}

func TestNativeOwnershipRejectsCrossScenarioSidecarBinding(t *testing.T) {
	broker, execution := nativeRecoveryFixture(t)
	execution.Request.Scenario = "agent-manager"
	execution.Request.Claim = filepath.Join("scenarios", "agent-manager")
	if err := broker.writeExecution(execution); err != nil {
		t.Fatal(err)
	}
	if _, err := broker.NativeOwnershipSnapshot(t.Context()); err == nil {
		t.Fatal("foreign scenario sidecar fabricated an execution owner")
	}
}

func TestNativeOwnershipObserverHonorsCanceledLockWait(t *testing.T) {
	broker, _ := nativeRecoveryFixture(t)
	unlock, err := broker.lockLedger(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := broker.NativeOwnershipSnapshot(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("observer ignored caller lock deadline: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("optional observer exceeded its caller's lock budget")
	}
}

func TestNativeOwnershipObserverDoesNotWaitForProcessCacheMutex(t *testing.T) {
	broker, _ := nativeRecoveryFixture(t)
	broker.mu.Lock()
	defer broker.mu.Unlock()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := broker.NativeOwnershipSnapshot(ctx); finished <- err }()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("read-only native ownership depended on the mutable process cache: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("read-only native ownership waited on the mutable process cache")
	}
}
