package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

type gpuOnCPUProcessSource struct {
	entries []repository.ProcessTimelineEntry
	err     error
	window  time.Duration
	owner   string
	top     int
	rank    string
}

func (s *gpuOnCPUProcessSource) GetProcessTimelineRanked(_ context.Context, window time.Duration, owner string, top int, rank string) ([]repository.ProcessTimelineEntry, error) {
	s.window, s.owner, s.top, s.rank = window, owner, top, rank
	return s.entries, s.err
}

func measuredGPUSnapshot(now time.Time) AlertSnapshot {
	usage := 17.0
	return AlertSnapshot{
		Now: now,
		Metrics: &models.MetricsResponse{
			GPUUsage:          &usage,
			GPUState:          models.MetricState{Status: "measured", Value: usage},
			GPUProcessesState: models.MetricState{Status: "measured", ObservedAt: now},
		},
	}
}

func TestGPUOnCPURuleFindsOwnerAttributedStableDiffusionCPUWork(t *testing.T) {
	now := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	source := &gpuOnCPUProcessSource{entries: []repository.ProcessTimelineEntry{{
		Owner: "image-tools", Comm: "sd-cli", PID: 712, CPUPct: 398.4, MaxCPUPct: 425,
		CPUSeconds: 5380, RSSKB: 811000, SampleCount: 39,
		FirstSeen: now.Add(-14 * time.Minute), LastSeen: now.Add(-20 * time.Second),
	}}}
	eval := NewGPUOnCPURule(source).Evaluate(context.Background(), measuredGPUSnapshot(now))
	if eval.Status != RuleSupported || len(eval.Conditions) != 1 {
		t.Fatalf("evaluation = %+v, want one supported owner condition", eval)
	}
	condition := eval.Conditions[0]
	if condition.Key != "image-tools:sd-cli" || condition.Severity != SeverityCritical || condition.Value != 398.4 || condition.MetricName != RuleGPUCapableWorkOnCPU {
		t.Fatalf("condition = %+v, want critical image-tools:sd-cli placement alert", condition)
	}
	if condition.Details["owner"] != "image-tools" || condition.Details["command"] != "sd-cli" || condition.Details["sample_count"] != int64(39) {
		t.Fatalf("condition details = %#v, want owner, command and sample coverage", condition.Details)
	}
	if source.window != gpuOnCPUWindow || source.owner != gpuOnCPUOwner || source.top != repository.MaxProcessTimelineGroups || source.rank != "cpu_seconds" {
		t.Fatalf("query = (%s, %q, %d, %q), want bounded 15m/owner/%d/cpu_seconds", source.window, source.owner, source.top, source.rank, repository.MaxProcessTimelineGroups)
	}
}

func TestGPUOnCPURuleTreatsSaturatedTimelineAsUnknown(t *testing.T) {
	entries := make([]repository.ProcessTimelineEntry, repository.MaxProcessTimelineGroups)
	rule := NewGPUOnCPURule(&gpuOnCPUProcessSource{entries: entries})
	if got := rule.Evaluate(context.Background(), measuredGPUSnapshot(time.Now().UTC())); got.Status != RuleUnsupported {
		t.Fatalf("saturated timeline status = %q, want unsupported rather than clear", got.Status)
	}
}

func TestGPUOnCPURuleRequiresSustainedFreshCPUWorkWithNoGPUVRAM(t *testing.T) {
	now := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	base := repository.ProcessTimelineEntry{
		Owner: "image-tools", Comm: "sd-cli", CPUPct: 398.4,
		SampleCount: 39, FirstSeen: now.Add(-14 * time.Minute), LastSeen: now.Add(-20 * time.Second),
	}
	for _, tc := range []struct {
		name   string
		mutate func(*repository.ProcessTimelineEntry)
	}{
		{name: "recent process", mutate: func(row *repository.ProcessTimelineEntry) { row.FirstSeen = now.Add(-30 * time.Second) }},
		{name: "stale sample", mutate: func(row *repository.ProcessTimelineEntry) { row.LastSeen = now.Add(-2 * time.Minute) }},
		{name: "few samples", mutate: func(row *repository.ProcessTimelineEntry) { row.SampleCount = 2 }},
		{name: "low CPU", mutate: func(row *repository.ProcessTimelineEntry) { row.CPUPct = 24.9 }},
		{name: "observed GPU use", mutate: func(row *repository.ProcessTimelineEntry) { row.GPUVRAMMB = 128 }},
		{name: "unregistered process", mutate: func(row *repository.ProcessTimelineEntry) { row.Comm = "python" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := base
			tc.mutate(&row)
			rule := NewGPUOnCPURule(&gpuOnCPUProcessSource{entries: []repository.ProcessTimelineEntry{row}})
			if got := rule.Evaluate(context.Background(), measuredGPUSnapshot(now)); got.Status != RuleSupported || len(got.Conditions) != 0 {
				t.Fatalf("evaluation = %+v, want clear supported evidence", got)
			}
		})
	}
}

func TestGPUOnCPURuleDoesNotInventGPUOrProcessEvidence(t *testing.T) {
	now := time.Now().UTC()
	entry := repository.ProcessTimelineEntry{
		Owner: "image-tools", Comm: "sd-cli", CPUPct: 400, SampleCount: 20,
		FirstSeen: now.Add(-5 * time.Minute), LastSeen: now.Add(-10 * time.Second),
	}
	rule := NewGPUOnCPURule(&gpuOnCPUProcessSource{entries: []repository.ProcessTimelineEntry{entry}})
	withoutGPU := AlertSnapshot{Now: now, Metrics: &models.MetricsResponse{GPUState: models.MetricState{Status: "unsupported", Reason: "no GPU"}}}
	if got := rule.Evaluate(context.Background(), withoutGPU); got.Status != RuleUnsupported {
		t.Fatalf("unsupported GPU status = %q, want unsupported", got.Status)
	}
	missingProcessInventory := measuredGPUSnapshot(now)
	missingProcessInventory.Metrics.GPUProcessesState = models.MetricState{Status: "failed", Reason: "probe failed"}
	if got := rule.Evaluate(context.Background(), missingProcessInventory); got.Status != RuleUnsupported {
		t.Fatalf("failed GPU process inventory status = %q, want unsupported", got.Status)
	}
	staleProcessInventory := measuredGPUSnapshot(now)
	staleProcessInventory.Metrics.GPUProcessesState.ObservedAt = now.Add(-2 * time.Minute)
	if got := rule.Evaluate(context.Background(), staleProcessInventory); got.Status != RuleUnsupported {
		t.Fatalf("stale GPU process inventory status = %q, want unsupported", got.Status)
	}
	if got := NewGPUOnCPURule(nil).Evaluate(context.Background(), measuredGPUSnapshot(now)); got.Status != RuleUnsupported {
		t.Fatalf("missing timeline status = %q, want unsupported", got.Status)
	}
	if got := NewGPUOnCPURule(&gpuOnCPUProcessSource{}).Evaluate(context.Background(), measuredGPUSnapshot(now)); got.Status != RuleUnsupported {
		t.Fatalf("empty timeline status = %q, want unsupported", got.Status)
	}
	if got := NewGPUOnCPURule(&gpuOnCPUProcessSource{err: errors.New("sqlite unavailable")}).Evaluate(context.Background(), measuredGPUSnapshot(now)); got.Status != RuleFailed {
		t.Fatalf("failed timeline status = %q, want failed", got.Status)
	}
}
