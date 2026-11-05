package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func TestKernelMemoryHistorySurvivesRepositoryReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitor.db")
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()
	for i, value := range []float64{13000, 14000} {
		if err := repo.SaveMetricCycle(ctx, when.Add(time.Duration(i)*time.Minute).Format(time.RFC3339), when.Add(time.Duration(i)*time.Minute), []repository.MetricObservation{{CollectorName: "memory", Values: map[string]interface{}{
			"status": "measured", "usage_percent": float64(48), "kernel_unreclaimable_bytes_status": "measured", "kernel_unreclaimable_bytes": value,
		}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	repo, err = NewRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	rows, err := repo.GetMetrics(ctx, repository.MetricsFilter{TimeRange: repository.TimeRange{StartTime: when.Add(-time.Minute), EndTime: when.Add(2 * time.Minute)}})
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%d error=%v", len(rows), err)
	}
	for i, row := range rows {
		if row.KernelMemory == nil || row.KernelMemory.UnreclaimableBytes.Status != "measured" || row.KernelMemory.UnreclaimableBytes.Value != float64(13000+i*1000) {
			t.Fatalf("history[%d]=%+v", i, row.KernelMemory)
		}
		if row.KernelMemory.SlabBytes.Status != "not_yet_sampled" {
			t.Fatal("absent historical field fabricated measured zero")
		}
	}
}
