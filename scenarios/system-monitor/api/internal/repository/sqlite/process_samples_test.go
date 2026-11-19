package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func TestQueryProcessSamplesUsesOwnerScopedHalfOpenRangeAndLimit(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	start := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	if err := repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: start, PID: 2, Owner: "device-control", Comm: "node", Cmdline: "secret command", Cwd: "/private"},
		{Timestamp: start.Add(5 * time.Second), PID: 3, Owner: "other", Comm: "other"},
		{Timestamp: start.Add(5 * time.Second), PID: 1, Owner: "device-control", Comm: "api"},
		{Timestamp: end, PID: 4, Owner: "device-control", Comm: "outside"},
	}); err != nil {
		t.Fatalf("save samples: %v", err)
	}
	rows, err := repo.QueryProcessSamples(ctx, repository.ProcessSampleQuery{Start: start, End: end, Owner: "device-control", Limit: 2})
	if err != nil {
		t.Fatalf("query samples: %v", err)
	}
	if len(rows) != 2 || rows[0].PID != 2 || rows[1].PID != 1 {
		t.Fatalf("rows = %#v, want owner rows in stable timestamp order", rows)
	}
	if rows[0].Cmdline != "" || rows[0].Cwd != "" {
		t.Fatalf("forensic projection unexpectedly returned sensitive process metadata: %#v", rows[0])
	}
}

func TestProcessTimeline_RanksByCPUAcrossRawRows(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Second)

	samples := []repository.ProcessSample{
		{Timestamp: base, PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 80, RSSKB: 1000, SwapKB: 200, SwapStatus: "measured", Threads: 4},
		{Timestamp: base.Add(time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 120, RSSKB: 1200, SwapKB: 400, SwapStatus: "measured", Threads: 4},
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
	if entries[0].RSSKB != 1100 {
		t.Fatalf("rss = %d, want average 1100", entries[0].RSSKB)
	}
	if entries[0].SwapKB != 300 {
		t.Fatalf("swap = %d, want average 300", entries[0].SwapKB)
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
		{Timestamp: minute.Add(5 * time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 60, RSSKB: 100, SwapKB: 50, SwapStatus: "measured", GPUVRAMMB: 40},
		{Timestamp: minute.Add(40 * time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 100, RSSKB: 300, SwapKB: 150, SwapStatus: "measured", GPUVRAMMB: 80},
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
	if e.RSSKB != 200 {
		t.Fatalf("rollup average rss = %d, want 200", e.RSSKB)
	}
	if e.SwapKB != 100 {
		t.Fatalf("rollup average swap = %d, want 100", e.SwapKB)
	}
	if e.GPUVRAMMB != 80 {
		t.Fatalf("rollup max GPU VRAM = %.1f, want 80", e.GPUVRAMMB)
	}
}

func TestProcessTimelineIncludesOnlyFullyContainedRollupMinutes(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	start := time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC)
	var samples []repository.ProcessSample
	for minute := start; minute.Before(start.Add(3 * time.Minute)); minute = minute.Add(time.Minute) {
		for _, offset := range []time.Duration{10 * time.Second, 40 * time.Second} {
			samples = append(samples, repository.ProcessSample{
				Timestamp: minute.Add(offset), PID: 7, Owner: "image-tools", Comm: "sd-cli", CPUPct: 100,
			})
		}
	}
	if err := repo.SaveProcessSamples(ctx, samples); err != nil {
		t.Fatalf("save samples: %v", err)
	}
	if _, err := repo.RollupProcessSamples(ctx, start, start.Add(3*time.Minute)); err != nil {
		t.Fatalf("rollup: %v", err)
	}
	entries, err := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: start.Add(10 * time.Second), End: start.Add(2*time.Minute + 10*time.Second), Top: 10, Rank: "cpu_seconds",
	})
	if err != nil {
		t.Fatalf("query timeline: %v", err)
	}
	if len(entries) != 1 || entries[0].SampleCount != 2 || !entries[0].FirstSeen.Equal(start.Add(time.Minute)) || !entries[0].LastSeen.Equal(start.Add(2*time.Minute)) {
		t.Fatalf("timeline = %+v, want only the fully contained middle-minute rollup", entries)
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
