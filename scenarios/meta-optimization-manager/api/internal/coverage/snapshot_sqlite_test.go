package coverage_test

import (
	"context"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	internalcoverage "meta-optimization-manager/internal/coverage"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/vrooli/api-core/spacedoc"
)

func TestSnapshotSaveAndLatestTTL(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internalcoverage.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	clk := scheduletest.New(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))
	repo := internalcoverage.NewSQLiteSnapshotRepository(h, clk)
	ctx := context.Background()

	status := internalcoverage.Status{
		ComputedAt:            clk.Now(),
		CoverageMethodVersion: "answer-active-reachable-fresh-eval-v2",
		Projections: []internalcoverage.ProjectionCoverage{
			{Projection: spacedoc.ProjectionAnswer, NowCount: 3, TotalCells: 36, CoverageRatio: 0.0833, Available: true, DenominatorConfidence: spacedoc.ConfidencePartial},
		},
	}
	if err := repo.Save(ctx, status); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Within TTL -> hit.
	got, ok := repo.Latest(ctx, 30*time.Second, clk.Now().Add(10*time.Second))
	if !ok {
		t.Fatal("expected a fresh snapshot hit")
	}
	if len(got.Projections) != 1 || got.Projections[0].NowCount != 3 {
		t.Errorf("round-trip mismatch: %+v", got.Projections)
	}
	if got.CoverageMethodVersion != "answer-active-reachable-fresh-eval-v2" {
		t.Errorf("coverage method version = %q", got.CoverageMethodVersion)
	}

	// Past TTL -> miss.
	if _, ok := repo.Latest(ctx, 30*time.Second, clk.Now().Add(2*time.Minute)); ok {
		t.Error("expected a stale-snapshot miss past TTL")
	}
}

func TestSnapshotLatestEmpty(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internalcoverage.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := internalcoverage.NewSQLiteSnapshotRepository(h, scheduletest.New(time.Now()))
	if _, ok := repo.Latest(context.Background(), time.Minute, time.Now()); ok {
		t.Error("expected miss on empty table")
	}
}
