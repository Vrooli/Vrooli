package invocationreadmodel

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
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

func TestProjectPersistsTokenFactorsAndResidency(t *testing.T) {
	now := time.Date(2026, time.August, 5, 19, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "vrooli help"})
	result := domain.NewToolResultEvent(run.ID, "shell", "call-1", "command output", nil)
	firstTurn := domain.NewMessageEvent(run.ID, "assistant", "first turn")
	secondTurn := domain.NewMessageEvent(run.ID, "assistant", "second turn")
	for index, event := range []*domain.RunEvent{call, result, firstTurn, secondTurn} {
		event.Sequence = int64(index + 1)
		event.Timestamp = now.Add(time.Duration(index) * time.Second)
	}
	events := []*domain.RunEvent{call, result, firstTurn, secondTurn}
	facts, _ := Project(run, events, now)
	if len(facts) != 1 {
		t.Fatalf("facts=%+v", facts)
	}
	if facts[0].ArgTokens == 0 || facts[0].ResultTokens == 0 || facts[0].TokenBasis != "estimated" || facts[0].ResidencyTurns != 2 || facts[0].ResidencySegment != 0 {
		t.Fatalf("token factors=%+v", facts[0].InvocationFact)
	}
	second, _ := Project(run, events, now)
	if !reflect.DeepEqual(facts, second) {
		t.Fatal("projection is not idempotent")
	}
}

func TestProjectMarksUnpairedTokenBasisUnknown(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "shell", "missing-result", map[string]any{"command": "echo hi"})
	facts, _ := Project(run, []*domain.RunEvent{call}, time.Now())
	if len(facts) != 1 || facts[0].ArgTokens == 0 || facts[0].TokenBasis != "unknown" || facts[0].ResultTokens != 0 {
		t.Fatalf("unpaired factors=%+v", facts)
	}
}

func TestProjectUsesProjectionTimeForEmptyEventSet(t *testing.T) {
	now := time.Date(2026, time.August, 4, 19, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	_, watermark := Project(run, nil, now)
	if watermark.LastEventID != "" || !watermark.LastEventAt.Equal(now) {
		t.Fatalf("empty projection watermark=%+v", watermark)
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
	amountA, amountB := int64(250000), int64(750000)
	fact := ProjectRun(run, []*domain.RunEvent{{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 20, CacheCreationTokens: 3, CacheReadTokens: 4}}, {Data: &domain.UsageEventData{InputTokens: 5}}, {Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &amountA}}, {Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &amountB}}, {Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "docs/a.md"}}}, {Data: &domain.ToolCallEventData{ToolName: "read_file", Input: map[string]any{"path": "docs/a.md"}}}}, ended.Add(time.Minute))
	if !fact.OccurredAt.Equal(ended) || fact.CostTimeBasis != "ended_at" || fact.DurationMS != 2500 {
		t.Fatalf("terminal timing=%+v", fact)
	}
	if fact.TotalTokens != 42 || fact.TotalCostUSD != 1 || fact.MeteredChargeMicroUSD != 1000000 || fact.ReadCalls != 2 || fact.FileRereads != 1 || fact.Model != "actual" || fact.RunnerType != "codex" {
		t.Fatalf("usage dimensions=%+v", fact)
	}
}

func TestProjectRunUsesTerminalAuthorityWithoutDoubleCountingPerTurnUsage(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	fact := ProjectRun(run, []*domain.RunEvent{
		{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 2, TurnIndex: 1}},
		{Data: &domain.UsageEventData{InputTokens: 20, OutputTokens: 3, TurnIndex: 2}},
		{Data: &domain.UsageEventData{InputTokens: 30, OutputTokens: 4, TurnIndex: 3, ReconciliationAuthority: true}},
	}, time.Now().UTC())
	if fact.TotalTokens != 34 || fact.InputTokens != 30 || fact.OutputTokens != 4 {
		t.Fatalf("terminal-authority usage=%+v, want only 30 input + 4 output", fact)
	}
}

