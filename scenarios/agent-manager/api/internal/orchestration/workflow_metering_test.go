package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowruntime"
	"github.com/google/uuid"
)

func TestWorkflowTerminalReceiptReconcilesEarlierUnknownCharge(t *testing.T) {
	zero := int64(0)
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusFailed, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription}}
	events := []*domain.RunEvent{
		{Data: &domain.UsageEventData{InputTokens: 4331, OutputTokens: 205, CacheReadTokens: 10880}},
		{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisUnpriced}},
		{Data: &domain.UsageEventData{InputTokens: 8906, OutputTokens: 415, CacheReadTokens: 21760}},
		{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisUnpriced}},
		{Data: &domain.UsageEventData{InputTokens: 8906, OutputTokens: 415, CacheReadTokens: 21760, Turns: 1, ReconciliationAuthority: true, Charge: &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}}},
	}
	got, err := meteredWorkflowChildState(run, events, time.Now())
	if err != nil || !got.TokensKnown || !got.ChargeMeasured || got.Tokens != 31081 || got.Turns != 1 || got.ChargeMicroUSD != 0 {
		t.Fatalf("original terminal receipt failed to reconcile: %+v %v", got, err)
	}
	// Reconciliation is a read projection; old unpriced observations survive.
	if len(events) != 5 || events[1].Data.(*domain.ChargeEventData).Basis != domain.ChargeBasisUnpriced {
		t.Fatal("accounting history rewritten")
	}
}

func TestWorkflowTerminalReceiptPreservesLaterEvidence(t *testing.T) {
	zero, positive, negative := int64(0), int64(7), int64(-1)
	for _, tc := range []struct {
		name        string
		later       *domain.RunEvent
		tokensKnown bool
		chargeKnown bool
		charge      int64
		invalid     bool
	}{
		{name: "later-usage", later: &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 5}}, chargeKnown: true},
		{name: "later-unpriced-charge", later: &domain.RunEvent{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisUnpriced}}, tokensKnown: true},
		{name: "later-metered-charge", later: &domain.RunEvent{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &positive}}, tokensKnown: true, chargeKnown: true, charge: 7},
		{name: "later-invalid-usage", later: &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: -1}}, invalid: true},
		{name: "later-invalid-charge", later: &domain.RunEvent{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &negative}}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusCancelled, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription}}
			receipt := &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: 10, ReconciliationAuthority: true, Charge: &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}}}
			earlier := &domain.RunEvent{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisUnpriced}}
			events := []*domain.RunEvent{earlier, receipt, tc.later}
			projected := terminalWorkflowReceiptProjection(events)
			if len(projected) != 2 || projected[0] != receipt || projected[1] != tc.later {
				t.Errorf("receipt must suppress only earlier evidence, retaining its later tail: %+v", projected)
			}
			got, err := meteredWorkflowChildState(run, events, time.Now())
			if tc.invalid {
				if err == nil {
					t.Fatal("receipt hid invalid later accounting")
				}
			} else if err != nil || got.TokensKnown != tc.tokensKnown || got.MeterCadence.TerminalAuthority != tc.tokensKnown || got.ChargeMeasured != tc.chargeKnown || got.ChargeMicroUSD != tc.charge {
				t.Errorf("receipt concealed later accounting: %+v %v", got, err)
			}
			if events[0] != earlier || events[1] != receipt || events[2] != tc.later || !receipt.Data.(*domain.UsageEventData).ReconciliationAuthority {
				t.Fatal("projection mutated durable accounting evidence")
			}
		})
	}
}

func TestWorkflowMeterReadsDurableUsageAndReconcilesTerminalAuthority(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning}
	base := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	usage := &domain.RunEvent{EventType: domain.EventTypeMetric, Timestamp: base, Data: &domain.UsageEventData{InputTokens: 70, OutputTokens: 30, Turns: 1, TurnIndex: 1}}
	second := &domain.RunEvent{EventType: domain.EventTypeMetric, Timestamp: base.Add(2 * time.Second), Data: &domain.UsageEventData{InputTokens: 70, OutputTokens: 30, Turns: 1, TurnIndex: 1}}
	events := []*domain.RunEvent{usage, second}
	state, err := meteredWorkflowChildState(run, events, time.Now())
	if err != nil || state.Terminal || !state.TokensKnown || state.Tokens != 100 {
		t.Fatalf("live meter: %+v %v", state, err)
	}
	if state.MeterCadence.Samples != 2 || state.MeterCadence.MeanIntervalSeconds != 2 || state.MeterCadence.MaxIntervalSeconds != 2 {
		t.Fatalf("meter cadence: %+v", state.MeterCadence)
	}
	run.Status = domain.RunStatusCancelled
	state, err = meteredWorkflowChildState(run, events, time.Now())
	if err != nil || !state.Terminal || state.TokensKnown {
		t.Fatal("cancelled status and a live reading became final usage")
	}
	events = append(events, &domain.RunEvent{EventType: domain.EventTypeMetric, Timestamp: base.Add(5 * time.Second), Data: &domain.UsageEventData{InputTokens: 80, OutputTokens: 43, ReconciliationAuthority: true}})
	state, err = meteredWorkflowChildState(run, events, time.Now())
	if err != nil || !state.Terminal || !state.Failed || state.Tokens != 123 {
		t.Fatalf("terminal reconciliation: %+v %v", state, err)
	}
	state, err = meteredWorkflowChildState(run, nil, time.Now())
	if err != nil || state.TokensKnown {
		t.Fatal("missing provider meter became known zero")
	}
	_, err = meteredWorkflowChildState(run, []*domain.RunEvent{{Data: &domain.UsageEventData{InputTokens: -1}}}, time.Now())
	if err == nil {
		t.Fatal("negative provider usage accepted")
	}
}

