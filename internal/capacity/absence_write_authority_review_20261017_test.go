package capacity

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

type absenceWriteResult struct {
	count int
	err   error
}

func TestAbsenceMutationRejectsProofAgedDuringDatabaseWait(t *testing.T) {
	for _, mutation := range []string{"expiry", "zero-sample"} {
		for _, wait := range []string{"pool", "writer"} {
			t.Run(mutation+"/"+wait, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
				path := filepath.Join(t.TempDir(), "capacity.db")
				store, err := NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk})
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				c, err := store.CreateClaim(ctx, residentClaim("absence-wait"), time.Second)
				if err != nil {
					t.Fatal(err)
				}
				c, err = store.RecordObserved(ctx, c.ClaimID, 2*gib, 3*gib, clk.Now(), nil)
				if err != nil {
					t.Fatal(err)
				}
				clk.Advance(2 * time.Second)
				at := clk.Now()
				proof := AbsenceProof{Generation: c.Generation, ObservedAt: at}
				var held *sql.Conn
				waitsBefore := store.db.Stats().WaitCount
				if wait == "pool" {
					held, err = store.db.Conn(ctx)
				} else {
					held, err = leaseAuthorityProbe(t, path).Conn(ctx)
					if err == nil {
						_, err = held.ExecContext(ctx, "BEGIN IMMEDIATE")
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				defer held.Close()
				if wait == "writer" {
					defer held.ExecContext(context.Background(), "ROLLBACK")
				}
				done := make(chan absenceWriteResult, 1)
				go func() {
					if mutation == "expiry" {
						rows, err := store.ExpireStaleClaims(ctx, at, AbsenceProofs{c.ClaimID: proof})
						done <- absenceWriteResult{len(rows), err}
					} else {
						row, err := store.RecordObserved(ctx, c.ClaimID, 0, 0, at, &proof)
						count := 0
						if IsActiveClaimStatus(row.Status) {
							count = 1
						}
						done <- absenceWriteResult{count, err}
					}
				}()
				for {
					stats := store.db.Stats()
					queued := stats.WaitCount > waitsBefore
					if wait == "writer" {
						queued = stats.InUse > 0
					}
					if queued {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("absence write did not reach held database authority")
					default:
						runtime.Gosched()
					}
				}
				clk.Advance(DefaultHeartbeatTTL + time.Second)
				if wait == "pool" {
					err = held.Close()
				} else {
					_, err = held.ExecContext(ctx, "ROLLBACK")
				}
				if err != nil {
					t.Fatal(err)
				}
				select {
				case result := <-done:
					if result.err != nil || result.count != 0 {
						t.Fatalf("aged absence mutated after %s wait: %+v", wait, result)
					}
				case <-ctx.Done():
					t.Fatal("absence write did not settle after database release")
				}
				got, err := store.GetClaim(ctx, c.ClaimID)
				if err != nil {
					t.Fatal(err)
				}
				if got.Status != StatusGranted || got.Generation != c.Generation || got.ObservedBytes != c.ObservedBytes || !got.ObservedAt.Equal(*c.ObservedAt) {
					t.Fatalf("aged proof altered current resident: before=%+v after=%+v", c, got)
				}
			})
		}
	}
}

func TestAbsenceMutationReadsClockUnderSQLiteWriteAuthority(t *testing.T) {
	for _, mutation := range []string{"expiry", "zero-sample"} {
		t.Run(mutation, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			path := filepath.Join(t.TempDir(), "capacity.db")
			clk := &leaseWriteAuthorityClock{clock: testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)), observed: make(chan error, 1)}
			store, err := NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			c, err := store.CreateClaim(ctx, residentClaim("authority-proof"), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			clk.clock.Advance(2 * time.Second)
			at := clk.clock.Now()
			proof := AbsenceProof{Generation: c.Generation, ObservedAt: at}
			clk.probe = leaseAuthorityProbe(t, path)
			clk.armed.Store(true)
			if mutation == "expiry" {
				rows, err := store.ExpireStaleClaims(ctx, at, AbsenceProofs{c.ClaimID: proof})
				if err != nil || len(rows) != 1 {
					t.Fatalf("fresh proof refused: rows=%+v error=%v", rows, err)
				}
			} else {
				row, err := store.RecordObserved(ctx, c.ClaimID, 0, 0, at, &proof)
				if err != nil || !IsActiveClaimStatus(row.Status) {
					t.Fatalf("fresh zero sample refused: row=%+v error=%v", row, err)
				}
			}
			select {
			case probeErr := <-clk.observed:
				if probeErr == nil || !isSQLiteLockContention(probeErr) {
					t.Fatalf("absence time read without write authority: competing writer error=%v", probeErr)
				}
			case <-ctx.Done():
				t.Fatal("absence proof clock was not checked")
			}
		})
	}
}

type absenceBatchClock struct {
	clock       *testenv.Clock
	advanceOnce atomic.Bool
}

func (c *absenceBatchClock) Now() time.Time {
	at := c.clock.Now()
	if c.advanceOnce.CompareAndSwap(true, false) {
		c.clock.Advance(DefaultHeartbeatTTL + time.Second)
	}
	return at
}

func TestExpiryChecksEachProofAtItsOwnMutationTime(t *testing.T) {
	ctx := context.Background()
	clk := &absenceBatchClock{clock: testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))}
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: clk})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first, err := store.CreateClaim(ctx, residentClaim("first"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	clk.clock.Advance(time.Millisecond)
	second, err := store.CreateClaim(ctx, residentClaim("second"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	clk.clock.Advance(time.Millisecond)
	operation := residentClaim("operation")
	operation.OwnerKind = OwnerKindOp
	op, err := store.CreateClaim(ctx, operation, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	clk.clock.Advance(2 * time.Second)
	at := clk.clock.Now()
	proofs := AbsenceProofs{first.ClaimID: {Generation: first.Generation, ObservedAt: at}, second.ClaimID: {Generation: second.Generation, ObservedAt: at}}
	clk.advanceOnce.Store(true)
	expired, err := store.ExpireStaleClaims(ctx, at, proofs)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 2 {
		t.Fatalf("batch reused one proof-age clock or lost operation TTL: %+v", expired)
	}
	kept, err := store.GetClaim(ctx, second.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Status != StatusGranted {
		t.Fatalf("proof aged during earlier row mutation but expired resident: %+v", kept)
	}
	for _, row := range expired {
		got, err := store.GetClaim(ctx, row.ClaimID)
		if err != nil {
			t.Fatal(err)
		}
		if !row.UpdatedAt.Equal(got.UpdatedAt) {
			t.Fatalf("returned mutation time differs from durable row: returned=%+v durable=%+v", row, got)
		}
		if row.ClaimID == first.ClaimID && !row.UpdatedAt.Equal(at) {
			t.Fatalf("first row timestamp wrong: %+v", row)
		}
		if row.ClaimID == op.ClaimID && !row.UpdatedAt.Equal(clk.clock.Now()) {
			t.Fatalf("operation mutation timestamp wrong: %+v", row)
		}
	}
}
