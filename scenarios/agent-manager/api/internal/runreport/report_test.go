package runreport

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

type reportSource struct {
	run    *domain.Run
	events []*domain.RunEvent
	diff   *sandbox.DiffResult
}

type receiptReportSource struct {
	reportSource
	receipts ReceiptSummary
}

type durableReportSource struct {
	reportSource
	facts      []runsignal.InvocationFact
	accounting runsignal.TimeAccounting
}

type canonicalUsageReportSource struct {
	reportSource
	tokens               int64
	usageFullyReconciled bool
}

func (s canonicalUsageReportSource) CanonicalUsageProjection(*domain.Run, []*domain.RunEvent) (int64, bool) {
	return s.tokens, s.usageFullyReconciled
}

func (s durableReportSource) DurableInvocationFacts(context.Context, uuid.UUID) ([]runsignal.InvocationFact, bool, error) {
	return s.facts, true, nil
}

func (s durableReportSource) DurableTimeAccounting(context.Context, uuid.UUID) (runsignal.TimeAccounting, bool, error) {
	return s.accounting, true, nil
}

func (s receiptReportSource) Receipts(context.Context, uuid.UUID) (ReceiptSummary, error) {
	return s.receipts, nil
}

func TestBuildPrefersDurableTimeAccountingAfterEventRetention(t *testing.T) {
	id := uuid.New()
	start, end := time.Now().Add(-10*time.Minute), time.Now()
	want := runsignal.TimeAccounting{ToolExecutingMS: 2 * time.Minute.Milliseconds(), UnattributableMS: 8 * time.Minute.Milliseconds(), ToolTokens: 12}
	report, err := Build(context.Background(), durableReportSource{reportSource: reportSource{run: &domain.Run{ID: id, StartedAt: &start, EndedAt: &end}}, accounting: want}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.TimeAccounting != want {
		t.Fatalf("accounting=%+v want %+v", report.TimeAccounting, want)
	}
}

func (s reportSource) Run(context.Context, uuid.UUID) (*domain.Run, error) { return s.run, nil }
func (s reportSource) Events(context.Context, uuid.UUID) ([]*domain.RunEvent, error) {
	return s.events, nil
}

func (s reportSource) Diff(context.Context, uuid.UUID) (*sandbox.DiffResult, error) {
	return s.diff, nil
}

func TestBuildIncludesInvestigationDiscriminators(t *testing.T) {
	id := uuid.New()
	start := time.Now().Add(-time.Minute)
	end := time.Now()
	exit := 1
	run := &domain.Run{ID: id, Status: domain.RunStatusFailed, StartedAt: &start, EndedAt: &end, LastHeartbeat: &start, ExitCode: &exit, ErrorMsg: "tool failed", RequestedModel: "primary", ActualModel: "fallback", Result: &domain.RunResult{Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionAmbiguous, Rule: "multiple"}, Candidates: []domain.FinalOutputCandidate{{}, {}}, Structured: &domain.StructuredResult{Status: domain.StructuredResultInvalid, Method: "deterministic", Diagnostics: []domain.StructuredDiagnostic{{Code: "schema_invalid"}}}}}
	events := []*domain.RunEvent{
		domain.NewToolCallEvent(id, "read_file", "read-1", map[string]interface{}{"path": "README.md"}),
		domain.NewToolCallEvent(id, "read_file", "read-2", map[string]interface{}{"path": "README.md"}),
		domain.NewToolCallEvent(id, "shell", "1", map[string]interface{}{"command": "agent-manager"}),
		domain.NewToolCallEvent(id, "shell", "2", map[string]interface{}{"command": "agent-manager"}),
		domain.NewToolResultEvent(id, "shell", "1", "", assertErr{}),
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeModelFallbackAttempted, Timestamp: end},
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Timestamp: end, Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 5}},
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Timestamp: end, Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: func() *int64 { v := int64(1250000); return &v }()}},
	}
	report, err := Build(context.Background(), reportSource{run: run, events: events, diff: &sandbox.DiffResult{}}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Result.SelectionStatus != domain.FinalOutputSelectionAmbiguous || report.Result.StructuredStatus != domain.StructuredResultInvalid {
		t.Fatalf("result discriminators missing: %#v", report.Result)
	}
	if report.Tools[1].Failures != 1 || report.ProjectOwnedToolCalls != 2 || report.ExternalToolCalls != 2 || report.FallbackCount != 1 {
		t.Fatalf("tool/fallback discriminators missing: %#v", report)
	}
	if report.Tokens != 15 || report.CostUSD != 1.25 {
		t.Fatalf("cost discriminators = %d/%f", report.Tokens, report.CostUSD)
	}
	if report.RepeatedToolCalls != 1 {
		t.Fatalf("repeated calls = %d, want one repeated command signature", report.RepeatedToolCalls)
	}
	if report.FilesReadMoreThanOnce != 1 {
		t.Fatalf("files reread = %d, want 1", report.FilesReadMoreThanOnce)
	}
}

