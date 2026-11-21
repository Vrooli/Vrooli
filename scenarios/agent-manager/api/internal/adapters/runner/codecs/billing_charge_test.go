package codecs

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// chargeFromEvents returns the single charge event and its payload.
func chargeFromEvents(t *testing.T, events []*domain.RunEvent) *domain.ChargeEventData {
	t.Helper()
	var charge *domain.ChargeEventData
	for _, e := range events {
		if c, ok := e.Data.(*domain.ChargeEventData); ok {
			if charge != nil {
				t.Fatalf("expected exactly one charge event, found more")
			}
			charge = c
		}
	}
	if charge == nil {
		t.Fatalf("expected a charge event, found none")
	}
	return charge
}

func assertCharge(t *testing.T, charge *domain.ChargeEventData, wantBasis domain.ChargeBasis, wantMicro int64) {
	t.Helper()
	if charge.Basis != wantBasis {
		t.Errorf("charge basis = %q, want %q", charge.Basis, wantBasis)
	}
	if charge.AmountMicroUSD == nil {
		t.Fatalf("charge amount is nil, want %d", wantMicro)
	}
	if *charge.AmountMicroUSD != wantMicro {
		t.Errorf("charge amount = %d micro-USD, want %d", *charge.AmountMicroUSD, wantMicro)
	}
}

// A run's immutable billing snapshot must decide the charge basis. A
// subscription or local run records an explicit zero charge; only metered
// runs record the runner-reported cost.
func TestClaudeResultChargeHonorsRunBilling(t *testing.T) {
	cases := []struct {
		name      string
		billing   domain.BillingSnapshot
		wantBasis domain.ChargeBasis
		wantMicro int64
	}{
		{"metered", domain.BillingSnapshot{Mode: domain.BillingModeMetered}, domain.ChargeBasisMetered, 84249},
		{"subscription", domain.BillingSnapshot{Mode: domain.BillingModeSubscription}, domain.ChargeBasisSubscription, 0},
		{"local", domain.BillingSnapshot{Mode: domain.BillingModeLocal}, domain.ChargeBasisLocal, 0},
		{"unset_defaults_metered", domain.BillingSnapshot{}, domain.ChargeBasisMetered, 84249},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClaudeForTest()
			state := c.NewState().(*claudeState)
			state.SetStateBilling(tc.billing)
			events, err := c.DecodeStreamLine(state, uuid.New(), claudeCodeSamples["result_success"])
			if err != nil {
				t.Fatalf("DecodeStreamLine error: %v", err)
			}
			assertCharge(t, chargeFromEvents(t, events), tc.wantBasis, tc.wantMicro)
		})
	}
}

// The durable-transcript replay path must honor the same snapshot through the
// transcript billing setter (it never replays BuildArgs).
func TestClaudeTranscriptChargeHonorsRunBilling(t *testing.T) {
	c := NewClaudeForTest()
	parser := c.NewTranscriptParser()
	setter, ok := parser.(interface {
		SetTranscriptBilling(domain.BillingSnapshot)
	})
	if !ok {
		t.Fatalf("claude transcript parser does not implement SetTranscriptBilling")
	}
	setter.SetTranscriptBilling(domain.BillingSnapshot{Mode: domain.BillingModeSubscription})
	result := parser.ParseTranscriptLine(uuid.New(), claudeCodeSamples["result_success"])
	if result.Err != nil {
		t.Fatalf("ParseTranscriptLine error: %v", result.Err)
	}
	assertCharge(t, chargeFromEvents(t, result.Events), domain.ChargeBasisSubscription, 0)
}

func TestOpenCodeStepFinishChargeHonorsRunBilling(t *testing.T) {
	cases := []struct {
		name      string
		billing   domain.BillingSnapshot
		wantBasis domain.ChargeBasis
		wantMicro int64
	}{
		{"metered", domain.BillingSnapshot{Mode: domain.BillingModeMetered}, domain.ChargeBasisMetered, 4200},
		{"subscription", domain.BillingSnapshot{Mode: domain.BillingModeSubscription}, domain.ChargeBasisSubscription, 0},
		{"local", domain.BillingSnapshot{Mode: domain.BillingModeLocal}, domain.ChargeBasisLocal, 0},
		{"unset_defaults_metered", domain.BillingSnapshot{}, domain.ChargeBasisMetered, 4200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewOpenCodeForTest()
			state := c.NewState().(*opencodeState)
			state.SetStateBilling(tc.billing)
			events, err := c.DecodeStreamLine(state, uuid.New(), opencodeSamples["step_finish_terminal"])
			if err != nil {
				t.Fatalf("DecodeStreamLine error: %v", err)
			}
			assertCharge(t, chargeFromEvents(t, events), tc.wantBasis, tc.wantMicro)
		})
	}
}

func TestOpenCodeTranscriptChargeHonorsRunBilling(t *testing.T) {
	c := NewOpenCodeForTest()
	parser := c.NewTranscriptParser()
	setter, ok := parser.(interface {
		SetTranscriptBilling(domain.BillingSnapshot)
	})
	if !ok {
		t.Fatalf("opencode transcript parser does not implement SetTranscriptBilling")
	}
	setter.SetTranscriptBilling(domain.BillingSnapshot{Mode: domain.BillingModeSubscription})
	result := parser.ParseTranscriptLine(uuid.New(), opencodeSamples["step_finish_terminal"])
	if result.Err != nil {
		t.Fatalf("ParseTranscriptLine error: %v", result.Err)
	}
	assertCharge(t, chargeFromEvents(t, result.Events), domain.ChargeBasisSubscription, 0)
}
