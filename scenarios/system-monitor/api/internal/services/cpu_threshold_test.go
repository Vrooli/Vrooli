package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

type fakeCPUObservationSource struct{ observation models.CPUMetrics }

func (f *fakeCPUObservationSource) ReadCPUObservation(context.Context) (models.CPUMetrics, error) {
	return f.observation, nil
}

func TestCPUThresholdRequiresSustainedMeasuredWindow(t *testing.T) {
	settings := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()))
	next := settings.GetSettings()
	next.Active = true
	next.CPUSustainedWindowTicks = 3
	next.CPUEscalationDebounceTicks = 1
	if err := settings.UpdateSettings(next); err != nil {
		t.Fatal(err)
	}
	repo := memory.NewRepository()
	alerts := NewAlertService(&config.Config{}, repo)
	source := &fakeCPUObservationSource{observation: models.CPUMetrics{UsageState: models.MetricState{Status: "measured", Value: 99, Provenance: "fixture"}}}
	scheduler := NewThresholdScheduler(settings, alerts, repo, nil, WithCPUObservationSource(source), WithThresholdClock(NewStubClock(time.Unix(0, 0))))
	for i := 0; i < 2; i++ {
		scheduler.evaluateCPU(context.Background(), next)
	}
	violations, err := repo.GetThresholdViolations(context.Background(), repository.TimeRange{StartTime: time.Unix(-1, 0), EndTime: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("short CPU spike produced %d violations", len(violations))
	}
	scheduler.evaluateCPU(context.Background(), next)
	violations, err = repo.GetThresholdViolations(context.Background(), repository.TimeRange{StartTime: time.Unix(-1, 0), EndTime: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || violations[0].MetricName != "cpu_usage" {
		t.Fatalf("sustained CPU load violations = %#v", violations)
	}
}

func TestCPUThresholdSkipsUnmeasuredWithoutClearingWindow(t *testing.T) {
	settings := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()))
	next := settings.GetSettings()
	next.Active = true
	next.CPUSustainedWindowTicks = 2
	next.CPUEscalationDebounceTicks = 1
	_ = settings.UpdateSettings(next)
	repo := memory.NewRepository()
	alerts := NewAlertService(&config.Config{}, repo)
	source := &fakeCPUObservationSource{}
	scheduler := NewThresholdScheduler(settings, alerts, repo, nil, WithCPUObservationSource(source), WithThresholdClock(NewStubClock(time.Unix(0, 0))))
	source.observation = models.CPUMetrics{UsageState: models.MetricState{Status: "measured", Value: 95}}
	scheduler.evaluateCPU(context.Background(), next)
	source.observation = models.CPUMetrics{UsageState: models.MetricState{Status: "unsupported", Reason: "backend unavailable"}}
	scheduler.evaluateCPU(context.Background(), next)
	source.observation = models.CPUMetrics{UsageState: models.MetricState{Status: "measured", Value: 95}}
	scheduler.evaluateCPU(context.Background(), next)
	violations, err := repo.GetThresholdViolations(context.Background(), repository.TimeRange{StartTime: time.Unix(-1, 0), EndTime: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 {
		t.Fatalf("unmeasured gap reset sustained CPU window: %#v", violations)
	}
}
