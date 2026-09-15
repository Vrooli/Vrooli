package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func TestMetricCycleSurvivesFreshSchemaRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "system-monitor.db")
	when := time.Date(2026, 8, 20, 12, 0, 0, 456, time.UTC)

	repo, err := NewRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveMetricCycle(context.Background(), "cycle-restart", when, []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 7.25}},
	}); err != nil {
		repo.Close()
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	rows, err := reopened.GetMetrics(context.Background(), repository.MetricsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CycleID != "cycle-restart" || !rows[0].Timestamp.Equal(when) {
		t.Fatalf("restart lost cycle: %#v", rows)
	}
}

func TestMetricCyclePersistsOneLogicalSample(t *testing.T) {
	repo, err := NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	when := time.Date(2026, 8, 20, 12, 0, 0, 456, time.UTC)
	if err := repo.SaveMetricCycle(context.Background(), "cycle-sqlite", when, []repository.MetricObservation{
		{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 12.5}},
		{CollectorName: "network", Values: map[string]interface{}{"status": "unsupported", "reason": "backend unavailable"}},
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CycleID != "cycle-sqlite" || !rows[0].Timestamp.Equal(when) {
		t.Fatalf("unexpected cycle rows: %#v", rows)
	}
	if rows[0].ConnectionsState.Status != "unsupported" {
		t.Fatalf("network state = %#v", rows[0].ConnectionsState)
	}
	if rows[0].CPUState.Status != "measured" || rows[0].CPUState.Value != 12.5 {
		t.Fatalf("CPU state = %#v", rows[0].CPUState)
	}
	if err := repo.SaveMetricCycle(context.Background(), "cycle-sqlite", when, nil); err == nil {
		t.Fatal("duplicate cycle was accepted")
	}
}
