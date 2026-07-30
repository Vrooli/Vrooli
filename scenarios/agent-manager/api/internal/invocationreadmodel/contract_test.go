package invocationreadmodel

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

func TestFactPreservesExplicitUnknownDimensionsAndProjectionProvenance(t *testing.T) {
	occurredAt := time.Date(2026, time.July, 29, 17, 0, 0, 0, time.UTC)
	fact := Fact{
		InvocationFact: runreport.InvocationFact{
			Version:      runreport.InvocationFactVersion,
			Ownership:    "unknown",
			Availability: "unknown",
		},
		RunID:      "run-1",
		OccurredAt: occurredAt,
		TimeBasis:  "run_end",
		RunnerType: "unknown",
		RunStatus:  "completed",
	}

	if fact.Ownership != "unknown" || fact.Availability != "unknown" || fact.RunnerType != "unknown" {
		t.Fatalf("unknown analytical values must remain explicit: %+v", fact)
	}
	if fact.OccurredAt != occurredAt || fact.TimeBasis != "run_end" || fact.Version != runreport.InvocationFactVersion {
		t.Fatalf("projection provenance changed: %+v", fact)
	}
}

func TestProjectUsesCallTimestampAndRunDimensions(t *testing.T) {
	now := time.Date(2026, time.July, 29, 18, 0, 0, 0, time.UTC)
	run := &domain.Run{Status: domain.RunStatusComplete, Tag: "analytics", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-test"}}
	event := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "vrooli help"})
	event.Timestamp = now.Add(-time.Minute)
	event.Sequence = 3
	facts, watermark := Project(run, []*domain.RunEvent{event}, now)
	if len(facts) != 1 || facts[0].OccurredAt != event.Timestamp || facts[0].TimeBasis != "call_event" {
		t.Fatalf("facts=%+v", facts)
	}
	if facts[0].RunnerType != string(domain.RunnerTypeCodex) || facts[0].Model != "gpt-test" || facts[0].Tag != "analytics" {
		t.Fatalf("dimensions=%+v", facts[0])
	}
	if watermark.LastEventID != event.ID.String() || watermark.LastEventAt != event.Timestamp {
		t.Fatalf("watermark=%+v", watermark)
	}
}

func TestProjectRun_UsesTerminalTimingAndAggregatesUsage(t *testing.T) {
	created := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	started := created.Add(time.Minute)
	ended := started.Add(2500 * time.Millisecond)
	run := &domain.Run{ID: uuid.New(), CreatedAt: created, StartedAt: &started, EndedAt: &ended, Status: domain.RunStatusComplete, Tag: "throughput", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "configured"}, ActualModel: "actual"}
	fact := ProjectRun(run, []*domain.RunEvent{{Data: &domain.CostEventData{InputTokens: 10, OutputTokens: 20, CacheCreationTokens: 3, CacheReadTokens: 4, TotalCostUSD: 0.25, CostSource: domain.CostSourceRunnerReported}}, {Data: &domain.CostEventData{InputTokens: 5, TotalCostUSD: 0.75, CostSource: domain.CostSourcePricingTableEstimate}}, {Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "docs/a.md"}}}, {Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "docs/a.md"}}}}, ended.Add(time.Minute))
	if !fact.OccurredAt.Equal(ended) || fact.CostTimeBasis != "ended_at" || fact.DurationMS != 2500 {
		t.Fatalf("terminal timing=%+v", fact)
	}
	if fact.TotalTokens != 42 || fact.TotalCostUSD != 1 || fact.AuthoritativeCostUSD != 0.25 || fact.EstimatedCostUSD != 0.75 || fact.UnknownCostUSD != 0 || fact.ReadCalls != 2 || fact.FileRereads != 1 || fact.Model != "actual" || fact.RunnerType != "codex" {
		t.Fatalf("usage dimensions=%+v", fact)
	}
}

func TestProjectErrorsRetainsOnlyAnalyticalErrorVocabulary(t *testing.T) {
	now := time.Date(2026, time.July, 29, 18, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Tag: "analytics", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-test"}}
	event := domain.NewErrorEvent(run.ID, "runner_failed", "secret detail", false)
	event.Timestamp = now
	facts := ProjectErrors(run, []*domain.RunEvent{event}, now.Add(time.Minute))
	if len(facts) != 1 || facts[0].ErrorCode != "runner_failed" || facts[0].TimeBasis != "error_event" || facts[0].RunID != run.ID.String() || facts[0].Model != "gpt-test" {
		t.Fatalf("error facts=%+v", facts)
	}
}
