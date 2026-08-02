package invocationreadmodel

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"github.com/google/uuid"
)

func TestFactPreservesExplicitUnknownDimensionsAndProjectionProvenance(t *testing.T) {
	occurredAt := time.Date(2026, time.July, 29, 17, 0, 0, 0, time.UTC)
	fact := Fact{
		InvocationFact: runsignal.InvocationFact{
			Version:      runsignal.InvocationFactVersion,
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
	if fact.OccurredAt != occurredAt || fact.TimeBasis != "run_end" || fact.Version != runsignal.InvocationFactVersion {
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

func TestProjectDerivesDurableEpisodesAndSelfReportSpansFromSameEvents(t *testing.T) {
	now := time.Date(2026, time.July, 30, 18, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, Tag: "evidence", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-test"}}
	first := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "missing-command"})
	second := domain.NewToolCallEvent(run.ID, "shell", "call-2", map[string]any{"command": "missing-command"})
	message := domain.NewMessageEvent(run.ID, "assistant", "I am blocked until the toolchain is repaired")
	first.Timestamp, first.Sequence = now, 1
	second.Timestamp, second.Sequence = now.Add(time.Minute), 2
	message.Timestamp, message.Sequence = now.Add(2*time.Minute), 3
	events := []*domain.RunEvent{first, second, message}
	facts, watermark := Project(run, events, now.Add(3*time.Minute))
	episodes := ProjectEpisodes(run, facts, events, now.Add(3*time.Minute))
	spans := ProjectSelfReportSpans(run, events, now.Add(3*time.Minute))
	if len(episodes) == 0 || episodes[0].RunID != run.ID.String() || episodes[0].OccurredAt.IsZero() || episodes[0].ClassifierVersion != runsignal.EpisodeClassifierVersion {
		t.Fatalf("episodes=%+v", episodes)
	}
	if len(spans) != 1 || spans[0].EventID != message.ID.String() || spans[0].Text != "I am blocked" || spans[0].ClassifierVersion != runsignal.SelfReportClassifierVersion {
		t.Fatalf("spans=%+v", spans)
	}
	if watermark.EpisodeClassifierVersion != runsignal.EpisodeClassifierVersion || watermark.SelfReportClassifierVersion != runsignal.SelfReportClassifierVersion {
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
