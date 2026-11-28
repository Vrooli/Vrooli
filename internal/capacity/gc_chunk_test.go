package capacity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

// A terminal claim that a residency transition still references must survive
// GC. Deleting it violates the foreign key, and the old single DELETE then
// rolled back every other prunable row too, so GC never pruned anything.
func TestGCTerminalClaimsSkipsClaimsWithResidencyHistoryAndStillPrunesTheRest(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	clk := testenv.NewClock(t0)
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	release := func(owner string) CapacityClaim {
		c := sampleClaim()
		c.OwnerID = owner
		created, err := store.CreateClaim(ctx, c, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReleaseClaim(ctx, created.ClaimID); err != nil {
			t.Fatal(err)
		}
		return created
	}
	audited := release("audited")
	if _, err := store.db.ExecContext(ctx, `INSERT INTO capacity_residency_transitions(transition_id,claim_id,claim_generation,revision,state,transition_json,started_at,completed_at) VALUES('t1',?,1,1,'completed','{}',?,?)`, audited.ClaimID, formatTime(t0), formatTime(t0)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO capacity_residency_transition_events(transition_id,claim_id,claim_generation,revision,event_kind,transition_json,recorded_at) VALUES('t1',?,1,1,'completed','{}',?)`, audited.ClaimID, formatTime(t0)); err != nil {
		t.Fatal(err)
	}
	plain := release("plain")

	res, err := store.GCTerminalClaims(ctx, t0.Add(time.Hour))
	if err != nil {
		t.Fatalf("GC failed on a claim with residency history: %v", err)
	}
	if res.Count != 1 {
		t.Fatalf("pruned %d claims, want exactly the unaudited one", res.Count)
	}
	if _, err := store.GetClaim(ctx, plain.ClaimID); err == nil {
		t.Fatal("unaudited terminal claim was not pruned")
	}
	if _, err := store.GetClaim(ctx, audited.ClaimID); err != nil {
		t.Fatalf("claim with residency history must survive GC: %v", err)
	}
}

// GC deletes in bounded transactions so the SQLite writer lock is released
// between batches, and a backlog larger than one batch is still fully pruned
// by one call within its budget.
func TestGCTerminalClaimsPrunesBacklogLargerThanOneChunk(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	clk := testenv.NewClock(t0)
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	total := gcTerminalChunkRows*2 + 37
	for i := 0; i < total; i++ {
		c := sampleClaim()
		c.OwnerID = fmt.Sprintf("owner-%d", i)
		created, err := store.CreateClaim(ctx, c, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.ReleaseClaim(ctx, created.ClaimID); err != nil {
			t.Fatal(err)
		}
	}
	res, err := store.GCTerminalClaims(ctx, t0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != total {
		t.Fatalf("pruned %d of %d terminal claims", res.Count, total)
	}
	var left int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM capacity_claims`).Scan(&left); err != nil || left != 0 {
		t.Fatalf("claims left = %d (%v)", left, err)
	}
}
