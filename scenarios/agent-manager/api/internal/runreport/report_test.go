package runreport

import (
	"context"
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
		{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Timestamp: end, Data: &domain.CostEventData{InputTokens: 10, OutputTokens: 5, TotalCostUSD: 1.25}},
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
	if report.RepeatedToolCalls != 2 {
		t.Fatalf("repeated calls = %d, want 2", report.RepeatedToolCalls)
	}
	if report.FilesReadMoreThanOnce != 1 {
		t.Fatalf("files reread = %d, want 1", report.FilesReadMoreThanOnce)
	}
}

func TestBuildPrefersDetailedCostEventsOverSummary(t *testing.T) {
	id := uuid.New()
	run := &domain.Run{ID: id, Summary: &domain.RunSummary{TokensUsed: 999, CostEstimate: 9.99}}
	events := []*domain.RunEvent{{ID: uuid.New(), RunID: id, EventType: domain.EventTypeMetric, Data: &domain.CostEventData{InputTokens: 4, OutputTokens: 6, TotalCostUSD: 0.12}}}
	report, err := Build(context.Background(), reportSource{run: run, events: events}, id)
	if err != nil {
		t.Fatal(err)
	}
	if report.Tokens != 10 || report.CostUSD != 0.12 {
		t.Fatalf("event cost should replace summary, got tokens=%d cost=%f", report.Tokens, report.CostUSD)
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
