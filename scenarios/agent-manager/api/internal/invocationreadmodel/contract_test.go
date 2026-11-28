package invocationreadmodel

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/toolfriction"

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
	facts, watermark := Project(run, []*domain.RunEvent{event}, now, nil)
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
	facts, _ := Project(run, events, now, nil)
	if len(facts) != 1 {
		t.Fatalf("facts=%+v", facts)
	}
	if facts[0].ArgTokens == 0 || facts[0].ResultTokens == 0 || facts[0].TokenBasis != "estimated" || facts[0].ResidencyRequests != 2 || facts[0].ResidencySegment != 0 {
		t.Fatalf("token factors=%+v", facts[0].InvocationFact)
	}
	second, _ := Project(run, events, now, nil)
	if !reflect.DeepEqual(facts, second) {
		t.Fatal("projection is not idempotent")
	}
}

func TestProjectMarksUnpairedTokenBasisUnknown(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "shell", "missing-result", map[string]any{"command": "echo hi"})
	facts, _ := Project(run, []*domain.RunEvent{call}, time.Now(), nil)
	if len(facts) != 1 || facts[0].ArgTokens == 0 || facts[0].TokenBasis != "unknown" || facts[0].ResultTokens != 0 {
		t.Fatalf("unpaired factors=%+v", facts)
	}
}

