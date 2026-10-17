package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestWriteTransactionTakesItsLockUpFront pins the exact defect that made
// program bindings lose sources at random.
//
// Every transaction in this store reads before it writes. Under a deferred
// transaction that read takes a WAL snapshot, and a commit from another
// connection in the meantime turns the later write into SQLITE_BUSY_SNAPSHOT
// (extended code 517) — which SQLite deliberately excludes from the busy
// handler, so the busy_timeout in the DSN never applied and the loser failed.
//
// The interleaving below is the production shape: `vrooli scenario demand
// acquire` runs as a separate process per binding, and a program's gather
// fan-out runs up to eight of them at once against one file. Separate sql.DB
// handles reproduce that; goroutines on one store cannot, because
// SetMaxOpenConns(1) serializes them on a single connection.
//
// With _txlock=immediate, writer A holds RESERVED from BEGIN, so writer B waits
// under busy_timeout instead of racing ahead, and A's write always succeeds.
// Without it, B commits first and A fails.
func TestWriteTransactionTakesItsLockUpFront(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.db")

	seed, err := NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatalf("open seed store: %v", err)
	}
	defer seed.Close()
	for _, name := range []string{"reader-writer", "other-writer"} {
		if _, err := seed.CreateInstance(ctx, Instance{InstanceID: name, Scenario: name, Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand}); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	open := func(name string) *sql.DB {
		db, err := sql.Open("sqlite", buildDSN(path))
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		db.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = db.Close() })
		return db
	}
	connA, connB := open("A"), open("B")

	tx, err := connA.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("A begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// A reads first, exactly as AcquireDemandLease does before it writes.
	var seen int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_instances`).Scan(&seen); err != nil {
		t.Fatalf("A read: %v", err)
	}

	// B commits a write while A holds its transaction open.
	committed := make(chan error, 1)
	go func() {
		_, err := connB.ExecContext(ctx, `UPDATE runtime_instances SET status = ? WHERE instance_id = ?`, StatusStopped, "other-writer")
		committed <- err
	}()

	// Give B time to either commit (deferred: it races ahead) or block
	// (immediate: A already holds RESERVED). Either way A must still succeed.
	select {
	case err := <-committed:
		if err != nil {
			t.Fatalf("B write: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// B is correctly waiting on A's reserved lock.
	}

	if _, err := tx.ExecContext(ctx, `UPDATE runtime_instances SET status = ? WHERE instance_id = ?`, StatusStopped, "reader-writer"); err != nil {
		t.Fatalf("A upgraded a read snapshot to a write and lost: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("A commit: %v", err)
	}
	if err := <-committed; err != nil {
		t.Fatalf("B write after A committed: %v", err)
	}
}

// TestImmediateTxLockIsDeclared keeps the pragma from being dropped by a later
// DSN edit. The failure it prevents is silent and intermittent, so the DSN
// itself is worth pinning even though the behavioural test above covers it.
func TestImmediateTxLockIsDeclared(t *testing.T) {
	if dsn := buildDSN("/tmp/example.db"); !strings.Contains(dsn, "_txlock=immediate") {
		t.Fatalf("write DSN must take the reserved lock at BEGIN: %s", dsn)
	}
}

// TestConcurrentConnectionsAcquireDemandLeases exercises the real acquisition
// path across separate connections, the way a program's gather fan-out does.
func TestConcurrentConnectionsAcquireDemandLeases(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime.db")

	const writers = 8

	seed, err := NewSQLiteStore(ctx, Config{DBPath: path})
	if err != nil {
		t.Fatalf("open seed store: %v", err)
	}
	defer seed.Close()
	if _, err := seed.CreateInstance(ctx, Instance{InstanceID: "fanout", Scenario: "fanout", Status: StatusRunning, SupervisionPolicy: SupervisionPolicyDemand}); err != nil {
		t.Fatalf("seed instance: %v", err)
	}

	stores := make([]*SQLiteStore, writers)
	for i := range stores {
		store, err := NewSQLiteStore(ctx, Config{DBPath: path})
		if err != nil {
			t.Fatalf("open store %d: %v", i, err)
		}
		defer store.Close()
		stores[i] = store
	}

	for round := range 5 {
		ready := make(chan struct{})
		errs := make([]error, writers)
		var wg sync.WaitGroup
		for i := range writers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-ready
				_, errs[i] = stores[i].AcquireDemandLease(ctx, DemandLease{
					LeaseID:    fmt.Sprintf("lease-%d-%d", round, i),
					Scenario:   "fanout",
					ConsumerID: "fanout-consumer",
					Kind:       DemandLeaseProgram,
				}, time.Minute)
			}()
		}
		close(ready)
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d writer %d lost a demand lease to contention: %v", round, i, err)
			}
		}
	}
}

// TestBusyWindowStaysWithinCallerPatience pins the other half of a cross-module
// invariant. Writers now queue at BEGIN under busy_timeout instead of failing,
// which only helps if the caller waits at least that long: demand.Client bounds
// one control-plane round trip at ControlPlaneTimeout (15s). Raising the window
// past that would turn a patient queue back into a caller-side timeout.
func TestBusyWindowStaysWithinCallerPatience(t *testing.T) {
	const callerPatience = 15 * time.Second
	dsn := buildDSN("/tmp/example.db")
	const marker = "busy_timeout(10000)"
	if !strings.Contains(dsn, marker) {
		t.Fatalf("write DSN must declare %s so contended writers wait rather than fail: %s", marker, dsn)
	}
	if window := 10 * time.Second; window >= callerPatience {
		t.Fatalf("busy window %s must stay under the caller's %s bound", window, callerPatience)
	}
}