func TestProjectAttributesPerTurnUsageAndConservesAgainstTerminalTotal(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete, ResolvedConfig: &domain.RunConfig{
		PreambleInjectedTokens: 10,
		PreambleTokenBasis:     tokenaccounting.BasisEstimated,
	}}
	callOne := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "one"})
	callTwo := domain.NewToolCallEvent(run.ID, "shell", "call-2", map[string]any{"command": "two"})
	events := []*domain.RunEvent{
		callOne,
		{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 10, TurnIndex: 1}},
		callTwo,
		{Data: &domain.UsageEventData{InputTokens: 120, OutputTokens: 20, CacheReadTokens: 5, CacheCreationTokens: 2, TurnIndex: 2}},
		{Data: &domain.UsageEventData{InputTokens: 220, OutputTokens: 30, CacheReadTokens: 5, CacheCreationTokens: 2, TurnIndex: 2, ReconciliationAuthority: true}},
	}
	facts, _ := Project(run, events, time.Now().UTC())
	if len(facts) != 2 {
		t.Fatalf("facts=%d, want two", len(facts))
	}
	if facts[0].IncurredInputTokens != 0 || facts[0].IncurredOutputTokens != 10 {
		t.Fatalf("first incurred=%+v", facts[0].InvocationFact)
	}
	if facts[1].IncurredInputTokens != 120 || facts[1].IncurredOutputTokens != 20 || facts[1].IncurredCacheReadTokens != 5 || facts[1].IncurredCacheCreationTokens != 2 {
		t.Fatalf("second incurred=%+v", facts[1].InvocationFact)
	}
	runFact := ProjectRun(run, events, time.Now().UTC())
	var incurred int64
	for _, fact := range facts {
		incurred += fact.IncurredInputTokens + fact.IncurredOutputTokens + fact.IncurredCacheReadTokens + fact.IncurredCacheCreationTokens
	}
	if runFact.PreambleInjectedTokens+runFact.PreambleFixedTokens+incurred+runFact.UnattributedTokens != runFact.TotalTokens {
		t.Fatalf("accounting does not conserve: preamble=%d+%d incurred=%d residual=%d total=%d", runFact.PreambleInjectedTokens, runFact.PreambleFixedTokens, incurred, runFact.UnattributedTokens, runFact.TotalTokens)
	}
	if runFact.UnattributedTokens != 0 || runFact.UnattributedReason != "" {
		t.Fatalf("unexpected residual=%+v", runFact)
	}
}

func TestProjectRunTerminalOnlyUsageIsUnattributedWithReason(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	fact := ProjectRun(run, []*domain.RunEvent{{Data: &domain.UsageEventData{InputTokens: 30, OutputTokens: 5, ReconciliationAuthority: true}}}, time.Now().UTC())
	if fact.TotalTokens != 35 || fact.UnattributedTokens != 35 || fact.UnattributedReason == "" {
		t.Fatalf("terminal-only attribution=%+v", fact)
	}
}

func TestProjectRunConservesAcrossFixtureClasses(t *testing.T) {
	newRun := func() *domain.Run {
		return &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete, ResolvedConfig: &domain.RunConfig{
			PreambleInjectedTokens: 10,
			PreambleTokenBasis:     tokenaccounting.BasisEstimated,
		}}
	}
	fixtures := []struct {
		name   string
		events []*domain.RunEvent
	}{
		{name: "no-usage", events: nil},
		{name: "terminal-only", events: []*domain.RunEvent{{Data: &domain.UsageEventData{InputTokens: 30, OutputTokens: 5, ReconciliationAuthority: true}}}},
		{name: "per-turn", events: func() []*domain.RunEvent {
			run := newRun()
			return []*domain.RunEvent{
				domain.NewToolCallEvent(run.ID, "shell", "fixture-call", map[string]any{"command": "echo"}),
				{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 5, TurnIndex: 1}},
				{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 5, TurnIndex: 1, ReconciliationAuthority: true}},
			}
		}()},
		{name: "two-compactions", events: func() []*domain.RunEvent {
			run := newRun()
			return []*domain.RunEvent{
				domain.NewToolCallEvent(run.ID, "shell", "fixture-call-1", map[string]any{"command": "one"}),
				{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 5, TurnIndex: 1}},
				domain.NewCompactionEvent(run.ID, "one", "auto", "", 2, 100, 50, ""),
				domain.NewToolCallEvent(run.ID, "shell", "fixture-call-2", map[string]any{"command": "two"}),
				{Data: &domain.UsageEventData{InputTokens: 50, OutputTokens: 4, TurnIndex: 2}},
				domain.NewCompactionEvent(run.ID, "two", "auto", "", 2, 50, 25, ""),
				domain.NewToolCallEvent(run.ID, "shell", "fixture-call-3", map[string]any{"command": "three"}),
				{Data: &domain.UsageEventData{InputTokens: 25, OutputTokens: 3, TurnIndex: 3}},
				{Data: &domain.UsageEventData{InputTokens: 175, OutputTokens: 12, TurnIndex: 3, ReconciliationAuthority: true}},
			}
		}()},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			run := newRun()
			fact := ProjectRun(run, fixture.events, time.Now().UTC())
			facts, _ := Project(run, fixture.events, time.Now().UTC())
			var incurred int64
			for _, item := range facts {
				incurred += item.IncurredInputTokens + item.IncurredOutputTokens + item.IncurredCacheReadTokens + item.IncurredCacheCreationTokens
			}
			if fact.PreambleInjectedTokens+fact.PreambleFixedTokens+incurred+fact.UnattributedTokens != fact.TotalTokens {
				t.Fatalf("non-conserving buckets: preamble=%d+%d incurred=%d residual=%d total=%d", fact.PreambleInjectedTokens, fact.PreambleFixedTokens, incurred, fact.UnattributedTokens, fact.TotalTokens)
			}
			if (fact.UnattributedTokens != 0 || fixture.name == "no-usage") && fact.UnattributedReason == "" {
				t.Fatalf("residual=%d has no reason", fact.UnattributedTokens)
			}
		})
	}
}