func TestProjectUsesProjectionTimeForEmptyEventSet(t *testing.T) {
	now := time.Date(2026, time.August, 4, 19, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	_, watermark := Project(run, nil, now, nil)
	if watermark.LastEventID != "" || !watermark.LastEventAt.Equal(now) {
		t.Fatalf("empty projection watermark=%+v", watermark)
	}
}

func TestProjectDerivesDurableSelfReportSpansFromSameEvents(t *testing.T) {
	now := time.Date(2026, time.July, 30, 18, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, Tag: "evidence", ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-test"}}
	first := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "missing-command"})
	answer := domain.NewMessageEvent(run.ID, "assistant", "Done.")
	answer.Data.(*domain.MessageEventData).Terminal = true
	second := domain.NewToolCallEvent(run.ID, "shell", "call-2", map[string]any{"command": "missing-command"})
	message := domain.NewMessageEvent(run.ID, "assistant", "I am blocked until the toolchain is repaired")
	first.Timestamp, first.Sequence = now, 1
	answer.Timestamp, answer.Sequence = now.Add(30*time.Second), 2
	second.Timestamp, second.Sequence = now.Add(time.Minute), 3
	message.Timestamp, message.Sequence = now.Add(2*time.Minute), 4
	events := []*domain.RunEvent{first, answer, second, message}
	_, watermark := Project(run, events, now.Add(3*time.Minute), nil)
	spans := ProjectSelfReportSpans(run, events, now.Add(3*time.Minute))
	if len(spans) != 1 || spans[0].EventID != message.ID.String() || spans[0].Text != "I am blocked" || spans[0].ClassifierVersion != runsignal.SelfReportClassifierVersion {
		t.Fatalf("spans=%+v", spans)
	}
	if watermark.SelfReportClassifierVersion != runsignal.SelfReportClassifierVersion {
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
	if fact.TotalTokens != 42 || fact.TotalCostUSD != 1 || fact.MeteredChargeMicroUSD != 1000000 || fact.Model != "actual" || fact.RunnerType != "codex" {
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

func TestProjectRunProviderTurnsOverrideMessageCountAndAccumulateContinuations(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Summary: &domain.RunSummary{TurnsUsed: 6}}
	events := []*domain.RunEvent{
		{Data: &domain.StatusEventData{NewStatus: "running"}},
		{Data: &domain.UsageEventData{InputTokens: 89765, CacheReadTokens: 1108224, OutputTokens: 7080, Turns: 1, ReconciliationAuthority: true}},
	}
	first := ProjectRun(run, events, time.Now().UTC())
	if first.Turns != 1 || first.TotalTokens != 1205069 {
		t.Fatalf("summary displaced provider authority: %+v", first)
	}
	events = append(events, &domain.RunEvent{Data: &domain.StatusEventData{NewStatus: "running"}}, &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 5, Turns: 1, ReconciliationAuthority: true}})
	continued := ProjectRun(run, events, time.Now().UTC())
	if continued.Turns != 2 || continued.TotalTokens != 1205084 {
		t.Fatalf("continuation invocation counts were reset or inflated: %+v", continued)
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
	facts, _ := Project(run, events, time.Now().UTC(), nil)
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

func TestProjectRunReconcilesEachContinuationWithinItsOwnInvocation(t *testing.T) {
	id := uuid.New()
	run := &domain.Run{ID: id, Status: domain.RunStatusComplete}
	events := []*domain.RunEvent{
		domain.NewStatusEvent(id, "starting", "running", "execute"),
		{Data: &domain.UsageEventData{InputTokens: 3}},
		{Data: &domain.UsageEventData{InputTokens: 10, ReconciliationAuthority: true}},
		domain.NewStatusEvent(id, "complete", "running", "Continuation requested"),
		domain.NewStatusEvent(id, "running", "running", "Codex continuation started"),
		{Data: &domain.UsageEventData{InputTokens: 4}},
		{Data: &domain.UsageEventData{InputTokens: 10, ReconciliationAuthority: true}},
		{Data: &domain.UsageEventData{InputTokens: 10, ReconciliationAuthority: true}},
	}
	if got := ProjectRun(run, events, time.Now()); got.TotalTokens != 20 {
		t.Fatalf("distinct equal-cost invocations collapsed or intermediate/replayed usage counted: %+v", got)
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
			facts, _ := Project(run, fixture.events, time.Now().UTC(), nil)
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

// A result stays in context, and is re-read by every provider request, until
// the next compaction; residency counts those requests.
func TestResidencyCountsRequestsUntilTheNextCompaction(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "Read", "call-1", map[string]any{"path": "a.go"})
	result := domain.NewToolResultEvent(run.ID, "Read", "call-1", "result", nil)
	callAfterCompaction := domain.NewToolCallEvent(run.ID, "Read", "call-2", map[string]any{"path": "b.go"})
	resultAfterCompaction := domain.NewToolResultEvent(run.ID, "Read", "call-2", "result", nil)
	usage := func(turn int) *domain.RunEvent {
		return &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 5, TurnIndex: turn}}
	}
	events := []*domain.RunEvent{
		call, result, usage(1), usage(2),
		domain.NewCompactionEvent(run.ID, "first", "manual", "", 3, 100, 50, "/compact"),
		usage(3), callAfterCompaction, resultAfterCompaction, usage(4),
		domain.NewCompactionEvent(run.ID, "second", "manual", "", 2, 50, 25, "/compact"),
		usage(5),
	}
	facts, _ := Project(run, events, time.Now().UTC(), nil)
	if len(facts) != 2 {
		t.Fatalf("facts=%d, want two", len(facts))
	}
	if facts[0].ResidencySegment != 0 || facts[0].ResidencyRequests != 2 {
		t.Fatalf("first fact=%+v, want 2 requests in segment 0", facts[0])
	}
	if facts[1].ResidencySegment != 1 || facts[1].ResidencyRequests != 1 {
		t.Fatalf("second fact=%+v, want 1 request in segment 1", facts[1])
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

// TestProjectApportionsCompoundCommandTokensAcrossSegments guards the defect
// where every segment of a compound command carried the whole call's payload.
// Each existing conservation fixture uses a single-word command, so all of them
// pass with segment_count == 1 and none of them can observe the fan-out.
func TestProjectApportionsCompoundCommandTokensAcrossSegments(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	// `cd` is shell navigation, not an invocation: the call has two segments.
	const compound = "cd /repo && rg pattern | head -20"
	output := strings.Repeat("a matching line of output\n", 40)
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": compound})
	result := domain.NewToolResultEvent(run.ID, "shell", "call-1", output, nil)
	usage := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 8, CacheCreationTokens: 4, TurnIndex: 1}}
	events := []*domain.RunEvent{call, result, usage}

	facts, _ := Project(run, events, time.Now().UTC(), nil)
	if len(facts) != 2 {
		t.Fatalf("facts=%d, want one per invocation (no fact for cd) of %q", len(facts), compound)
	}
	for _, fact := range facts {
		if fact.SegmentCount != 2 || fact.CallEventID != call.ID.String() {
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
		t.Fatalf("summed arg tokens = %d, want the call's %d (fan-out would give %d)", gotArg, wantArg, wantArg*2)
	}
	if gotResult != wantResult {
		t.Fatalf("summed result tokens = %d, want the call's %d (fan-out would give %d)", gotResult, wantResult, wantResult*2)
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

// A call's output counts once, on the segment that holds it. A compound
// command with one result gives its tokens to its holder (the segment that
// also holds its duration); the segments before it are unknown, not an even
// share of output they may not have printed.
func TestProjectGivesACompoundCommandsTokensToItsHolder(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "shell", "call-1", map[string]any{"command": "git status && rg pattern | head -5"})
	result := domain.NewToolResultEvent(run.ID, "shell", "call-1", strings.Repeat("line\n", 100), nil)
	facts, _ := Project(run, []*domain.RunEvent{call, result}, time.Now().UTC(), nil)
	got := map[string]Fact{}
	for _, fact := range facts {
		got[fact.Executable] = fact
	}
	if len(facts) != 3 || got["rg"].ResultTokens != 125 || got["rg"].TokenBasis != tokenaccounting.BasisEstimated {
		t.Fatalf("facts=%+v, want rg to hold the call's 125 tokens", facts)
	}
	for _, other := range []string{"git", "head"} {
		if got[other].ResultTokens != 0 || got[other].TokenBasis != tokenaccounting.BasisUnknown {
			t.Fatalf("%s=%+v, want unknown tokens: its output is inside rg's", other, got[other].InvocationFact)
		}
	}
}

// A script that only runs its commands, with no nested result to size them,
// gives its result evenly to the commands' holders: no unit is the script's
// own, and no command can be told apart.
func TestProjectSplitsAnUnsizedExecResultAcrossItsCommands(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "exec", "A", map[string]any{"input": `await tools.exec_command({cmd:"rg -n x ."}); await tools.exec_command({cmd:"git status && jq . a.json"});`})
	output := domain.NewToolResultEvent(run.ID, "", "A", strings.Repeat("x", 4000), nil)
	facts, _ := Project(run, []*domain.RunEvent{call, output}, time.Now().UTC(), nil)
	got := map[string]int64{}
	for _, fact := range facts {
		got[fact.CommandPath] = fact.ResultTokens
	}
	if len(facts) != 3 || got["rg"] != 500 || got["jq"] != 500 || got["git status"] != 0 {
		t.Fatalf("tokens=%v, want rg and jq (the holders) 500 each, git status none", got)
	}
}

// Each command of a Codex exec batch takes the tokens its own nested result
// says it could have shown (printed bytes, capped by its max_output_tokens);
// the script's own unit takes the rest of the result. A compound command's
// one output sits on its holder, a command without a nested result is
// unknown, and the call still sums to its result tokens. A forked rollout
// that kept no nested results gives the whole result to the script.
func TestProjectGivesEachExecCommandItsOwnOutputTokens(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	const script = `const r=await Promise.allSettled([
tools.exec_command({cmd:"prompt-manager skill read vrooli-memory; plan-manager plans list --json",max_output_tokens:6500}),
tools.exec_command({cmd:"rg -n x .",max_output_tokens:1000}),
tools.exec_command({cmd:"cat notes.md"})
]);for(let i=0;i<r.length;i++)text(JSON.stringify(r[i]))`
	nested := func(command string, outputBytes int) *domain.RunEvent {
		event := domain.NewToolResultEvent(run.ID, runsignal.NestedCommandTool, "exec-"+command, "", nil)
		data := event.Data.(*domain.ToolResultEventData)
		exit, known := 0, true
		data.ExitCode, data.OutcomeKnown = &exit, &known
		data.RawOutputJSON = fmt.Sprintf(`{"command":["/bin/bash","-lc",%q],"output_bytes":%d}`, command, outputBytes)
		return event
	}
	type want struct {
		tokens, bytes int64
		basis         tokenaccounting.Basis
	}
	unknown := want{basis: tokenaccounting.BasisUnknown}
	for name, tc := range map[string]struct {
		nested      []*domain.RunEvent
		resultBytes int
		want        map[string]want
	}{
		"own outputs, script takes the rest": {
			nested:      []*domain.RunEvent{nested("prompt-manager skill read vrooli-memory; plan-manager plans list --json", 1_048_608), nested("rg -n x .", 300)},
			resultBytes: 80_000, // 20,000 tokens
			want: map[string]want{
				"prompt-manager skill read": unknown,
				"plan-manager plans list":   {6500, 1_048_608, tokenaccounting.BasisEstimated},
				"rg":                        {75, 300, tokenaccounting.BasisEstimated},
				"cat":                       unknown,
				"":                          {20_000 - 6500 - 75, 0, tokenaccounting.BasisEstimated},
			},
		},
		"claims beyond the result scale to it": {
			nested:      []*domain.RunEvent{nested("prompt-manager skill read vrooli-memory; plan-manager plans list --json", 1_048_608), nested("rg -n x .", 300)},
			resultBytes: 4_000, // 1,000 tokens against 6,575 claimed
			want: map[string]want{
				"prompt-manager skill read": unknown,
				"plan-manager plans list":   {989, 1_048_608, tokenaccounting.BasisEstimated},
				"rg":                        {11, 300, tokenaccounting.BasisEstimated},
				"cat":                       unknown,
				"":                          {0, 0, tokenaccounting.BasisEstimated},
			},
		},
		"no nested results": {
			resultBytes: 12_000,
			want: map[string]want{
				"prompt-manager skill read": unknown,
				"plan-manager plans list":   unknown,
				"rg":                        unknown,
				"cat":                       unknown,
				"":                          {3000, 0, tokenaccounting.BasisEstimated},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			call := domain.NewToolCallEvent(run.ID, "exec", "A", map[string]any{"input": script})
			output := domain.NewToolResultEvent(run.ID, "", "A", strings.Repeat("x", tc.resultBytes), nil)
			events := append(append([]*domain.RunEvent{call}, tc.nested...), output)
			facts, _ := Project(run, events, time.Now().UTC(), nil)
			if len(facts) != len(tc.want) {
				t.Fatalf("facts=%d, want %d", len(facts), len(tc.want))
			}
			var sum int64
			for _, fact := range facts {
				sum += fact.ResultTokens
				got := want{fact.ResultTokens, fact.OutputBytes, fact.TokenBasis}
				if got != tc.want[fact.CommandPath] {
					t.Errorf("%q: got %+v, want %+v", fact.CommandPath, got, tc.want[fact.CommandPath])
				}
			}
			if sum != int64(tc.resultBytes/4) {
				t.Fatalf("segments sum to %d tokens, want the call's %d", sum, tc.resultBytes/4)
			}
		})
	}
}

// An exec call's result tokens are what the agent saw; they split across the
// commands it ran by each command's own output size and still sum to the
// call. Residency is read from that same output.
func TestProjectSplitsExecResultTokensByCommandOutput(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "exec", "A", map[string]any{"input": `await tools.exec_command({cmd:"rg -n x ."}); await tools.exec_command({cmd:"git log -1"});`})
	nested := func(command string, outputBytes int) *domain.RunEvent {
		event := domain.NewToolResultEvent(run.ID, runsignal.NestedCommandTool, "exec-"+command, "", nil)
		data := event.Data.(*domain.ToolResultEventData)
		exit, known := 0, true
		data.ExitCode, data.OutcomeKnown = &exit, &known
		data.RawOutputJSON = fmt.Sprintf(`{"command":["/bin/bash","-lc",%q],"output_bytes":%d}`, command, outputBytes)
		return event
	}
	output := domain.NewToolResultEvent(run.ID, "", "A", strings.Repeat("x", 4003), nil) // 1,001 tokens
	usage := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 1, TurnIndex: 1}}
	events := []*domain.RunEvent{call, nested("rg -n x .", 300), nested("git log -1", 100), output, usage}
	facts, _ := Project(run, events, time.Now().UTC(), nil)
	if len(facts) != 2 {
		t.Fatalf("facts=%+v", facts)
	}
	rg, git := facts[0], facts[1]
	if rg.ResultTokens != 751 || git.ResultTokens != 250 || rg.OutputBytes != 300 || git.OutputBytes != 100 {
		t.Fatalf("result tokens rg=%d git=%d bytes %d/%d, want 751/250 by output size", rg.ResultTokens, git.ResultTokens, rg.OutputBytes, git.OutputBytes)
	}
	for _, fact := range facts {
		if fact.OutputEventID != output.ID.String() || fact.ResidencyRequests != 1 || fact.Outcome != runsignal.OutcomeSuccess {
			t.Fatalf("fact=%+v", fact.InvocationFact)
		}
	}
}

