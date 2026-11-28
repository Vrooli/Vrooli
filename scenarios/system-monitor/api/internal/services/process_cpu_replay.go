package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

const (
	processCPUReplayWindow   = 15 * time.Minute
	processCPUReplayCadence  = 25 * time.Second
	processCPUReplayTopLimit = sustainedProcessCPUTop
)

var ErrInvalidProcessCPUReplayWindow = errors.New("invalid process CPU replay window")

// ProcessCPUReplayResult is a read-only, in-memory replay of the owner CPU
// alert evaluator over one historical 15-minute interval.
type ProcessCPUReplayResult struct {
	WindowStart       time.Time            `json:"window_start"`
	WindowEnd         time.Time            `json:"window_end"`
	EvaluatedUntil    time.Time            `json:"evaluated_until"`
	EvaluationPasses  int                  `json:"evaluation_passes"`
	RowsAtEvaluation  int                  `json:"rows_at_evaluation"`
	TopLimit          int                  `json:"top_limit"`
	PossiblyTruncated bool                 `json:"possibly_truncated"`
	SampleCoverage    repository.TimeRange `json:"sample_coverage"`
	Alerts            []*models.Alert      `json:"alerts"`
}

type processCPUReplaySource struct {
	monitor     processTimelineRangeReader
	clock       Clock
	lastEntries []repository.ProcessTimelineEntry
	lastErr     error
}

type processTimelineRangeReader interface {
	GetProcessTimelineInRange(context.Context, time.Time, time.Time, string, int, string) ([]repository.ProcessTimelineEntry, error)
}

func (s *processCPUReplaySource) GetCurrentMetricsFresh(context.Context) (*models.MetricsResponse, error) {
	return nil, nil
}

func (s *processCPUReplaySource) GetMemoryPressure(context.Context) (*models.MemoryPressure, error) {
	return nil, nil
}

func (s *processCPUReplaySource) GetProcessTimelineRanked(ctx context.Context, window time.Duration, owner string, top int, rank string) ([]repository.ProcessTimelineEntry, error) {
	now := s.clock.Now()
	// The repository includes only rollups fully contained in this interval,
	// while retaining raw samples through now. This preserves exact sample
	// freshness without leaking a later observation from the current rollup
	// minute into the historical replay.
	start := now.UTC().Add(-window)
	end := now.UTC()
	s.lastEntries, s.lastErr = s.monitor.GetProcessTimelineInRange(ctx, start, end, owner, top, rank)
	return s.lastEntries, s.lastErr
}

// ReplayProcessCPUAlert re-evaluates retained historical process samples with
// the actual rule and evaluator through the full interval. Alerts live only in
// a new in-memory repository, so the source store receives no writes. The
// replay sends no notifications and does not call escalation or remediation.
func ReplayProcessCPUAlert(ctx context.Context, monitor processTimelineRangeReader, start, end time.Time) (*ProcessCPUReplayResult, error) {
	if monitor == nil {
		return nil, errors.New("monitor service is required")
	}
	if start.IsZero() || end.IsZero() || !end.Equal(start.Add(processCPUReplayWindow)) {
		return nil, fmt.Errorf("%w: interval must be exactly %s", ErrInvalidProcessCPUReplayWindow, processCPUReplayWindow)
	}
	if end.After(time.Now()) {
		return nil, fmt.Errorf("%w: interval must end in the past", ErrInvalidProcessCPUReplayWindow)
	}

	clock := NewStubClock(start.UTC())
	alertsRepo := memory.NewRepository()
	settings := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()), WithSettingsClock(clock))
	next := settings.GetSettings()
	next.Active = true
	next.AlertSustainSeconds = 300
	if err := settings.UpdateSettings(next); err != nil {
		return nil, fmt.Errorf("configure replay evaluator: %w", err)
	}
	alertService := NewAlertService(nil, alertsRepo, WithAlertClock(clock))
	source := &processCPUReplaySource{monitor: monitor, clock: clock}
	evaluator := NewAlertEvaluator(settings, source, alertsRepo, alertService, nil,
		WithAlertEvaluatorClock(clock), WithAlertRules(NewSustainedProcessCPURule(source)))
	result := &ProcessCPUReplayResult{
		WindowStart: start.UTC(), WindowEnd: end.UTC(), TopLimit: processCPUReplayTopLimit,
	}

	for elapsed := time.Duration(0); elapsed <= processCPUReplayWindow; elapsed += processCPUReplayCadence {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		clock.Set(start.Add(elapsed).UTC())
		evaluator.RunOnce(ctx)
		result.EvaluationPasses++
		result.EvaluatedUntil = clock.Now()
		if source.lastErr != nil {
			return nil, fmt.Errorf("read historical process timeline: %w", source.lastErr)
		}
		entries := source.lastEntries
		result.RowsAtEvaluation = len(entries)
		result.PossiblyTruncated = len(entries) == processCPUReplayTopLimit
		result.SampleCoverage = repository.TimeRange{}
		for _, entry := range entries {
			if result.SampleCoverage.StartTime.IsZero() || entry.FirstSeen.Before(result.SampleCoverage.StartTime) {
				result.SampleCoverage.StartTime = entry.FirstSeen
			}
			if entry.LastSeen.After(result.SampleCoverage.EndTime) {
				result.SampleCoverage.EndTime = entry.LastSeen
			}
		}
	}
	alerts, resultErr := alertsRepo.ListAlerts(ctx, repository.AlertFilter{Type: ConditionAlertType})
	if resultErr != nil {
		return nil, fmt.Errorf("read replay alert state: %w", resultErr)
	}
	result.Alerts = alerts
	if result.Alerts == nil {
		result.Alerts = []*models.Alert{}
	}
	return result, nil
}

var _ AlertSnapshotSource = (*processCPUReplaySource)(nil)
var _ ProcessTimelineSource = (*processCPUReplaySource)(nil)
