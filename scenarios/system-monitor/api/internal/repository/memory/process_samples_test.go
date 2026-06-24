package memory

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func TestMemoryProcessTimeline_RollupAndQuery(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()
	minute := time.Date(2026, 6, 24, 10, 30, 0, 0, time.UTC)

	_ = repo.SaveProcessSamples(ctx, []repository.ProcessSample{
		{Timestamp: minute.Add(5 * time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 60, RSSKB: 100},
		{Timestamp: minute.Add(40 * time.Second), PID: 1, Comm: "osv-scanner", Owner: "security-health", CPUPct: 100, RSSKB: 300},
		{Timestamp: minute.Add(2 * time.Second), PID: 2, Comm: "node", Owner: "web-console", CPUPct: 10, RSSKB: 5000},
	})

	// Before rollup: ranked by CPU, security-health first.
	entries, err := repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: minute.Add(-time.Minute), End: minute.Add(2 * time.Minute), Top: 10,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(entries) != 2 || entries[0].Owner != "security-health" {
		t.Fatalf("pre-rollup ranking wrong: %+v", entries)
	}

	// Roll up the security-health/web-console minute.
	res, err := repo.RollupProcessSamples(ctx, time.Time{}, minute.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("rollup: %v", err)
	}
	if res.RawRowsConsumed != 3 {
		t.Fatalf("consumed %d, want 3", res.RawRowsConsumed)
	}

	entries, _ = repo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: minute.Add(-time.Minute), End: minute.Add(2 * time.Minute), Top: 10,
	})
	if len(entries) != 2 {
		t.Fatalf("post-rollup entries = %d, want 2", len(entries))
	}
	if !entries[0].Aggregated || entries[0].Owner != "security-health" {
		t.Fatalf("post-rollup top entry wrong: %+v", entries[0])
	}
	if entries[0].CPUPct < 79 || entries[0].CPUPct > 81 {
		t.Fatalf("rollup avg = %f, want ~80", entries[0].CPUPct)
	}
}
