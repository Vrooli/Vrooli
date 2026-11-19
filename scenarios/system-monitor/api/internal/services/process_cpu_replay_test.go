package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

type replayTestCollector struct{ collectors.BaseCollector }

func (replayTestCollector) Collect(context.Context) (*collectors.MetricData, error) { return nil, nil }

func TestReplayProcessCPUAlertReadsHistoricalSamplesAndDoesNotPersistAlert(t *testing.T) {
	start := time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC)
	repo := memory.NewRepository()
	var rows []repository.ProcessSample
	for at := start; !at.After(start.Add(15 * time.Minute)); at = at.Add(20 * time.Second) {
		rows = append(rows, repository.ProcessSample{
			Timestamp: at, PID: 712, Comm: "sd-cli", Owner: "image-tools",
			CPUPct: 509.229, CPUSeconds: 50, RSSKB: 3522953,
		})
	}
	if err := repo.SaveProcessSamples(context.Background(), rows); err != nil {
		t.Fatalf("save historical samples: %v", err)
	}
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: time.Minute}}
	monitor := NewMonitorService(cfg, repo, nil,
		WithProcessSampling(repo, nil, nil),
		WithCollectors(&replayTestCollector{BaseCollector: collectors.NewBaseCollector("replay-test", time.Minute)}),
	)
	defer monitor.cancel()

	got, err := ReplayProcessCPUAlert(context.Background(), monitor, start, start.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("ReplayProcessCPUAlert: %v", err)
	}
	if len(got.Alerts) != 1 {
		t.Fatalf("replay alerts = %+v, want one owner-attributed alert", got.Alerts)
	}
	alert := got.Alerts[0]
	if alert.Details["owner"] != "image-tools" || alert.Details["command"] != "sd-cli" || alert.Details["rule_id"] != RuleSustainedProcessCPU {
		t.Fatalf("replay alert details = %#v, want image-tools/sd-cli identity", alert.Details)
	}
	if got.EvaluatedUntil.Sub(start) > 15*time.Minute || got.EvaluationPasses == 0 || got.RowsAtEvaluation != 1 || got.PossiblyTruncated {
		t.Fatalf("replay bounds/coverage = %+v", got)
	}
	stored, err := repo.ListAlerts(context.Background(), repository.AlertFilter{Type: ConditionAlertType})
	if err != nil {
		t.Fatalf("inspect original alert repository: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("replay wrote %d alerts to production repository, want none", len(stored))
	}
}

func TestReplayProcessCPUAlertUsesOnlyCompletedHistoricalRollups(t *testing.T) {
	start := time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC)
	repo := memory.NewRepository()
	var rows []repository.ProcessSample
	for minute := start; minute.Before(start.Add(15 * time.Minute)); minute = minute.Add(time.Minute) {
		for _, offset := range []time.Duration{10 * time.Second, 30 * time.Second, 50 * time.Second} {
			rows = append(rows, repository.ProcessSample{
				Timestamp: minute.Add(offset), PID: 712, Comm: "sd-cli", Owner: "image-tools",
				CPUPct: 509.229, CPUSeconds: 50, RSSKB: 3522953,
			})
		}
	}
	if err := repo.SaveProcessSamples(context.Background(), rows); err != nil {
		t.Fatalf("save historical samples: %v", err)
	}
	rolled, err := repo.RollupProcessSamples(context.Background(), start, start.Add(15*time.Minute))
	if err != nil || rolled.RawRowsConsumed != int64(len(rows)) {
		t.Fatalf("roll historical samples: result=%+v err=%v", rolled, err)
	}
	cfg := &config.Config{Monitoring: config.MonitoringConfig{MetricsInterval: time.Minute}}
	monitor := NewMonitorService(cfg, repo, nil,
		WithProcessSampling(repo, nil, nil),
		WithCollectors(&replayTestCollector{BaseCollector: collectors.NewBaseCollector("replay-rollup", time.Minute)}),
	)
	defer monitor.cancel()

	got, err := ReplayProcessCPUAlert(context.Background(), monitor, start, start.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("ReplayProcessCPUAlert: %v", err)
	}
	if len(got.Alerts) != 1 || got.Alerts[0].Details["owner"] != "image-tools" || got.Alerts[0].Details["command"] != "sd-cli" {
		t.Fatalf("rollup replay alerts = %+v, want image-tools/sd-cli alert", got.Alerts)
	}
	if got.EvaluatedUntil.Sub(start) > 15*time.Minute || got.SampleCoverage.EndTime.After(got.EvaluatedUntil) {
		t.Fatalf("rollup replay used future or out-of-window evidence: %+v", got)
	}
}

func TestReplayProcessCPUAlertRejectsUnboundedOrFutureWindows(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start time.Time
		end   time.Time
	}{
		{name: "wrong duration", start: time.Now().Add(-time.Hour), end: time.Now()},
		{name: "future", start: time.Now().Add(time.Hour), end: time.Now().Add(time.Hour + 15*time.Minute)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReplayProcessCPUAlert(context.Background(), &MonitorService{}, tc.start, tc.end); err == nil {
				t.Fatal("ReplayProcessCPUAlert succeeded, want bounded-window validation error")
			}
		})
	}
}
