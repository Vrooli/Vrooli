package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

type canceledReportRangeRepository struct {
	*memory.MemoryRepository
}

func (r canceledReportRangeRepository) GetEarliestMetricTime(ctx context.Context) (time.Time, error) {
	return time.Time{}, ctx.Err()
}

// failingListsRepo wraps a memory repository but returns errors
// for ListInvestigations and ListAlerts.
type failingListsRepo struct {
	*memory.MemoryRepository
}

func (r *failingListsRepo) ListInvestigations(_ context.Context, _ repository.InvestigationFilter) ([]*models.Investigation, error) {
	return nil, fmt.Errorf("investigations table locked")
}

func (r *failingListsRepo) ListAlerts(_ context.Context, _ repository.AlertFilter) ([]*models.Alert, error) {
	return nil, fmt.Errorf("alerts connection refused")
}

func TestGenerateReport_WarningsOnPartialFailure(t *testing.T) {
	clk := NewStubClock(time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC))
	repo := &failingListsRepo{MemoryRepository: memory.NewRepository()}
	cfg := &config.Config{}

	// Seed at least one metric so the report has data to work with.
	now := clk.Now()
	err := repo.SaveMetricCycle(context.Background(), "report-seed", now, []repository.MetricObservation{{CollectorName: "test", Values: map[string]interface{}{
		"cpu_usage":    50.0,
		"memory_usage": 40.0,
		"timestamp":    now.Format(time.RFC3339),
	}}})
	if err != nil {
		t.Fatalf("seed metrics: %v", err)
	}

	svc := NewReportService(cfg, repo, WithReportClock(clk))

	report, err := svc.GenerateReport(context.Background(), "daily")
	if err != nil {
		t.Fatalf("GenerateReport should succeed with warnings, got error: %v", err)
	}

	if len(report.Warnings) == 0 {
		t.Fatal("expected warnings to be populated when investigations and alerts fail")
	}

	foundInvWarning := false
	foundAlertWarning := false
	for _, w := range report.Warnings {
		if strings.Contains(w, "Investigation data unavailable") {
			foundInvWarning = true
		}
		if strings.Contains(w, "Alert data unavailable") {
			foundAlertWarning = true
		}
	}
	if !foundInvWarning {
		t.Error("expected warning about investigation data being unavailable")
	}
	if !foundAlertWarning {
		t.Error("expected warning about alert data being unavailable")
	}

	// Report should still have valid data despite partial failures.
	if report.ReportID == "" {
		t.Error("expected report to have a valid ID")
	}
	if report.AlertsCount != 0 {
		t.Errorf("expected 0 alerts count with failed fetch, got %d", report.AlertsCount)
	}
	if report.InvestigationsCount != 0 {
		t.Errorf("expected 0 investigations count with failed fetch, got %d", report.InvestigationsCount)
	}
}

func TestGenerateReportPropagatesCanceledRangeLookup(t *testing.T) {
	repo := canceledReportRangeRepository{MemoryRepository: memory.NewRepository()}
	svc := NewReportService(&config.Config{}, repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.GenerateReport(ctx, "daily")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GenerateReport error = %v, want context.Canceled", err)
	}
}

func TestReportStartSelectsWeeklyWindowAndDefaultsToDaily(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name     string
		typeName string
		days     int
	}{
		{name: "daily", typeName: "daily", days: 1},
		{name: "weekly", typeName: "weekly", days: 7},
		{name: "unknown", typeName: "custom", days: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := reportStart(now, tc.typeName), now.AddDate(0, 0, -tc.days); !got.Equal(want) {
				t.Fatalf("reportStart(%q) = %s, want %s", tc.typeName, got, want)
			}
		})
	}
}