// The text features of a command (error signature, truncation, bypass owner)
// are read from the events at projection, with a port resolved to its owner
// then, so a read never reopens run events.
func TestProjectStoresToolFeaturesReadFromTheEvents(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	call := domain.NewToolCallEvent(run.ID, "Bash", "call-1", map[string]any{"command": "curl -s localhost:8123/api/runs"})
	exit := 7
	result := domain.NewToolResultEvent(run.ID, "Bash", "call-1", "Error: connection refused\n<persisted-output>", nil)
	result.Data.(*domain.ToolResultEventData).ExitCode = &exit
	facts, _ := Project(run, []*domain.RunEvent{call, result}, time.Now().UTC(), func(port int) string {
		if port == 8123 {
			return "agent-manager"
		}
		return ""
	})
	if len(facts) == 0 || facts[0].Tool == nil {
		t.Fatalf("facts = %+v, want a command fact with stored tool features", facts)
	}
	tool := facts[0].Tool
	if tool.BypassOwner != "agent-manager" || tool.BypassMethod != toolfriction.BypassHTTP || !tool.Truncated || tool.ErrorLine != "Error: connection refused" || tool.Command == "" {
		t.Fatalf("tool features = %+v", tool)
	}
}

// A Codex exec result carries its text in blocks (its Output is empty), and a
// 1 MiB command output reaches the model cut to what the harness showed. The
// result tokens are what the model read, so output weight and context tax see
// the command; the printed size stays on the fact.
func TestProjectCountsCodexExecBlocksAsResultTokens(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now().UTC(), Status: domain.RunStatusComplete}
	var events []*domain.RunEvent
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("call-%d", i)
		call := domain.NewToolCallEvent(run.ID, "exec", id, map[string]any{"input": `await tools.exec_command({cmd:"vrooli locks"});`})
		nested := domain.NewToolResultEvent(run.ID, runsignal.NestedCommandTool, "exec-"+id, "", nil)
		data := nested.Data.(*domain.ToolResultEventData)
		exit, known := 0, true
		data.ExitCode, data.OutcomeKnown = &exit, &known
		data.RawOutputJSON = `{"command":["/bin/bash","-lc","vrooli locks"],"output_bytes":1048576}`
		output := domain.NewToolResultEvent(run.ID, "", id, "", nil)
		output.Data.(*domain.ToolResultEventData).OutputBlocks = []domain.ToolOutputBlock{{Type: "input_text", Text: strings.Repeat("x", 40_000)}}
		usage := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 1, TurnIndex: i + 1}}
		events = append(events, call, nested, output, usage)
	}
	facts, _ := Project(run, events, time.Now().UTC(), nil)
	var calls []toolfriction.Call
	for _, fact := range facts {
		if fact.ResultTokens != 10_000 || fact.TokenBasis != tokenaccounting.BasisEstimated || fact.OutputBytes != 1048576 {
			t.Fatalf("fact = %+v: want 10,000 estimated tokens read of 1 MiB printed", fact.InvocationFact)
		}
		calls = append(calls, toolfriction.Observe(toolfriction.Fact{InvocationFact: fact.InvocationFact, RunID: fact.RunID, At: fact.OccurredAt}, *fact.Tool))
	}
	report := toolfriction.Analyze(toolfriction.Window{Calls: calls})
	for _, f := range report.Findings {
		if f.Kind == toolfriction.KindOutputWeight && f.CommandPath == "vrooli locks" && f.Cost > 0 && strings.Contains(f.Description, "printed about") {
			return
		}
	}
	t.Fatalf("no output-weight finding for vrooli locks naming the printed size: %+v", report.Findings)
}