func TestProjectCompactionResidencyIsPiecewiseAndNonZero(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "Read", "call-1", map[string]any{"path": "a.go"})
	result := domain.NewToolResultEvent(run.ID, "Read", "call-1", "result", nil)
	callAfterCompaction := domain.NewToolCallEvent(run.ID, "Read", "call-2", map[string]any{"path": "b.go"})
	resultAfterCompaction := domain.NewToolResultEvent(run.ID, "Read", "call-2", "result", nil)
	events := []*domain.RunEvent{
		call,
		result,
		{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 5, TurnIndex: 1}},
		domain.NewCompactionEvent(run.ID, "first", "manual", "", 3, 100, 50, "/compact"),
		{Data: &domain.UsageEventData{InputTokens: 50, OutputTokens: 4, TurnIndex: 2}},
		callAfterCompaction,
		resultAfterCompaction,
		domain.NewCompactionEvent(run.ID, "second", "manual", "", 2, 50, 25, "/compact"),
		{Data: &domain.UsageEventData{InputTokens: 25, OutputTokens: 3, TurnIndex: 3}},
	}
	facts, _ := Project(run, events, time.Now().UTC())
	if len(facts) != 2 {
		t.Fatalf("facts=%d, want two", len(facts))
	}
	if facts[0].ResidencySegment != 0 || facts[0].ResidencyTurns <= 0 || facts[0].ResidencyTurns >= 3 {
		t.Fatalf("pre-boundary fact=%+v, want nonzero attenuated residency in segment 0", facts[0])
	}
	if facts[1].ResidencySegment != 1 || facts[1].ResidencyTurns <= 0 {
		t.Fatalf("post-boundary fact=%+v, want nonzero segment 1 residency", facts[1])
	}
}

func TestProjectRunDerivesFixedPreambleFromPerTurnMinimum(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete, ResolvedConfig: &domain.RunConfig{
		PreambleInjectedTokens: 20,
		PreambleTokenBasis:     tokenaccounting.BasisEstimated,
	}}
	fact := ProjectRun(run, []*domain.RunEvent{
		{Data: &domain.UsageEventData{InputTokens: 80, OutputTokens: 5, TurnIndex: 1}},
		{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 6, TurnIndex: 2}},
		{Data: &domain.UsageEventData{InputTokens: 90, OutputTokens: 7, TurnIndex: 3}},
	}, time.Now().UTC())
	if fact.PreambleInjectedTokens != 20 || fact.PreambleFixedTokens != 60 || fact.PreambleTokenBasis != tokenaccounting.BasisEstimated {
		t.Fatalf("preamble=%+v, want injected=20 fixed=60 estimated", fact)
	}
}

func TestProjectRunWithoutPerTurnUsageLeavesPreambleUnknown(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), ExecutionMode: domain.ExecutionModeImported, ResolvedConfig: &domain.RunConfig{
		PreambleInjectedTokens: 20,
		PreambleTokenBasis:     tokenaccounting.BasisEstimated,
	}}
	fact := ProjectRun(run, nil, time.Now().UTC())
	if fact.PreambleInjectedTokens != 0 || fact.PreambleFixedTokens != 0 || fact.PreambleTokenBasis != tokenaccounting.BasisUnknown {
		t.Fatalf("preamble=%+v, want zero values with unknown basis", fact)
	}
}

func TestProjectRun_SumsUsageWhenChargeIsUnpriced(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().Add(-time.Minute), Summary: &domain.RunSummary{TurnsUsed: 3}}
	fact := ProjectRun(run, []*domain.RunEvent{
		{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 5}},
		{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisUnpriced}},
	}, time.Now())
	if fact.TotalTokens != 15 || fact.UnpricedTokenCount != 15 {
		t.Fatalf("usage=%d unpriced=%d", fact.TotalTokens, fact.UnpricedTokenCount)
	}
	if fact.Turns != 3 {
		t.Fatalf("turns=%d", fact.Turns)
	}
}

