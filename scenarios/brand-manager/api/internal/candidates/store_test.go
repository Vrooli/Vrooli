package candidates

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func openCandidateStore(t *testing.T) (*sql.DB, Store) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:candidate-lifecycle?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return db, NewSQLiteStore(db, nil)
}

func TestSQLiteCandidateTransitionCASAndInvalidStatus(t *testing.T) {
	db, store := openCandidateStore(t)
	ctx := context.Background()
	if _, err := store.Create(ctx, Candidate{ID: "invalid-initial", BrandID: "brand-1", AssetID: "asset-0", Status: StatusPicked}); err == nil {
		t.Fatal("candidate creation accepted a non-proposed initial status")
	}
	candidate, err := store.Create(ctx, Candidate{ID: "candidate-1", BrandID: "brand-1", AssetID: "asset-1"})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Status != StatusProposed {
		t.Fatalf("initial status = %q", candidate.Status)
	}
	if _, err := store.TransitionStatus(ctx, candidate.ID, StatusProposed, StatusRejected, "no"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TransitionStatus(ctx, candidate.ID, StatusProposed, StatusPicked, ""); err == nil {
		t.Fatal("stale status was accepted")
	} else {
		var conflict ErrStatusConflict
		if !errors.As(err, &conflict) {
			t.Fatalf("stale status error = %v, want conflict", err)
		}
	}
	if _, err := store.TransitionStatus(ctx, candidate.ID, StatusRejected, StatusPicked, ""); err == nil {
		t.Fatal("skipped restore edge was accepted")
	}
	if _, err := store.TransitionStatus(ctx, candidate.ID, StatusRejected, StatusProposed, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE logo_candidates SET status=? WHERE id=?`, "UNKNOWN", candidate.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, candidate.ID); err == nil {
		t.Fatal("unknown persisted status was accepted")
	} else {
		var invalid ErrInvalidStatus
		if !errors.As(err, &invalid) {
			t.Fatalf("unknown status error = %v, want ErrInvalidStatus", err)
		}
	}
}

func TestSQLitePickAtomicallySupersedesPreviousCandidate(t *testing.T) {
	_, store := openCandidateStore(t)
	ctx := context.Background()
	first, err := store.Create(ctx, Candidate{ID: "candidate-1", BrandID: "brand-1", AssetID: "asset-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(ctx, Candidate{ID: "candidate-2", BrandID: "brand-1", AssetID: "asset-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pick(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pick(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	first, err = store.Get(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err = store.Get(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusSuperseded || second.Status != StatusPicked {
		t.Fatalf("after second pick: first=%q second=%q", first.Status, second.Status)
	}
	if _, err := store.Pick(ctx, first.ID); err != nil {
		t.Fatalf("superseded candidate should support direct re-pick: %v", err)
	}
	first, _ = store.Get(ctx, first.ID)
	second, _ = store.Get(ctx, second.ID)
	if first.Status != StatusPicked || second.Status != StatusSuperseded {
		t.Fatalf("re-pick result = first:%q second:%q", first.Status, second.Status)
	}
}
