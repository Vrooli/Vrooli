package orchestration

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowruntime"
	"github.com/google/uuid"
)

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
