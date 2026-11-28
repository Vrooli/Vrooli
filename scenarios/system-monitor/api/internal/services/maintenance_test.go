package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func seedMetrics(t *testing.T, repo *memory.MemoryRepository, ages ...time.Duration) {
	t.Helper()
	for range ages {
		if err := repo.SaveMetricCycle(context.Background(), fmt.Sprintf("maintenance-%d", time.Now().UnixNano()), time.Now().UTC(), []repository.MetricObservation{{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": 1.0}}}); err != nil {
			t.Fatalf("SaveMetricCycle: %v", err)
		}
	}
}

func TestMaintenanceService_Cutoff(t *testing.T) {
	clock := NewStubClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	svc := NewMetricsMaintenanceService(memory.NewRepository(), WithMaintenanceClock(clock))

	got := svc.Cutoff(30)
	want := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Cutoff(30) = %v, want %v", got, want)
	}
}

func TestMaintenanceService_RetentionPreviewValidatesDays(t *testing.T) {
	svc := NewMetricsMaintenanceService(memory.NewRepository())
	if _, _, err := svc.RetentionPreview(context.Background(), 0); !errors.Is(err, ErrInvalidRetentionDays) {
		t.Errorf("expected ErrInvalidRetentionDays, got %v", err)
	}
}

func TestMaintenanceService_RetentionApplyRequiresConfirmation(t *testing.T) {
	svc := NewMetricsMaintenanceService(memory.NewRepository())
	if _, _, _, err := svc.RetentionApply(context.Background(), 30, false); !errors.Is(err, ErrConfirmationRequired) {
		t.Errorf("expected ErrConfirmationRequired, got %v", err)
	}
}

func TestMaintenanceService_RetentionPreviewEmptyRepo(t *testing.T) {
	svc := NewMetricsMaintenanceService(memory.NewRepository())
	est, _, err := svc.RetentionPreview(context.Background(), 30)
	if err != nil {
		t.Fatalf("RetentionPreview: %v", err)
	}
	if est.RowCount != 0 {
		t.Errorf("RowCount = %d, want 0 for empty repo", est.RowCount)
	}
}

func TestMaintenanceService_RetentionApplyPrunesOldRows(t *testing.T) {
	// Clock in the future so all rows (stamped ~now) fall before the cutoff.
	clock := NewStubClock(time.Now().Add(48 * time.Hour))
	repo := memory.NewRepository()
	seedMetrics(t, repo, time.Hour, time.Hour, time.Hour)

	svc := NewMetricsMaintenanceService(repo, WithMaintenanceClock(clock))
	res, _, _, err := svc.RetentionApply(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("RetentionApply: %v", err)
	}
	if res.DeletedRows != 3 {
		t.Errorf("DeletedRows = %d, want 3", res.DeletedRows)
	}
}

func TestMaintenanceService_CompactionUnsupportedOnMemory(t *testing.T) {
	svc := NewMetricsMaintenanceService(memory.NewRepository())
	if _, _, err := svc.CompactionPreview(context.Background()); !errors.Is(err, repository.ErrNotSupported) {
		t.Errorf("expected ErrNotSupported from memory CompactionPreview, got %v", err)
	}
	if _, err := svc.CompactionApply(context.Background(), true); !errors.Is(err, repository.ErrNotSupported) {
		t.Errorf("expected ErrNotSupported from memory CompactionApply, got %v", err)
	}
}

func TestMaintenanceService_RunScheduledRetention(t *testing.T) {
	clock := NewStubClock(time.Now().Add(48 * time.Hour))
	repo := memory.NewRepository()
	seedMetrics(t, repo, time.Hour, time.Hour)

	svc := NewMetricsMaintenanceService(repo, WithMaintenanceClock(clock))
	// CompactAfterRetention=true must not fail even though memory cannot compact.
	res, err := svc.RunScheduledRetention(context.Background(), Settings{MetricsRetentionDays: 1, CompactAfterRetention: true})
	if err != nil {
		t.Fatalf("RunScheduledRetention: %v", err)
	}
	if res.DeletedRows != 2 {
		t.Errorf("DeletedRows = %d, want 2", res.DeletedRows)
	}
}

func TestMaintenanceService_RunScheduledRetentionAppliesShorterCollectorWindows(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	repo := memory.NewRepository()
	save := func(collector string, age time.Duration) {
		t.Helper()
		if err := repo.SaveMetricCycle(context.Background(), fmt.Sprintf("%s-%d", collector, age), now.Add(-age), []repository.MetricObservation{{CollectorName: collector, Values: map[string]interface{}{"v": 1.0}}}); err != nil {
			t.Fatalf("SaveMetricCycle: %v", err)
		}
	}
	day := 24 * time.Hour
	save("device_graph", 3*day) // older than its 2d window -> pruned
	save("device_graph", 1*day) // inside its window -> kept
	save("cpu", 3*day)          // no override, inside global 30d -> kept
	save("network", 40*day)     // override longer than global is ignored; global prunes it

	svc := NewMetricsMaintenanceService(repo,
		WithMaintenanceClock(NewStubClock(now)),
		WithCollectorRetention(map[string]time.Duration{"device_graph": 2 * day, "network": 60 * day}),
	)
	res, err := svc.RunScheduledRetention(context.Background(), Settings{MetricsRetentionDays: 30})
	if err != nil {
		t.Fatalf("RunScheduledRetention: %v", err)
	}
	if res.DeletedRows != 2 {
		t.Errorf("DeletedRows = %d, want 2 (old device_graph + 40d network)", res.DeletedRows)
	}
	remaining, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Errorf("remaining cycles = %d, want 2 (recent device_graph + cpu)", len(remaining))
	}
}
