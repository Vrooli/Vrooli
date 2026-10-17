package capacity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

// Feature: an owner whose footprint changes resizes, it does not churn
//
//	As the capacity ledger
//	I want one row and one observed-usage history per resource lifetime
//	So that the right-sizing advisory has data to read, instead of being reset
//	by a release-and-reclaim on every model load.

// newResizeStore is a ledger with a fixed clock, so the tests never depend on
// wall-clock movement.
func newResizeStore(t *testing.T) *SQLiteStore {
	t.Helper()
	return testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: testenv.NewClock(time.Date(2026, 8, 21, 6, 0, 0, 0, time.UTC))})
	})
}

func grantedVRAMClaim(t *testing.T, store *SQLiteStore, owner string, amount int64) CapacityClaim {
	t.Helper()
	claim, err := store.CreateClaim(context.Background(), CapacityClaim{
		OwnerKind:      OwnerKindResource,
		OwnerID:        owner,
		ResourceKind:   ResourceKindVRAM,
		AmountBytes:    amount,
		PreferredBytes: amount,
		FloorBytes:     amount / 4,
		Priority:       PriorityService,
		Status:         StatusGranted,
	}, 0)
	if err != nil {
		t.Fatalf("CreateClaim: %v", err)
	}
	return claim
}

// Scenario: a resize keeps the claim's identity and its observed history.
func TestResizeClaimKeepsIdentityAndObservedHistory(t *testing.T) {
	// Given a granted claim with a recorded observed peak
	store := newResizeStore(t)
	claim := grantedVRAMClaim(t, store, "ollama", 4<<30)
	observed, err := store.RecordObserved(context.Background(), claim.ClaimID, 600<<20, 900<<20, time.Date(2026, 8, 21, 6, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RecordObserved: %v", err)
	}

	// When the owner's real footprint grows and the claim is resized
	resized, err := store.ResizeClaim(context.Background(), claim.ClaimID, observed.Generation, 12<<30)
	if err != nil {
		t.Fatalf("ResizeClaim: %v", err)
	}

	// Then it is the same claim, at the new amount
	if resized.ClaimID != claim.ClaimID {
		t.Fatalf("ClaimID = %q, want %q; a resize must not create a new row", resized.ClaimID, claim.ClaimID)
	}
	if resized.AmountBytes != 12<<30 || resized.PreferredBytes != 12<<30 {
		t.Fatalf("amount/preferred = %d/%d, want both %d", resized.AmountBytes, resized.PreferredBytes, int64(12)<<30)
	}
	// And it is granted, not degraded: a resize is an owner-side fact, not a
	// broker instruction
	if resized.Status != StatusGranted {
		t.Fatalf("Status = %q, want %q", resized.Status, StatusGranted)
	}
	// And the observed history survives, which is what right-sizing reads
	if resized.ObservedPeakBytes != 900<<20 {
		t.Fatalf("ObservedPeakBytes = %d, want %d; a resize must not discard observed usage", resized.ObservedPeakBytes, int64(900)<<20)
	}
	// And the generation advanced, so a concurrent writer is caught
	if resized.Generation <= observed.Generation {
		t.Fatalf("Generation = %d, want greater than %d", resized.Generation, observed.Generation)
	}

	// And the ledger still holds exactly one claim for the owner
	claims, err := store.ListClaims(context.Background(), ClaimFilter{OwnerID: "ollama"})
	if err != nil {
		t.Fatalf("ListClaims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("ledger holds %d claims for ollama, want 1", len(claims))
	}
}

// Scenario: a resize under a stale generation is refused.
func TestResizeClaimRefusesAStaleGeneration(t *testing.T) {
	// Given a claim that has already moved on
	store := newResizeStore(t)
	claim := grantedVRAMClaim(t, store, "whisper", 4<<30)
	if _, err := store.ResizeClaim(context.Background(), claim.ClaimID, claim.Generation, 6<<30); err != nil {
		t.Fatalf("first ResizeClaim: %v", err)
	}

	// When a writer resizes with the generation it read before that
	_, err := store.ResizeClaim(context.Background(), claim.ClaimID, claim.Generation, 8<<30)

	// Then it is refused rather than silently overwriting
	if err == nil {
		t.Fatal("ResizeClaim() = nil error for a stale generation; the optimistic-concurrency guard did not hold")
	}
}

// Scenario: a resize to zero is refused, because that is a release.
func TestResizeClaimRefusesANonPositiveAmount(t *testing.T) {
	store := newResizeStore(t)
	claim := grantedVRAMClaim(t, store, "reranker", 2<<30)

	for _, amount := range []int64{0, -1} {
		// When a resize to a non-positive amount is attempted
		_, err := store.ResizeClaim(context.Background(), claim.ClaimID, claim.Generation, amount)

		// Then it is refused, and the message points at the right verb
		if !errors.Is(err, ErrInvalidClaim) {
			t.Fatalf("ResizeClaim(%d) = %v, want ErrInvalidClaim", amount, err)
		}
		if !strings.Contains(err.Error(), "release the claim") {
			t.Fatalf("ResizeClaim(%d) message = %q, want it to name release", amount, err)
		}
	}
}

// Scenario: fifty resizes leave one row.
func TestFiftyResizesLeaveOneLedgerRow(t *testing.T) {
	// Given a granted claim
	store := newResizeStore(t)
	claim := grantedVRAMClaim(t, store, "ollama", 4<<30)
	generation := claim.Generation

	// When its owner's footprint changes fifty times
	for cycle := range 50 {
		resized, err := store.ResizeClaim(context.Background(), claim.ClaimID, generation, int64(4<<30)+int64(cycle)*int64(1<<28))
		if err != nil {
			t.Fatalf("resize cycle %d: %v", cycle, err)
		}
		generation = resized.Generation
	}

	// Then the ledger holds exactly one claim
	claims, err := store.ListClaims(context.Background(), ClaimFilter{OwnerID: "ollama"})
	if err != nil {
		t.Fatalf("ListClaims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("fifty resizes left %d ledger rows, want 1", len(claims))
	}
}