func TestBuildCountsRepeatedCommandSignaturesNotToolNames(t *testing.T) {
	id := uuid.New()
	first := domain.NewToolCallEvent(id, "shell", "shell-1", map[string]any{"command": "agent-manager run report run-1"})
	second := domain.NewToolCallEvent(id, "shell", "shell-2", map[string]any{"command": "agent-manager run report run-1"})
	events := []*domain.RunEvent{first, second}
	for i := 0; i < 30; i++ {
		events = append(events, domain.NewToolCallEvent(id, "exec", fmt.Sprintf("exec-%d", i), map[string]any{"input": fmt.Sprintf("const value = %d; text(value)", i)}))
	}
	report, err := Build(context.Background(), reportSource{
		run:    &domain.Run{ID: id},
		events: events,
	}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.RepeatedToolCalls != 1 {
		t.Fatalf("repeated calls = %d, want only the repeated command signature", report.RepeatedToolCalls)
	}
}

func TestBuildSeparatesTypedErrorBreakdownFromRawEventCounts(t *testing.T) {
	id := uuid.New()
	percent := 2.0
	events := []*domain.RunEvent{
		domain.NewErrorEvent(id, "provider_failure", "request failed", false),
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeError, Data: &domain.RateLimitEventData{Provider: "openai", Pool: "primary", UsedPercent: &percent}},
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeError},
	}
	report, err := Build(context.Background(), reportSource{run: &domain.Run{ID: id}, events: events}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Events[string(domain.EventTypeError)] != 3 {
		t.Fatalf("raw error event count = %d, want 3", report.Events[string(domain.EventTypeError)])
	}
	if report.ExecutionErrorEvents != 1 || report.RateLimitObservations != 1 || report.UnclassifiedErrorEvents != 1 {
		t.Fatalf("typed error breakdown = execution:%d quota:%d unclassified:%d", report.ExecutionErrorEvents, report.RateLimitObservations, report.UnclassifiedErrorEvents)
	}
	text := Text(report)
	for _, want := range []string{"error=3", "typed_execution_errors=1", "rate_limit_observations=1", "unclassified=1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report text missing %q: %s", want, text)
		}
	}
}

func TestTextDoesNotPresentUnmeasuredErrorBreakdownAsZero(t *testing.T) {
	report := &RunReport{EventsAvailability: Availability{State: AvailabilityUnavailable, Reason: "event store unavailable"}}
	text := Text(report)
	if !strings.Contains(text, "Error event breakdown: unavailable (unavailable)") {
		t.Fatalf("unavailable events were shown as a measured error breakdown:\n%s", text)
	}
}

func TestBuildPrefersDetailedCostEventsOverSummary(t *testing.T) {
	id := uuid.New()
	run := &domain.Run{ID: id, Summary: &domain.RunSummary{TokensUsed: 999, CostEstimate: 9.99}}
	amount := int64(120000)
	events := []*domain.RunEvent{{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{InputTokens: 4, OutputTokens: 6, ReconciliationAuthority: true}}, {ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &amount}}}
	report, err := Build(context.Background(), canonicalUsageReportSource{reportSource: reportSource{run: run, events: events}, tokens: 10, usageFullyReconciled: true}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Tokens != 10 || report.CostUSD != 0.12 {
		t.Fatalf("event cost should replace summary, got tokens=%d cost=%f", report.Tokens, report.CostUSD)
	}
}

