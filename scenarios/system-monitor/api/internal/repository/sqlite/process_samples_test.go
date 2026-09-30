package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func TestProcessTimeline_RanksByCPUAcrossRawRows(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Second)

	samples := []repository.ProcessSample{
		{Timestamp: base, PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 80, RSSKB: 1000, Threads: 4},
		{Timestamp: base.Add(time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 120, RSSKB: 1200, Threads: 4},
		{Timestamp: base, PID: 2, Comm: "node", Owner: "web-console", CPUPct: 10, RSSKB: 5000, Threads: 8},
	}
	if err := repo.SaveProcessSamples(ctx, samples); err != nil {
		t.Fatalf("save: %v", err)
	}

	entries, err := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: base.Add(-time.Minute), End: base.Add(time.Minute), Top: 10,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	// security-health averages (80+120)/2 = 100% > web-console 10% -> ranked first.
	if entries[0].Owner != "security-health" {
		t.Fatalf("rank: first owner = %q, want security-health", entries[0].Owner)
	}
	if entries[0].CPUPct < 99 || entries[0].CPUPct > 101 {
		t.Fatalf("security-health avg CPU = %f, want ~100", entries[0].CPUPct)
	}
	if entries[0].SampleCount != 2 {
		t.Fatalf("sample_count = %d, want 2", entries[0].SampleCount)
	}
	if entries[0].RSSKB != 1200 {
		t.Fatalf("rss = %d, want max 1200", entries[0].RSSKB)
	}
}

func TestProcessTimeline_OwnerFilter(t *testing.T) {
	repo, _ := NewInMemoryRepository()
	defer repo.Close()
	ctx := context.Background()
	base := time.Now().UTC()

	_ = repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: base, PID: 1, Comm: "a", Owner: "security-health", CPUPct: 50},
		{Timestamp: base, PID: 2, Comm: "b", Owner: "web-console", CPUPct: 90},
	})

	entries, err := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: base.Add(-time.Minute), End: base.Add(time.Minute), Owner: "security-health", Top: 10,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 1 || entries[0].Owner != "security-health" {
		t.Fatalf("owner filter returned %+v", entries)
	}
}

func TestRollupProcessSamples_AggregatesAndPrunesRaw(t *testing.T) {
	repo, _ := NewInMemoryRepository()
	defer repo.Close()
	ctx := context.Background()

	// Two raw rows in the same minute for the same owner+comm.
	minute := time.Date(2026, 6, 24, 10, 30, 0, 0, time.UTC)
	_ = repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: minute.Add(5 * time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 60, RSSKB: 100},
		{Timestamp: minute.Add(40 * time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 100, RSSKB: 300},
	})

	res, err := repo.RollupProcessSamples(ctx, time.Time{}, minute.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("rollup: %v", err)
	}
	if res.RawRowsConsumed != 2 {
		t.Fatalf("consumed %d raw rows, want 2", res.RawRowsConsumed)
	}
	if res.RollupRows != 1 {
		t.Fatalf("produced %d rollup rows, want 1", res.RollupRows)
	}

	// Raw rows must be gone after rollup.
	raw, _ := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: minute.Add(-time.Minute), End: minute.Add(2 * time.Minute), Top: 10,
	})
	if len(raw) != 1 {
		t.Fatalf("post-rollup entries = %d, want 1 (from rollup)", len(raw))
	}
	e := raw[0]
	if !e.Aggregated {
		t.Fatalf("entry should be marked aggregated")
	}
	// avg of 60 and 100 = 80.
	if e.CPUPct < 79 || e.CPUPct > 81 {
		t.Fatalf("rollup avg CPU = %f, want ~80", e.CPUPct)
	}
	if e.SampleCount != 2 {
		t.Fatalf("rollup sample_count = %d, want 2", e.SampleCount)
	}
	if e.RSSKB != 300 {
		t.Fatalf("rollup max rss = %d, want 300", e.RSSKB)
	}
}

func TestPruneProcessRollupsBefore(t *testing.T) {
	repo, _ := NewInMemoryRepository()
	defer repo.Close()
	ctx := context.Background()

	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: old.Add(time.Second), PID: 1, Comm: "x", Owner: "a", CPUPct: 5},
	})
	if _, err := repo.RollupProcessSamples(ctx, time.Time{}, old.Add(time.Hour)); err != nil {
		t.Fatalf("rollup: %v", err)
	}
	pruned, err := repo.PruneProcessRollupsBefore(ctx, old.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("pruned %d rollups, want 1", pruned)
	}
}
