package services

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func TestRetentionScheduler_ProcessRawThenRollup(t *testing.T) {
	repo := memory.NewRepository()
	ctx := context.Background()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	// One row well inside the raw window (kept) and two old rows (rolled up).
	old := now.Add(-8 * time.Hour)
	_ = repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: now.Add(-time.Minute), PID: 1, Comm: "fresh", Owner: "a", CPUPct: 10, RSSKB: 100},
		{Timestamp: old, PID: 2, Comm: "stale", Owner: "b", CPUPct: 20, RSSKB: 200},
		{Timestamp: old.Add(10 * time.Second), PID: 2, Comm: "stale", Owner: "b", CPUPct: 40, RSSKB: 400},
	})

	s := &RetentionScheduler{
		log:             slog.Default(),
		clock:           &StubClock{current: now},
		procRepo:        repo,
		rawRetention:    6 * time.Hour,
		rollupRetention: 30 * 24 * time.Hour,
	}
	s.runProcessRetention(ctx)

	// The fresh row stays raw; the two stale rows collapse to one rollup entry.
	entries, err := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: now.Add(-24 * time.Hour), End: now.Add(time.Minute), Top: 10,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	var fresh, rolled *repository.ProcessTimelineEntry
	for i := range entries {
		switch entries[i].Owner {
		case "a":
			fresh = &entries[i]
		case "b":
			rolled = &entries[i]
		}
	}
	if fresh == nil || fresh.Aggregated {
		t.Fatalf("fresh row should remain raw (not aggregated): %+v", fresh)
	}
	if rolled == nil || !rolled.Aggregated {
		t.Fatalf("stale rows should be aggregated into a rollup: %+v", rolled)
	}
	if rolled.SampleCount != 2 {
		t.Fatalf("rollup sample_count = %d, want 2", rolled.SampleCount)
	}
	// avg of 20 and 40 = 30.
	if rolled.CPUPct < 29 || rolled.CPUPct > 31 {
		t.Fatalf("rollup avg CPU = %f, want ~30", rolled.CPUPct)
	}
}

func TestRetentionScheduler_ProcessRollupPrune(t *testing.T) {
	repo := memory.NewRepository()
	ctx := context.Background()
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	// A very old sample that, once rolled up, should also be pruned from the
	// rollup table by the short rollup-retention window.
	_ = repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: now.Add(-40 * 24 * time.Hour), PID: 1, Comm: "ancient", Owner: "a", CPUPct: 5},
	})

	s := &RetentionScheduler{
		log:             slog.Default(),
		clock:           &StubClock{current: now},
		procRepo:        repo,
		rawRetention:    6 * time.Hour,
		rollupRetention: 30 * 24 * time.Hour,
	}
	s.runProcessRetention(ctx)

	entries, _ := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: now.Add(-365 * 24 * time.Hour), End: now, Top: 10,
	})
	if len(entries) != 0 {
		t.Fatalf("ancient rollup should be pruned, got %+v", entries)
	}
}
