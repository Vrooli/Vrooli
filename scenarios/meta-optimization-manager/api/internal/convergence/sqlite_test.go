package convergence_test

import (
	"context"
	"testing"
	"time"

	internalconv "meta-optimization-manager/internal/convergence"

	db "github.com/vrooli/api-core/databasetest"
)

func TestRepositorySaveAndTrend(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internalconv.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := internalconv.NewSQLiteRepository(h)
	ctx := context.Background()

	day1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	if err := repo.SaveFitness(ctx, []internalconv.TemplateFitness{{Template: "react-vite", PerReplicaCost: 1200, CoordinatedEditCount: 9, Tier: internalconv.TierFair}}, day1); err != nil {
		t.Fatalf("save day1: %v", err)
	}
	if err := repo.SaveFitness(ctx, []internalconv.TemplateFitness{{Template: "react-vite", PerReplicaCost: 900, CoordinatedEditCount: 5, Tier: internalconv.TierStrong}}, day2); err != nil {
		t.Fatalf("save day2: %v", err)
	}

	pts, err := repo.Trend(ctx, "react-vite")
	if err != nil {
		t.Fatalf("trend: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("want 2 trend points, got %d", len(pts))
	}
	// Oldest first: per-replica cost should fall (the compounding proof).
	if pts[0].PerReplicaCost != 1200 || pts[1].PerReplicaCost != 900 {
		t.Fatalf("trend order/values wrong: %+v", pts)
	}
	if !pts[0].At.Equal(day1) {
		t.Fatalf("trend timestamp not round-tripped: %v", pts[0].At)
	}
}

func TestRepositorySaveReferences(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internalconv.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := internalconv.NewSQLiteRepository(h)
	err := repo.SaveReferences(context.Background(), []internalconv.ReferenceHealth{
		{Scenario: "reference-react-vite", StabilityDays: 61, Breadth: 3, CleanOnAllTools: true, Eligibility: internalconv.EligibilityEligible},
	}, time.Now())
	if err != nil {
		t.Fatalf("save references: %v", err)
	}
}
