package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// Swap rides along inside the memory collector's payload. It must surface as
// its own timeline series, because a host can report healthy memory
// utilisation while swap fills — the divergence a single memory line hides.
func TestMemoryCycleProjectsSwapAsItsOwnSeries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "system-monitor.db")
	when := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	if err := repo.SaveMetricCycle(context.Background(), "cycle-swap", when, []repository.MetricObservation{
		{CollectorName: "memory", Values: map[string]interface{}{
			"status":        "measured",
			"usage_percent": 37.4,
			"swap": map[string]interface{}{
				"percent": 33.07,
				"total":   74088177664,
				"used":    24501260288,
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	results, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{
		TimeRange: repository.TimeRange{
			StartTime: when.Add(-time.Minute),
			EndTime:   when.Add(time.Minute),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	got := results[0]

	if got.SwapUsage == nil {
		t.Fatal("SwapUsage is nil: swap must be projected as its own series, not left inside the memory payload")
	}
	if *got.SwapUsage != 33.07 {
		t.Errorf("SwapUsage = %v, want 33.07", *got.SwapUsage)
	}
	if got.MemoryUsage != 37.4 {
		t.Errorf("MemoryUsage = %v, want 37.4 (memory must stay independent of swap)", got.MemoryUsage)
	}
	if got.SwapState.Status != "measured" {
		t.Errorf("SwapState.Status = %q, want %q", got.SwapState.Status, "measured")
	}
	if got.SwapState.Provenance != "system-monitor/memory.swap" {
		t.Errorf("SwapState.Provenance = %q, want %q", got.SwapState.Provenance, "system-monitor/memory.swap")
	}
	// The typed MetricValue the UI actually plots is built from
	// MetricState.Value, not from SwapUsage. If this carries the memory
	// reading, the swap line silently mirrors the memory line.
	if got.SwapState.Value != 33.07 {
		t.Errorf("SwapState.Value = %v, want 33.07 (must be swap, not the memory reading %v)", got.SwapState.Value, got.MemoryUsage)
	}
	if got.MemoryState.Value != 37.4 {
		t.Errorf("MemoryState.Value = %v, want 37.4", got.MemoryState.Value)
	}
}

// A memory payload with no swap block must leave the swap series absent rather
// than reporting a fabricated zero, which would read as "no swap in use".
func TestMemoryCycleWithoutSwapLeavesSwapUnset(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "system-monitor.db")
	when := time.Date(2026, 8, 21, 20, 0, 0, 0, time.UTC)

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	if err := repo.SaveMetricCycle(context.Background(), "cycle-noswap", when, []repository.MetricObservation{
		{CollectorName: "memory", Values: map[string]interface{}{"status": "measured", "usage_percent": 41.0}},
	}); err != nil {
		t.Fatal(err)
	}

	results, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{
		TimeRange: repository.TimeRange{StartTime: when.Add(-time.Minute), EndTime: when.Add(time.Minute)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].SwapUsage != nil {
		t.Errorf("SwapUsage = %v, want nil when the collector reported no swap block", *results[0].SwapUsage)
	}
}

func TestPressureCycleProjectsFlowSeriesAndOmitsUnsupportedFragmentationValue(t *testing.T) {
	repo, err := NewRepository(filepath.Join(t.TempDir(), "system-monitor.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	when := time.Date(2026, 8, 21, 21, 0, 0, 0, time.UTC)
	if err := repo.SaveMetricCycle(context.Background(), "cycle-pressure", when, []repository.MetricObservation{{CollectorName: "pressure", Values: map[string]interface{}{
		"swap_traffic_pages_per_second": 12.5,
		"swap_traffic_rate_status":      "measured",
		"pgmajfault_per_second":         3.25,
		"pgmajfault_rate_status":        "measured",
		"fragmentation_status":          "unsupported",
		"fragmentation_reason":          "buddy allocator is Linux-only",
	}}}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{TimeRange: repository.TimeRange{StartTime: when.Add(-time.Minute), EndTime: when.Add(time.Minute)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].SwapTrafficState.Status != "measured" || rows[0].SwapTrafficState.Value != 12.5 {
		t.Fatalf("swap flow state = %+v", rows[0].SwapTrafficState)
	}
	if rows[0].MajorFaultsState.Status != "measured" || rows[0].MajorFaultsState.Value != 3.25 {
		t.Fatalf("fault state = %+v", rows[0].MajorFaultsState)
	}
	if rows[0].FragmentationIndexState.Status != "unsupported" || rows[0].FragmentationIndexState.Value != 0 {
		t.Fatalf("fragmentation state = %+v", rows[0].FragmentationIndexState)
	}
}
