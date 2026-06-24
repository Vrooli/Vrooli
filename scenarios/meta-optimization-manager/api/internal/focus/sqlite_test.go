package focus_test

import (
	"context"
	"testing"
	"time"

	internalfocus "meta-optimization-manager/internal/focus"
	"meta-optimization-manager/internal/testutil/db"
	"meta-optimization-manager/internal/testutil/mocks"

	"github.com/vrooli/api-core/spacedoc"
)

func TestGapsRepositoryRoundTrip(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internalfocus.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	clk := mocks.NewFakeClock(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))
	repo := internalfocus.NewSQLiteRepository(h, clk)
	ctx := context.Background()

	g := internalfocus.Gap{
		ID:           "answer/1",
		Projection:   spacedoc.ProjectionAnswer,
		Title:        "explain domain map",
		Status:       spacedoc.StatusMissing,
		SourceCellID: "1",
		Notes:        []string{"from the space doc"},
		Approaches:   []string{"cartographer provider"},
	}
	if err := repo.Upsert(ctx, g); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, ok, err := repo.Get(ctx, "answer/1")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.Title != g.Title || got.Status != spacedoc.StatusMissing || len(got.Approaches) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	// Upsert again (append-style) replaces the row.
	g.Approaches = append(g.Approaches, "code-facts")
	if err := repo.Upsert(ctx, g); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	got, _, _ = repo.Get(ctx, "answer/1")
	if len(got.Approaches) != 2 {
		t.Fatalf("want 2 approaches after re-upsert, got %+v", got.Approaches)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: len=%d err=%v", len(list), err)
	}
}

func TestGapsRepositoryGetMissing(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internalfocus.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := internalfocus.NewSQLiteRepository(h, mocks.NewFakeClock(time.Now()))
	if _, ok, err := repo.Get(context.Background(), "nope"); ok || err != nil {
		t.Fatalf("expected miss, got ok=%v err=%v", ok, err)
	}
}
