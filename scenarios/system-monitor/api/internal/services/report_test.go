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

func TestOwnerResourceReportAggregatesCPUAndMemoryByOwner(t *testing.T) {
	repo := memory.NewRepository()
	start := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	if err := repo.SaveProcessSamples(context.Background(), []repository.ProcessSample{
		{Timestamp: start, Owner: "scenario-a", Comm: "worker", PID: 10, CPUSeconds: 900, RSSKB: 1024 * 1024, SwapKB: 512 * 1024, SwapStatus: "measured", GPUVRAMMB: 2048},
		{Timestamp: start.Add(time.Minute), Owner: "scenario-a", Comm: "worker", PID: 10, CPUSeconds: 900, RSSKB: 3 * 1024 * 1024, SwapKB: 1536 * 1024, SwapStatus: "measured", GPUVRAMMB: 1024},
		{Timestamp: start, Owner: "unknown", Comm: "mystery", PID: 11, CPUSeconds: 360, RSSKB: 1024, SwapKB: 2048, SwapStatus: "measured"},
	}); err != nil {
		t.Fatalf("save samples: %v", err)
	}

	service := NewReportService(&config.Config{}, repo)
	got, err := service.ownerResourceReport(context.Background(), repository.TimeRange{
		StartTime: start.Add(-time.Second),
		EndTime:   start.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("ownerResourceReport: %v", err)
	}
	if !got.Complete || got.AttributionComplete || !got.SwapCoverageComplete || len(got.Owners) != 2 {
		t.Fatalf("report completeness/owners = %+v", got)
	}
	owner := got.Owners[0]
	if owner.Owner != "scenario-a" || owner.ProcessGroups != 1 || owner.Samples != 2 {
		t.Fatalf("scenario-a grouping = %+v", owner)
	}
	if owner.CoreHours != .5 || owner.MeanRSSGiBProcessSum != 2 || owner.MeanSwapGiBProcessSum != 1 || owner.PeakSingleProcessVRAMGiB != 2 {
		t.Fatalf("scenario-a resource summary = %+v", owner)
	}
	if owner.SwapMeasuredSamples != 2 || owner.SwapUnknownSamples != 0 || got.SwapMeasuredSamples != 3 || got.SwapUnknownSamples != 0 {
		t.Fatalf("swap coverage = report:%+v owner:%+v", got, owner)
	}
}

func TestOwnerResourceReportMarksLegacySwapSamplesUnknown(t *testing.T) {
	repo := memory.NewRepository()
	start := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	if err := repo.SaveProcessSamples(context.Background(), []repository.ProcessSample{
		{Timestamp: start, Owner: "scenario-a", Comm: "worker", PID: 10, CPUSeconds: 10, RSSKB: 1024, SwapKB: 0},
		{Timestamp: start.Add(time.Minute), Owner: "scenario-a", Comm: "worker", PID: 10, CPUSeconds: 10, RSSKB: 1024, SwapKB: 512, SwapStatus: "measured"},
	}); err != nil {
		t.Fatalf("save samples: %v", err)
	}

	service := NewReportService(&config.Config{}, repo)
	got, err := service.ownerResourceReport(context.Background(), repository.TimeRange{
		StartTime: start.Add(-time.Second),
		EndTime:   start.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("ownerResourceReport: %v", err)
	}
	if got.SwapCoverageComplete || got.SwapMeasuredSamples != 1 || got.SwapUnknownSamples != 1 || len(got.Owners) != 1 {
		t.Fatalf("legacy swap coverage should be explicit and partial: %+v", got)
	}
	if got.Owners[0].SwapMeasuredSamples != 1 || got.Owners[0].SwapUnknownSamples != 1 {
		t.Fatalf("owner swap coverage should preserve measured and unknown counts: %+v", got.Owners[0])
	}
}

func TestOwnerResourceReportMarksMissingWindowEdgesIncomplete(t *testing.T) {
	repo := memory.NewRepository()
	start := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	if err := repo.SaveProcessSamples(context.Background(), []repository.ProcessSample{
		{Timestamp: start.Add(5 * time.Minute), Owner: "scenario-a", Comm: "worker", PID: 10, CPUSeconds: 100, RSSKB: 1024},
		{Timestamp: start.Add(55 * time.Minute), Owner: "scenario-a", Comm: "worker", PID: 10, CPUSeconds: 200, RSSKB: 2048},
	}); err != nil {
		t.Fatalf("save samples: %v", err)
	}

	service := NewReportService(&config.Config{}, repo)
	got, err := service.ownerResourceReport(context.Background(), repository.TimeRange{
		StartTime: start,
		EndTime:   start.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("ownerResourceReport: %v", err)
	}
	if got.Complete {
		t.Fatalf("report completeness = true, want incomplete when the query begins and ends outside the 2m sample coverage tolerance: %+v", got)
	}
	if !got.SampleCoverageStart.Equal(start.Add(5*time.Minute)) || !got.SampleCoverageEnd.Equal(start.Add(55*time.Minute)) {
		t.Fatalf("sample coverage = [%s, %s], want [%s, %s]", got.SampleCoverageStart, got.SampleCoverageEnd, start.Add(5*time.Minute), start.Add(55*time.Minute))
	}
}

func TestOwnerResourceFollowUpsMakeUnknownTailActionableWithoutClaimingHostTotals(t *testing.T) {
	actions := ownerResourceFollowUps(models.OwnerResourceReport{
		SwapMeasuredSamples: 108,
		SwapUnknownSamples:  2_855_367,
		Owners: []models.OwnerResourceSummary{
			{Owner: "unknown", ProcessGroups: 1678},
			{Owner: "web-console", ProcessGroups: 4},
		},
	})
	if len(actions) != 2 {
		t.Fatalf("follow-up count = %d, want swap and attribution actions: %v", len(actions), actions)
	}
	for _, want := range []string{
		"108 measured samples, 2855367 unknown",
		"--rank swap --top 100",
		"1678 unknown process groups",
		"Historical samples keep the owner label captured at collection time",
		"repair the resolver only if current user, app, service, or agent processes still appear unknown",
		"do not assign their bytes to a scenario by inference",
	} {
		found := false
		for _, action := range actions {
			if strings.Contains(action, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("follow-ups missing %q: %v", want, actions)
		}
	}
}

func TestOwnerResourceFollowUpsOmitFullyAccountedReport(t *testing.T) {
	if actions := ownerResourceFollowUps(models.OwnerResourceReport{SwapCoverageComplete: true, AttributionComplete: true}); len(actions) != 0 {
		t.Fatalf("complete report follow-ups = %v, want none", actions)
	}
}
