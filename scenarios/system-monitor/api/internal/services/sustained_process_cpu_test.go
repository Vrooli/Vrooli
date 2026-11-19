package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

type processTimelineFixture struct {
	entries []repository.ProcessTimelineEntry
	err     error
	window  time.Duration
	top     int
	rank    string
}

func (f *processTimelineFixture) GetProcessTimelineRanked(_ context.Context, window time.Duration, _ string, top int, rank string) ([]repository.ProcessTimelineEntry, error) {
	f.window, f.top, f.rank = window, top, rank
	return f.entries, f.err
}

func TestSustainedProcessCPURuleReportsOwnerCommandAndBoundedTimeline(t *testing.T) {
	now := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	source := &processTimelineFixture{entries: []repository.ProcessTimelineEntry{{
		Owner: "image-tools", Comm: "sd", PID: 712, CPUPct: 398.4, MaxCPUPct: 425,
		CPUSeconds: 5380, RSSKB: 811000, SampleCount: 39,
		FirstSeen: now.Add(-14 * time.Minute), LastSeen: now.Add(-20 * time.Second),
	}}}
	rule := NewSustainedProcessCPURule(source)
	eval := rule.Evaluate(context.Background(), AlertSnapshot{Now: now})
	if eval.Status != RuleSupported || len(eval.Conditions) != 1 {
		t.Fatalf("evaluation = %+v, want one supported owner condition", eval)
	}
	condition := eval.Conditions[0]
	if condition.Key != "image-tools:sd" || condition.Severity != SeverityCritical || condition.Value != 398.4 {
		t.Fatalf("condition = %+v, want owner+command key and measured CPU", condition)
	}
	if condition.Details["owner"] != "image-tools" || condition.Details["command"] != "sd" || condition.Details["sample_count"] != int64(39) {
		t.Fatalf("condition details = %#v, want owner, command and sample coverage", condition.Details)
	}
	if source.window != 15*time.Minute || source.top != repository.MaxProcessTimelineGroups || source.rank != "cpu_seconds" {
		t.Fatalf("query = (%s, %d, %q), want bounded 15m/%d/cpu_seconds", source.window, source.top, source.rank, repository.MaxProcessTimelineGroups)
	}
}

func TestSustainedProcessCPURuleTreatsSaturatedTimelineAsUnknown(t *testing.T) {
	entries := make([]repository.ProcessTimelineEntry, repository.MaxProcessTimelineGroups)
	rule := NewSustainedProcessCPURule(&processTimelineFixture{entries: entries})
	if got := rule.Evaluate(context.Background(), AlertSnapshot{Now: time.Now()}); got.Status != RuleUnsupported {
		t.Fatalf("saturated timeline status = %q, want unsupported rather than clear", got.Status)
	}
}

func TestSustainedProcessCPURuleRequiresFreshSustainedEvidence(t *testing.T) {
	now := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	base := repository.ProcessTimelineEntry{
		Owner: "typescript-code-graph", Comm: "sidecar", CPUPct: 99.5,
		SampleCount: 40, FirstSeen: now.Add(-14 * time.Minute), LastSeen: now.Add(-20 * time.Second),
	}
	for _, tc := range []struct {
		name   string
		mutate func(*repository.ProcessTimelineEntry)
	}{
		{name: "recently started", mutate: func(row *repository.ProcessTimelineEntry) { row.FirstSeen = now.Add(-7*time.Minute - 59*time.Second) }},
		{name: "stale sample", mutate: func(row *repository.ProcessTimelineEntry) { row.LastSeen = now.Add(-2 * time.Minute) }},
		{name: "insufficient samples", mutate: func(row *repository.ProcessTimelineEntry) { row.SampleCount = 9 }},
		{name: "below one core", mutate: func(row *repository.ProcessTimelineEntry) { row.CPUPct = 89.9 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := base
			tc.mutate(&row)
			rule := NewSustainedProcessCPURule(&processTimelineFixture{entries: []repository.ProcessTimelineEntry{row}})
			if got := rule.Evaluate(context.Background(), AlertSnapshot{Now: now}); got.Status != RuleSupported || len(got.Conditions) != 0 {
				t.Fatalf("evaluation = %+v, want clear supported evidence", got)
			}
		})
	}
}

func TestSustainedProcessCPURuleDoesNotTreatMissingOrFailedEvidenceAsClear(t *testing.T) {
	now := time.Now()
	if got := NewSustainedProcessCPURule(&processTimelineFixture{}).Evaluate(context.Background(), AlertSnapshot{Now: now}); got.Status != RuleUnsupported {
		t.Fatalf("empty timeline status = %q, want unsupported", got.Status)
	}
	if got := NewSustainedProcessCPURule(&processTimelineFixture{err: errors.New("db unavailable")}).Evaluate(context.Background(), AlertSnapshot{Now: now}); got.Status != RuleFailed {
		t.Fatalf("failed timeline status = %q, want failed", got.Status)
	}
	if !defaultSettings.AutoInvestigateCriticalRules[RuleSustainedProcessCPU] {
		t.Fatal("sustained process CPU critical rule must trigger the native investigation path by default")
	}
}