func TestBuildUsesCanonicalUsageAndKeepsKnownSummaryForUnfinalizedEvents(t *testing.T) {
	id := uuid.New()
	events := []*domain.RunEvent{
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 1}},
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{InputTokens: 20, OutputTokens: 2, ReconciliationAuthority: true}},
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{InputTokens: 20, OutputTokens: 2, ReconciliationAuthority: true}},
	}
	knownSummary := &domain.Run{ID: id, Summary: &domain.RunSummary{TokensUsed: 999}}
	report, err := Build(context.Background(), canonicalUsageReportSource{reportSource: reportSource{run: knownSummary, events: events}, tokens: 42, usageFullyReconciled: true}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Tokens != 42 {
		t.Fatalf("authoritative usage total=%d, want canonical 42", report.Tokens)
	}

	interimEvents := events[:1]
	interim, err := Build(context.Background(), canonicalUsageReportSource{reportSource: reportSource{run: knownSummary, events: interimEvents}, tokens: 11}, id)
	if err != nil {
		t.Fatal(err)
	}
	if interim.Tokens != 999 {
		t.Fatalf("interim usage replaced known whole-run total: tokens=%d", interim.Tokens)
	}

	partiallyReconciledEvents := append(append([]*domain.RunEvent{}, events[1:3]...), domain.NewStatusEvent(id, "complete", "running", "new invocation"), interimEvents[0])
	partiallyReconciled, err := Build(context.Background(), canonicalUsageReportSource{reportSource: reportSource{run: knownSummary, events: partiallyReconciledEvents}, tokens: 25}, id)
	if err != nil {
		t.Fatal(err)
	}
	if partiallyReconciled.Tokens != 999 {
		t.Fatalf("mixed reconciled and interim invocations replaced known whole-run total: tokens=%d", partiallyReconciled.Tokens)
	}
}

func TestBuildRetainsSummaryWhenEventReadIsUnavailable(t *testing.T) {
	id := uuid.New()
	knownSummary := &domain.Run{ID: id, Summary: &domain.RunSummary{TokensUsed: 999}}
	source := eventErrorReportSource{reportSource: reportSource{run: knownSummary}}
	report, err := Build(context.Background(), source, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Tokens != 999 || report.EventsAvailability.State != AvailabilityUnavailable {
		t.Fatalf("event read failure replaced the known summary: tokens=%d availability=%+v", report.Tokens, report.EventsAvailability)
	}
}

type eventErrorReportSource struct{ reportSource }

func (s eventErrorReportSource) Events(context.Context, uuid.UUID) ([]*domain.RunEvent, error) {
	return nil, context.DeadlineExceeded
}

func TestTextLabelsNonMeteredAndUnpricedCostHonestly(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		reason string
		want   string
	}{
		{name: "subscription", source: "subscription", want: "cost: subscription"},
		{name: "local", source: "local", want: "cost: local"},
		{name: "unpriced", source: "unpriced", reason: "model_unpriced", want: "cost: unpriced (model_unpriced)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := Text(&RunReport{CostUSD: 0, CostSource: test.source, ChargeReason: test.reason})
			if !strings.Contains(text, test.want) {
				t.Fatalf("report=%q want %q", text, test.want)
			}
			if test.source != "metered" && strings.Contains(text, "cost: $0.00") {
				t.Fatalf("non-metered report presented zero as a dollar charge: %q", text)
			}
		})
	}
}

func TestBoundProjectionReportsOversizedAvailability(t *testing.T) {
	bounded, dropped := BoundProjection(map[string]any{"too_large": string(make([]byte, 4096))})
	if len(bounded) != 0 || dropped != 1 {
		t.Fatalf("bounded projection = %#v, dropped = %d", bounded, dropped)
	}
	availability := projectionAvailability([]CrossScenarioCall{{Projection: bounded, ProjectionDropCount: dropped, PolicyVersion: "v1"}})
	if availability.State != AvailabilityOversized {
		t.Fatalf("projection availability = %#v, want oversized", availability)
	}
}

func TestProjectionAvailabilityDistinguishesUnobservedPolicy(t *testing.T) {
	availability := projectionAvailability(nil)
	if availability.State != AvailabilityPolicyAbsent {
		t.Fatalf("empty receipts availability = %#v, want policy_absent", availability)
	}
}

