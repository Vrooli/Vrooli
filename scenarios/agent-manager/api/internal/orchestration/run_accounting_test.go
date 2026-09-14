package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func accountingFor(t *testing.T, run *domain.Run, events []*domain.RunEvent) RunAccounting {
	t.Helper()
	state, err := meteredWorkflowChildState(run, events, time.Now())
	if err != nil {
		t.Fatalf("meter: %v", err)
	}
	got, err := runAccountingFromState(run, state)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	return got
}

func TestRunAccountingSettlesAuthoritativeMeteredReceipt(t *testing.T) {
	started := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(90*time.Second + 200*time.Millisecond)
	amount := int64(4500)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, StartedAt: &started, EndedAt: &ended, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisMetered}}
	events := []*domain.RunEvent{{EventType: domain.EventTypeMetric, Timestamp: started.Add(time.Minute), Data: &domain.UsageEventData{InputTokens: 1000, OutputTokens: 200, Turns: 3, ReconciliationAuthority: true, Charge: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &amount}}}}

	got := accountingFor(t, run, events)
	if !got.Terminal || !got.TokensKnown || !got.ChargeMeasured || got.Tokens != 1200 || got.Turns != 3 || got.ChargeMicroUSD != 4500 || got.WallSeconds != 91 {
		t.Fatalf("authoritative metered receipt = %+v", got)
	}
}

func TestRunAccountingKeepsTokensUnknownWithoutTerminalReceipt(t *testing.T) {
	started := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(time.Minute)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusCancelled, StartedAt: &started, EndedAt: &ended}
	events := []*domain.RunEvent{{EventType: domain.EventTypeMetric, Timestamp: started, Data: &domain.UsageEventData{InputTokens: 70, OutputTokens: 30, Turns: 1}}}

	got := accountingFor(t, run, events)
	if !got.Terminal || got.TokensKnown {
		t.Fatalf("a live reading under a terminal status became final usage: %+v", got)
	}
}

func TestRunAccountingReportsLiveRunAsNotTerminal(t *testing.T) {
	started := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, StartedAt: &started}
	events := []*domain.RunEvent{{EventType: domain.EventTypeMetric, Timestamp: started, Data: &domain.UsageEventData{InputTokens: 70, OutputTokens: 30, Turns: 1}}}

	got := accountingFor(t, run, events)
	if got.Terminal || got.WallSeconds != 0 {
		t.Fatalf("running run projected as settled: %+v", got)
	}
}

func TestRunAccountingMeasuresExplicitSubscriptionZero(t *testing.T) {
	started := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(10 * time.Second)
	zero := int64(0)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusFailed, StartedAt: &started, EndedAt: &ended, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription}}
	events := []*domain.RunEvent{{EventType: domain.EventTypeMetric, Timestamp: started, Data: &domain.UsageEventData{InputTokens: 10, OutputTokens: 5, Turns: 1, ReconciliationAuthority: true, Charge: &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}}}}

	got := accountingFor(t, run, events)
	if !got.Terminal || !got.TokensKnown || !got.ChargeMeasured || got.ChargeMicroUSD != 0 || got.WallSeconds != 10 {
		t.Fatalf("explicit subscription zero charge = %+v", got)
	}
}

func TestRunAccountingCountsFinishedReviewTurnAsTerminal(t *testing.T) {
	started := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(5 * time.Second)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusNeedsReview, StartedAt: &started, EndedAt: &ended}

	got := accountingFor(t, run, nil)
	if !got.Terminal || got.WallSeconds != 5 {
		t.Fatalf("finished needs_review turn = %+v", got)
	}
}

func TestRunAccountingRejectsEndBeforeStart(t *testing.T) {
	started := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	ended := started.Add(-time.Second)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, StartedAt: &started, EndedAt: &ended}
	state, err := meteredWorkflowChildState(run, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runAccountingFromState(run, state); err == nil {
		t.Fatal("negative elapsed time accepted")
	}
}
