package capacity

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func leaseAuthorityProbe(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+url.PathEscape(path)+"?_pragma=busy_timeout(0)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestHeartbeatRechecksExpiryAfterSQLiteWriterWait(t *testing.T) {
	for _, ownerKind := range []string{OwnerKindOp, OwnerKindResource} {
		t.Run(ownerKind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			path := filepath.Join(t.TempDir(), "capacity.db")
			store, err := NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			claim, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: ownerKind, OwnerID: "writer-wait", ResourceKind: ResourceKindRAM, AmountBytes: 1, PreferredBytes: 1}, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			probe := leaseAuthorityProbe(t, path)
			writer, err := probe.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Close()
			if _, err := writer.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
				t.Fatal(err)
			}
			defer writer.ExecContext(context.Background(), "ROLLBACK")
			done := make(chan error, 1)
			go func() { _, err := store.HeartbeatClaim(ctx, claim.ClaimID, claim.Generation, time.Minute); done <- err }()
			// The transaction connection is in use while the independent SQLite writer
			// holds write authority. Advancing the clock does not require an entry read.
			for store.db.Stats().InUse == 0 {
				select {
				case <-ctx.Done():
					t.Fatal("heartbeat did not acquire its transaction connection")
				default:
					runtime.Gosched()
				}
			}
			clk.Advance(2 * time.Second)
			if _, err := writer.ExecContext(ctx, "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			var heartbeatErr error
			select {
			case heartbeatErr = <-done:
			case <-ctx.Done():
				t.Fatal("heartbeat did not settle after writer released")
			}
			got, err := store.GetClaim(ctx, claim.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			if ownerKind == OwnerKindOp {
				if !errors.Is(heartbeatErr, ErrStaleGeneration) {
					t.Fatalf("expired operation renewed after SQLite writer wait: %v", heartbeatErr)
				}
				if got.Generation != claim.Generation || !got.HeartbeatDeadlineAt.Equal(*claim.HeartbeatDeadlineAt) || !got.LastHeartbeatAt.Equal(*claim.LastHeartbeatAt) {
					t.Fatalf("rejected heartbeat mutated claim: before=%+v after=%+v", claim, got)
				}
			} else if heartbeatErr != nil || got.Generation != claim.Generation || !got.HeartbeatDeadlineAt.After(clk.Now()) {
				t.Fatalf("resident liveness behavior changed: error=%v claim=%+v", heartbeatErr, got)
			}
		})
	}
}

type leaseWriteAuthorityClock struct {
	clock    *testenv.Clock
	probe    *sql.DB
	armed    atomic.Bool
	observed chan error
}

func (c *leaseWriteAuthorityClock) Now() time.Time {
	if c.armed.CompareAndSwap(true, false) {
		// A competing writer with busy_timeout=0 must be refused at the instant the
		// owner reads renewal time. Deferred BeginTx alone would allow this write.
		_, err := c.probe.ExecContext(context.Background(), "UPDATE capacity_claims SET claim_id=claim_id WHERE 0")
		c.observed <- err
	}
	return c.clock.Now()
}

func TestHeartbeatReadsClockUnderSQLiteWriteAuthority(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "capacity.db")
	clk := &leaseWriteAuthorityClock{clock: testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)), observed: make(chan error, 1)}
	store, err := NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindOp, OwnerID: "write-authority", ResourceKind: ResourceKindRAM, AmountBytes: 1, PreferredBytes: 1}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	clk.probe = leaseAuthorityProbe(t, path)
	clk.armed.Store(true)
	renewed, err := store.HeartbeatClaim(ctx, claim.ClaimID, claim.Generation, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case probeErr := <-clk.observed:
		if probeErr == nil || !isSQLiteLockContention(probeErr) {
			t.Fatalf("renewal time read without write authority: competing writer error=%v", probeErr)
		}
	case <-ctx.Done():
		t.Fatal("heartbeat did not read its clock")
	}
	if renewed.Generation != claim.Generation || renewed.AmountBytes != claim.AmountBytes || renewed.OwnerID != claim.OwnerID || renewed.Status != claim.Status {
		t.Fatalf("write-authority barrier mutated identity/capacity: before=%+v after=%+v", claim, renewed)
	}
}

func TestSQLiteOpenedPathIdentitySurvivesWorkingDirectoryChange(t *testing.T) {
	ctx := context.Background()
	first := t.TempDir()
	second := t.TempDir()
	t.Chdir(first)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join("nested", "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	claim, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindOp, OwnerID: "canonical-path", ResourceKind: ResourceKindRAM, AmountBytes: 1, PreferredBytes: 1}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.EvalSymlinks(filepath.Join(first, "nested", "capacity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(second)
	if store.DBPath() != expected {
		t.Fatalf("opened path reinterpreted in later cwd: got=%q want=%q", store.DBPath(), expected)
	}
	// Force a later connection to prove the DSN, not just reported metadata,
	// retains the file that the owner originally opened.
	store.db.SetMaxIdleConns(0)
	if got, err := store.GetClaim(ctx, claim.ClaimID); err != nil || got.ClaimID != claim.ClaimID {
		t.Fatalf("reopened wrong relative ledger: claim=%+v error=%v", got, err)
	}
	alias := filepath.Join(second, "alias.db")
	if err := os.Symlink(expected, alias); err != nil {
		t.Fatal(err)
	}
	aliased, err := NewSQLiteStore(ctx, Config{DBPath: alias})
	if err != nil {
		t.Fatal(err)
	}
	defer aliased.Close()
	if aliased.DBPath() != expected {
		t.Fatalf("opened file alias has different identity: got=%q want=%q", aliased.DBPath(), expected)
	}
	memory, err := NewSQLiteStore(ctx, Config{DBPath: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()
	if memory.DBPath() != ":memory:" {
		t.Fatalf("memory identity became a disk lock: %q", memory.DBPath())
	}
}
