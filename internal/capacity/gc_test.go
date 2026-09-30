package capacity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

// TestGCTerminalClaimsRetentionBoundary proves GC prunes terminal claims strictly
// older than the cutoff, never prunes active claims, and never prunes a terminal
// claim exactly at the boundary (the cutoff is exclusive).
func TestGCTerminalClaimsRetentionBoundary(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	clk := testenv.NewClock(t0)
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	// A terminal claim, released at t0.
	term, err := store.CreateClaim(ctx, sampleClaim(), time.Hour)
	if err != nil {
		t.Fatalf("create terminal claim: %v", err)
	}
	if _, err := store.ReleaseClaim(ctx, term.ClaimID); err != nil {
		t.Fatalf("release: %v", err)
	}
	// An active claim, created at t0 — must never be pruned.
	active := sampleClaim()
	active.OwnerID = "kyutai-stt"
	activeClaim, err := store.CreateClaim(ctx, active, time.Hour)
	if err != nil {
		t.Fatalf("create active claim: %v", err)
	}

	// Boundary: cutoff == updated_at => nothing pruned (exclusive `< cutoff`).
	res, err := store.GCTerminalClaims(ctx, t0)
	if err != nil {
		t.Fatalf("gc at boundary: %v", err)
	}
	if res.Count != 0 {
		t.Fatalf("boundary cutoff must prune nothing, pruned %d", res.Count)
	}

	// Past the boundary: the terminal claim is pruned, the active one survives.
	res, err = store.GCTerminalClaims(ctx, t0.Add(time.Second))
	if err != nil {
		t.Fatalf("gc past boundary: %v", err)
	}
	if res.Count != 1 {
		t.Fatalf("expected 1 pruned, got %d", res.Count)
	}
	if res.Bytes != term.AmountBytes {
		t.Errorf("pruned bytes = %d, want %d", res.Bytes, term.AmountBytes)
	}
	if _, err := store.GetClaim(ctx, term.ClaimID); !errors.Is(err, ErrNotFound) {
		t.Errorf("terminal claim should be gone, got err=%v", err)
	}
	if _, err := store.GetClaim(ctx, activeClaim.ClaimID); err != nil {
		t.Errorf("active claim must survive GC: %v", err)
	}
}

// TestGCNeverPrunesActiveStatuses proves reserved/granted/degraded are immune to
// GC regardless of how old their updated_at is.
func TestGCNeverPrunesActiveStatuses(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	if _, err := store.CreateClaim(ctx, sampleClaim(), time.Hour); err != nil {
		t.Fatalf("create: %v", err)
	}
	// A far-future cutoff would prune anything eligible.
	res, err := store.GCTerminalClaims(ctx, clk.Now().Add(1000*time.Hour))
	if err != nil {
		t.Fatalf("gc: %v", err)
	}
	if res.Count != 0 {
		t.Fatalf("active claim must never be pruned, pruned %d", res.Count)
	}
}

// TestNewSQLiteStoreRefusesLiveDefaultPathUnderTest is the hard test-isolation
// seam: a capacity test that forgets to pass Config.DBPath/HomeDir would silently
// write the live ledger (VROOLI_HOME does not isolate it). The store must refuse.
func TestNewSQLiteStoreRefusesLiveDefaultPathUnderTest(t *testing.T) {
	_, err := NewSQLiteStore(context.Background(), Config{})
	if err == nil {
		t.Fatal("NewSQLiteStore with empty Config must refuse the live default path under test")
	}
}