func TestProjectRunMarksImportedNoUsageAsUnknownCostBasis(t *testing.T) {
	now := time.Now().UTC()
	run := &domain.Run{ID: uuid.New(), CreatedAt: now.Add(-time.Minute), ExecutionMode: domain.ExecutionModeImported, Status: domain.RunStatusComplete}
	fact := ProjectRun(run, nil, now)
	if fact.CostTimeBasis != "unknown" {
		t.Fatalf("cost time basis=%q, want unknown", fact.CostTimeBasis)
	}
}

func TestConsumptionPerSuccessfulCompletionCountsFailedAttempts(t *testing.T) {
	got := ConsumptionPerSuccessfulCompletion([]RunFact{{Status: "failed", TotalTokens: 90}, {Status: "complete", TotalTokens: 10}})
	if got.Consumption != 100 || got.SuccessfulRuns != 1 || got.PerSuccessfulRun != 100 {
		t.Fatalf("efficiency=%+v", got)
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

// TestProjectApportionsCompoundCommandTokensAcrossSegments guards the defect
// where every segment of a compound command carried the whole call's payload.
// Each existing conservation fixture uses a single-word command, so all of them
// pass with segment_count == 1 and none of them can observe the fan-out.
func TestProjectApportionsCompoundCommandTokensAcrossSegments(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	const compound = "cd /repo && rg pattern | head -20"
	output := strings.Repeat("a matching line of output\n", 40)
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": compound})
	result := domain.NewToolResultEvent(run.ID, "shell", "call-1", output, nil)
	usage := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 8, CacheCreationTokens: 4, TurnIndex: 1}}
	events := []*domain.RunEvent{call, result, usage}

	facts, _ := Project(run, events, time.Now().UTC())
	if len(facts) != 3 {
		t.Fatalf("facts=%d, want one per segment of %q", len(facts), compound)
	}
	for _, fact := range facts {
		if fact.SegmentCount != 3 || fact.CallEventID != call.ID.String() {
			t.Fatalf("segment fact does not belong to the one call: %+v", fact.InvocationFact)
		}
	}

	encoded, err := json.Marshal(map[string]any{"command": compound})
	if err != nil {
		t.Fatalf("marshal call input: %v", err)
	}
	wantArg := tokenaccounting.EstimateText(string(encoded)).Tokens
	wantResult := tokenaccounting.EstimateText(output).Tokens
	if wantArg == 0 || wantResult == 0 {
		t.Fatalf("fixture must produce a non-zero payload: arg=%d result=%d", wantArg, wantResult)
	}

	var gotArg, gotResult, gotIncurred int64
	for _, fact := range facts {
		gotArg += fact.ArgTokens
		gotResult += fact.ResultTokens
		gotIncurred += fact.IncurredInputTokens + fact.IncurredOutputTokens + fact.IncurredCacheReadTokens + fact.IncurredCacheCreationTokens
	}
	if gotArg != wantArg {
		t.Fatalf("summed arg tokens = %d, want the call's %d (fan-out would give %d)", gotArg, wantArg, wantArg*3)
	}
	if gotResult != wantResult {
		t.Fatalf("summed result tokens = %d, want the call's %d (fan-out would give %d)", gotResult, wantResult, wantResult*3)
	}

	// The run ledger sums the by-call attribution map rather than the fact
	// rows, so it stayed correct while the per-command ranking was inflated.
	// Reconciling the fact rows against it is what ties the two together.
	runFact := ProjectRun(run, events, time.Now().UTC())
	if runFact.PreambleInjectedTokens+runFact.PreambleFixedTokens+gotIncurred+runFact.UnattributedTokens != runFact.TotalTokens {
		t.Fatalf("facts do not reconcile to the run: preamble=%d+%d incurred=%d residual=%d total=%d",
			runFact.PreambleInjectedTokens, runFact.PreambleFixedTokens, gotIncurred, runFact.UnattributedTokens, runFact.TotalTokens)
	}
}

// TestProjectDoesNotConcentrateCompoundTokensOnTheLeadingSegment pins the
// choice of an even split. Charging the call to segment zero would send the
// plurality of all shell token spend to `cd`, which leads every compound line.
func TestProjectDoesNotConcentrateCompoundTokensOnTheLeadingSegment(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "cd /repo && rg pattern"})
	result := domain.NewToolResultEvent(run.ID, "shell", "call-1", strings.Repeat("line\n", 100), nil)
	facts, _ := Project(run, []*domain.RunEvent{call, result}, time.Now().UTC())
	if len(facts) != 2 {
		t.Fatalf("facts=%d, want two segments", len(facts))
	}
	leading, trailing := facts[0], facts[1]
	if leading.Executable != "cd" {
		t.Fatalf("fixture assumption broken: leading executable=%q", leading.Executable)
	}
	if leading.ResultTokens > trailing.ResultTokens+1 {
		t.Fatalf("leading segment absorbed the payload: cd=%d rg=%d", leading.ResultTokens, trailing.ResultTokens)
	}
}
