package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func TestMetricCycleKeepsCollectorsTogetherWhenTimesDiffer(t *testing.T) {
	repo := memory.NewRepository()
	observed := time.Date(2026, 8, 20, 12, 0, 0, 123, time.UTC)
	err := repo.SaveMetricCycle(context.Background(), "cycle-1", observed, []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 42.0}},
		{CollectorName: "memory", Values: map[string]interface{}{"status": "failed", "reason": "permission denied", "usage_percent": 0.0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d timeline rows, want one cycle", len(rows))
	}
	if rows[0].CycleID != "cycle-1" || !rows[0].Timestamp.Equal(observed) {
		t.Fatalf("cycle identity/time not preserved: %#v", rows[0])
	}
	if rows[0].CPUState.Status != "measured" || rows[0].CPUState.Value != 42 {
		t.Fatalf("CPU state = %#v", rows[0].CPUState)
	}
	if rows[0].MemoryState.Status != "failed" || rows[0].MemoryUsage != 0 {
		t.Fatalf("failed memory observation fabricated a value: %#v", rows[0])
	}
	if err := repo.SaveMetricCycle(context.Background(), "cycle-1", observed, nil); err == nil {
		t.Fatal("duplicate cycle was accepted")
	}
}