// longRun is a deterministic synthetic run of n events: tool calls with their
// results, provider requests (per-turn usage, or assistant messages when
// perTurn is false) and a compaction every compactEvery events (0: none).
func longRun(runID uuid.UUID, n, compactEvery int, perTurn bool) []*domain.RunEvent {
	events := make([]*domain.RunEvent, 0, n)
	for i := 0; len(events) < n; i++ {
		switch {
		case compactEvery > 0 && i > 0 && i%compactEvery == 0:
			events = append(events, domain.NewCompactionEvent(runID, "auto", "auto", "", 3, 100, 50, ""))
		case i%5 == 0:
			id := fmt.Sprintf("call-%d", i)
			events = append(events, domain.NewToolCallEvent(runID, "Bash", id, map[string]any{"command": "make test"}),
				domain.NewToolResultEvent(runID, "Bash", id, strings.Repeat("x", i%300), nil))
		case perTurn:
			events = append(events, &domain.RunEvent{ID: uuid.New(), Data: &domain.UsageEventData{InputTokens: 100, OutputTokens: 5, TurnIndex: i}})
		default:
			events = append(events, domain.NewProviderMessageEvent(runID, "assistant", "ok", domain.MessageEventData{}))
		}
	}
	return events
}

// Residency is the requests between a result and the next compaction (or the
// run's end), counted directly here as the reference for long runs.
func residencyByDefinition(events []*domain.RunEvent, perTurn bool) map[string]int64 {
	out := map[string]int64{}
	for i, ev := range events {
		if _, ok := ev.Data.(*domain.ToolResultEventData); !ok {
			continue
		}
		for _, later := range events[i+1:] {
			if _, ok := later.Data.(*domain.CompactionEventData); ok {
				break
			}
			if usage, ok := later.Data.(*domain.UsageEventData); ok && perTurn && !usage.ReconciliationAuthority {
				out[ev.ID.String()]++
			}
			if message, ok := later.Data.(*domain.MessageEventData); ok && !perTurn && message.Role == "assistant" {
				out[ev.ID.String()]++
			}
		}
	}
	return out
}

// Residency on long runs (with and without compactions, with per-turn usage
// or assistant messages as requests) matches its definition result by result.
func TestResidencyOnLongRunsMatchesItsDefinition(t *testing.T) {
	runID := uuid.New()
	for _, tc := range []struct {
		n, compactEvery int
		perTurn         bool
	}{{20_000, 0, true}, {20_000, 3_000, true}, {20_000, 0, false}, {20_000, 2_500, false}} {
		events := longRun(runID, tc.n, tc.compactEvery, tc.perTurn)
		_, _, _, got, _ := tokenFactors(events)
		want := residencyByDefinition(events, tc.perTurn)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("n=%d compactEvery=%d perTurn=%v: residency differs from its definition (%d vs %d results)", tc.n, tc.compactEvery, tc.perTurn, len(got), len(want))
		}
	}
}

func BenchmarkTokenFactorsLongRun(b *testing.B) {
	events := longRun(uuid.New(), 50_000, 0, true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenFactors(events)
	}
}