func TestBuildKeepsUnverifiedReceiptWhenLedgerIsUnobserved(t *testing.T) {
	id := uuid.New()
	report, err := Build(context.Background(), receiptReportSource{
		reportSource: reportSource{run: &domain.Run{ID: id}, events: []*domain.RunEvent{}},
		receipts:     ReceiptSummary{Availability: Availability{State: AvailabilityUnobserved, Reason: "no verified receipts"}, Calls: []CrossScenarioCall{{ReceiptEventID: "receipt-unverified", TargetScenario: "search-hub", Verified: false, DurationMS: 42}}},
	}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.LedgerAvailability.State != "unobserved" || report.LedgerAvailability.Reason == "" || len(report.CrossScenarioCalls) != 1 || report.CrossScenarioCalls[0].Verified {
		t.Fatalf("unverified receipt evidence was not retained honestly: %+v", report)
	}
}

func TestBuildRendersDurableEvidenceAfterRawEventsAreSwept(t *testing.T) {
	id := uuid.New()
	fact := runsignal.InvocationFact{CallEventID: "swept-call", ToolName: "shell", Ownership: "project", Outcome: "failure", Availability: "available"}
	report, err := Build(context.Background(), durableReportSource{
		reportSource: reportSource{run: &domain.Run{ID: id, Status: domain.RunStatusFailed}, events: nil},
		facts:        []runsignal.InvocationFact{fact},
	}, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.InvocationFacts) != 1 || report.InvocationFacts[0].CallEventID != fact.CallEventID {
		t.Fatalf("report durable evidence = %+v, want swept run's projected fact", report.InvocationFacts)
	}
	if len(report.Events) != 0 {
		t.Fatalf("report raw events = %+v, want none after the retention sweep", report.Events)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "failed" }

func TestBuildCorrelatesUnnamedReturnsAndDistinguishesUnknownFromMissing(t *testing.T) {
	runID := uuid.New()
	unknown := false
	zero := 0
	nonzero := 7
	calls := []*domain.RunEvent{
		domain.NewToolCallEvent(runID, "exec", "unknown", nil), domain.NewToolCallEvent(runID, "exec", "zero", nil), domain.NewToolCallEvent(runID, "exec", "failure", nil),
	}
	for _, tc := range []struct {
		id   string
		exit *int
	}{{"unknown", nil}, {"zero", &zero}, {"failure", &nonzero}} {
		evt := domain.NewToolResultEvent(runID, "", tc.id, "", nil)
		data := evt.Data.(*domain.ToolResultEventData)
		data.ExitCode = tc.exit
		if tc.exit == nil {
			data.OutcomeKnown = &unknown
		}
		calls = append(calls, evt)
	}
	report, err := Build(context.Background(), reportSource{run: &domain.Run{ID: runID}, events: calls}, runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Tools) != 1 || report.Tools[0].Name != "exec" || report.Tools[0].Calls != 3 || report.Tools[0].Successes != 1 || report.Tools[0].Failures != 1 || report.Tools[0].Unresolved != 0 || report.Tools[0].Returned != 3 || report.Tools[0].UnknownOutcome != 1 {
		t.Fatalf("tools=%+v", report.Tools)
	}
	if !strings.Contains(Text(report), "calls=3 returned=3 success=1 failed=1 unknown=1 unresolved=0") {
		t.Fatalf("text lost independent counts: %s", Text(report))
	}
}

func TestBuildOversizedHistoricalExitCountsReturnedUnknown(t *testing.T) {
	id := uuid.New()
	code := int(4294967296)
	call := domain.NewToolCallEvent(id, "exec", "original", nil)
	result := &domain.RunEvent{ID: uuid.New(), RunID: id, EventType: domain.EventTypeToolResult, Data: &domain.ToolResultEventData{ToolCallID: "original", ExitCode: &code, Success: true}}
	report, err := Build(t.Context(), reportSource{run: &domain.Run{ID: id}, events: []*domain.RunEvent{call, result}}, id)
	if err != nil || len(report.Tools) != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	tool := report.Tools[0]
	if tool.Returned != 1 || tool.UnknownOutcome != 1 || tool.Unresolved != 0 || tool.Successes != 0 || tool.Failures != 0 {
		t.Fatalf("tool=%+v", tool)
	}
}