func TestMeterCadenceClampsOutOfOrderObservationsAndDisclosesZeroSamples(t *testing.T) {
	if got := workflowruntime.MeterCadenceFromObservations(nil); got.Samples != 0 || got.MeanIntervalSeconds != 0 {
		t.Fatalf("empty cadence: %+v", got)
	}
	base := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	got := workflowruntime.MeterCadenceFromObservations([]time.Time{base, base.Add(-time.Second), base.Add(3 * time.Second)})
	if got.Samples != 3 || got.MeanIntervalSeconds != 2 || got.MaxIntervalSeconds != 4 {
		t.Fatalf("out-of-order cadence: %+v", got)
	}
}

func TestWorkflowReceiptDistinguishesKnownZeroFromUnknownCharge(t *testing.T) {
	zero, positive := int64(0), int64(7)
	for _, tc := range []struct {
		name     string
		snapshot domain.ChargeBasis
		charge   *domain.ChargeEventData
		known    bool
	}{
		{"subscription", domain.ChargeBasisSubscription, &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}, true},
		{"local", domain.ChargeBasisLocal, &domain.ChargeEventData{Basis: domain.ChargeBasisLocal, AmountMicroUSD: &zero}, true},
		{"metered-zero", domain.ChargeBasisMetered, &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &zero}, true},
		{"metered-positive", domain.ChargeBasisMetered, &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &positive}, true},
		{"missing-charge", domain.ChargeBasisSubscription, nil, false},
		{"missing-snapshot", domain.ChargeBasisUnknown, &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}, false},
		{"missing-amount", domain.ChargeBasisMetered, &domain.ChargeEventData{Basis: domain.ChargeBasisMetered}, false},
		{"unpriced", domain.ChargeBasisSubscription, &domain.ChargeEventData{Basis: domain.ChargeBasisUnpriced}, false},
		{"unknown-zero", domain.ChargeBasisUnknown, &domain.ChargeEventData{Basis: domain.ChargeBasisUnknown, AmountMicroUSD: &zero}, false},
		{"conflicting-basis", domain.ChargeBasisLocal, &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}, false},
		{"subscription-nonzero", domain.ChargeBasisSubscription, &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &positive}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, Billing: domain.BillingSnapshot{Basis: tc.snapshot}, Summary: &domain.RunSummary{CostEstimate: 123}}
			events := []*domain.RunEvent{{Data: &domain.UsageEventData{InputTokens: 21, ReconciliationAuthority: true}}}
			if tc.charge != nil {
				events = append(events, &domain.RunEvent{Data: tc.charge})
			}
			state, err := meteredWorkflowChildState(run, events, time.Now())
			if err != nil || state.ChargeMeasured != tc.known || !state.TokensKnown || state.Tokens != 21 {
				t.Fatalf("receipt %+v: %v", state, err)
			}
			if tc.known && tc.charge.Basis != domain.ChargeBasisMetered && state.ChargeMicroUSD != 0 {
				t.Fatalf("known zero changed: %+v", state)
			}
		})
	}
}

func TestContinuationReceiptRequiresLatestInvocationAuthority(t *testing.T) {
	for _, correction := range []int{5, 10} {
		id := uuid.New()
		run := &domain.Run{ID: id, Status: domain.RunStatusCancelled, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription}}
		zero := int64(0)
		usage := func(tokens int, authority bool) *domain.RunEvent {
			return &domain.RunEvent{Data: &domain.UsageEventData{InputTokens: tokens, Turns: 1, TurnIndex: 1, ReconciliationAuthority: authority}}
		}
		charge := func() *domain.RunEvent {
			return &domain.RunEvent{Data: &domain.ChargeEventData{Basis: domain.ChargeBasisSubscription, AmountMicroUSD: &zero}}
		}
		events := []*domain.RunEvent{domain.NewStatusEvent(id, "starting", "running", "execute"), usage(10, true), charge(), domain.NewStatusEvent(id, "complete", "running", "Continuation requested")}
		state, err := meteredWorkflowChildState(run, events, time.Now())
		if err != nil || state.TokensKnown || state.ChargeMeasured || state.MeterCadence.TerminalAuthority {
			t.Fatalf("earlier terminal receipt settled killed correction: %+v %v", state, err)
		}
		events = append(events, usage(correction, false))
		state, err = meteredWorkflowChildState(run, events, time.Now())
		if err != nil || state.TokensKnown || state.Tokens != 10+correction {
			t.Fatalf("partial correction discarded or treated final: %+v %v", state, err)
		}
		terminal := usage(correction, true)
		events = append(events, terminal, terminal, charge()) // repeated same invocation receipt
		state, err = meteredWorkflowChildState(run, events, time.Now())
		if err != nil || !state.TokensKnown || !state.ChargeMeasured || state.Tokens != 10+correction {
			t.Fatalf("continuation receipt lost/double counted: %+v %v", state, err)
		}
	}
}
