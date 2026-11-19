package services

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

// incidentReplayTimeline models the historical 12 Nov samples as they become
// visible to a rolling 15-minute query. The recorded window had 161 sd-cli
// samples, 509% mean CPU and 3,014.52 CPU seconds over 15 minutes.
type incidentReplayTimeline struct {
	clock      *StubClock
	start      time.Time
	owner      string
	command    string
	cpuPct     float64
	maxCPUPct  float64
	cpuSeconds float64
	rssKB      int64
	samples    int64
}

func (s incidentReplayTimeline) GetProcessTimelineRanked(_ context.Context, _ time.Duration, _ string, _ int, _ string) ([]repository.ProcessTimelineEntry, error) {
	now := s.clock.Now()
	lastSeen := now.Add(-20 * time.Second)
	if lastSeen.Before(s.start) {
		lastSeen = s.start
	}
	elapsed := now.Sub(s.start)
	if elapsed < 0 {
		elapsed = 0
	}
	samples := int64(float64(s.samples) * float64(elapsed) / float64(15*time.Minute))
	if samples > s.samples {
		samples = s.samples
	}
	return []repository.ProcessTimelineEntry{{
		Owner: s.owner, Comm: s.command, CPUPct: s.cpuPct,
		MaxCPUPct: s.maxCPUPct, CPUSeconds: s.cpuSeconds * float64(elapsed) / float64(15*time.Minute),
		RSSKB: s.rssKB, SampleCount: samples, FirstSeen: s.start, LastSeen: lastSeen,
	}}, nil
}

func TestReplayNovember12StableDiffusionIncidentOpensOwnerFlagWithin15Minutes(t *testing.T) {
	for _, profile := range []struct {
		name    string
		owner   string
		command string
		cpuPct  float64
		maxCPU  float64
		seconds float64
		rssKB   int64
		samples int64
	}{
		{name: "four over-admitted Stable Diffusion jobs", owner: "image-tools", command: "sd-cli", cpuPct: 509.229, maxCPU: 1118.539, seconds: 3014.52, rssKB: 3522953, samples: 161},
		{name: "long-running code graph sidecar", owner: "typescript-code-graph", command: "node", cpuPct: 95.506, maxCPU: 105.75, seconds: 133.07, rssKB: 406689, samples: 41},
	} {
		t.Run(profile.name, func(t *testing.T) {
			start := time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC)
			clock := NewStubClock(start)
			repo := memory.NewRepository()
			settings := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()), WithSettingsClock(NewStubClock(start)))
			configured := settings.GetSettings()
			configured.Active = true
			configured.AlertSustainSeconds = 300
			if err := settings.UpdateSettings(configured); err != nil {
				t.Fatalf("configure evaluator: %v", err)
			}
			snapshots := &fakeSnapshotSource{metrics: &models.MetricsResponse{Timestamp: start}}
			alerts := NewAlertService(nil, repo, WithAlertClock(clock))
			timeline := incidentReplayTimeline{
				clock: clock, start: start, owner: profile.owner, command: profile.command,
				cpuPct: profile.cpuPct, maxCPUPct: profile.maxCPU, cpuSeconds: profile.seconds,
				rssKB: profile.rssKB, samples: profile.samples,
			}
			evaluator := NewAlertEvaluator(settings, snapshots, repo, alerts, nil,
				WithAlertEvaluatorClock(clock), WithAlertRules(NewSustainedProcessCPURule(timeline)))

			var openedAt time.Time
			for elapsed := time.Duration(0); elapsed <= 15*time.Minute; elapsed += 25 * time.Second {
				clock.Set(start.Add(elapsed))
				snapshots.setCPU(25, clock.Now())
				evaluator.RunOnce(context.Background())
				active, err := repo.ListAlerts(context.Background(), repository.AlertFilter{Type: ConditionAlertType, Status: "active"})
				if err != nil {
					t.Fatalf("read replay alerts: %v", err)
				}
				for _, alert := range active {
					if alert.Details["rule_id"] == RuleSustainedProcessCPU && alert.Details["owner"] == profile.owner && alert.Details["command"] == profile.command {
						openedAt = alert.Timestamp
						break
					}
				}
				if !openedAt.IsZero() {
					break
				}
			}
			if openedAt.IsZero() {
				t.Fatalf("12 Nov %s did not open owner-attributed alert within 15 minutes", profile.name)
			}
			delay := openedAt.Sub(start)
			if delay > 15*time.Minute {
				t.Fatalf("incident flag delay = %s, want <= 15m", delay)
			}
			if delay < 13*time.Minute {
				t.Fatalf("incident flag delay = %s, want a full 8m signal age plus 5m evaluator sustain", delay)
			}
		})
	}
}
