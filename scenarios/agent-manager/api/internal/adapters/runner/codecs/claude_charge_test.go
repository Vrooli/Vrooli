package codecs

import (
	"context"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestClaudeRetainedCostStateRequiresExplicitZeroEvidence(t *testing.T) {
	const explicit = `{"type":"cost-state","sessionId":"original-session","totalCostUSD":0,"totalAPIDuration":0,"totalAPIDurationWithoutRetries":0,"modelUsage":{},"hasUnknownModelCost":false}`
	for _, tc := range []struct {
		name, line string
		known      bool
	}{
		{"explicit-zero", explicit, true},
		{"missing-cost", strings.Replace(explicit, `"totalCostUSD":0,`, "", 1), false},
		{"missing-duration", strings.Replace(explicit, `"totalAPIDuration":0,`, "", 1), false},
		{"unknown-cost", strings.Replace(explicit, `"hasUnknownModelCost":false`, `"hasUnknownModelCost":true`, 1), false},
		{"missing-model-usage", strings.Replace(explicit, `"modelUsage":{},`, "", 1), false},
		{"nonempty-model-usage", strings.Replace(explicit, `"modelUsage":{}`, `"modelUsage":{"unknown":{}}`, 1), false},
		{"api-work", strings.Replace(explicit, `"totalAPIDuration":0`, `"totalAPIDuration":10`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewClaudeForTest().NewTranscriptParser()
			parser.(runner.TranscriptBillingSetter).SetTranscriptBilling(domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription})
			result := parser.ParseTranscriptLine(uuid.New(), tc.line)
			var receipt *domain.UsageEventData
			for _, event := range result.Events {
				if usage, ok := event.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority {
					receipt = usage
				}
			}
			if (receipt != nil) != tc.known {
				t.Fatalf("known=%v receipt=%+v err=%v", tc.known, receipt, result.Err)
			}
			if result.Terminal != nil {
				t.Fatal("cost snapshot terminated an interactive session")
			}
			if receipt != nil && (receipt.InputTokens != 0 || receipt.OutputTokens != 0 || receipt.Charge == nil || receipt.Charge.AmountMicroUSD == nil || *receipt.Charge.AmountMicroUSD != 0) {
				t.Fatalf("zero receipt changed: %+v", receipt)
			}
		})
	}
}

type fakeClaudePricing struct{ totalUSD float64 }

func (f fakeClaudePricing) CalculateCost(context.Context, PricingCostRequest) (*PricingCostCalculation, error) {
	return &PricingCostCalculation{TotalCostUSD: f.totalUSD, CostSource: "test", ChargeReason: "test"}, nil
}

// TestClaudeInteractiveUsageEmitsChargeWhenPriced proves that an interactive
// (on-disk) Claude turn with usage but no cost line produces a metered charge
// from the pricing lookup, so the run is not reported as null-as-zero.
func TestClaudeInteractiveUsageEmitsChargeWhenPriced(t *testing.T) {
	c := NewClaudeForTest()
	c.pricingService = fakeClaudePricing{totalUSD: 1.5}
	parser := c.NewTranscriptParser()
	parser.(runner.TranscriptModelSetter).SetTranscriptModel("claude-haiku-4-5")

	line := `{"type":"assistant","sessionId":"s1","message":{"model":"claude-haiku-4-5","role":"assistant","stop_reason":"end_turn","usage":{"input_tokens":1000,"output_tokens":500}}}`
	res := parser.ParseTranscriptLine(uuid.New(), line)

	var charge *domain.ChargeEventData
	for _, event := range res.Events {
		if data, ok := event.Data.(*domain.ChargeEventData); ok {
			charge = data
		}
	}
	if charge == nil || charge.AmountMicroUSD == nil || *charge.AmountMicroUSD != 1_500_000 {
		t.Fatalf("expected a metered charge of 1.5 USD, got %+v", charge)
	}
}

// TestClaudeInteractiveUsageChargeUnknownWithoutPricing proves that without a
// pricing lookup the interactive turn emits no charge event, so the accounting
// stays unknown rather than claiming a zero charge.
func TestClaudeInteractiveUsageChargeUnknownWithoutPricing(t *testing.T) {
	c := NewClaudeForTest()
	parser := c.NewTranscriptParser()
	parser.(runner.TranscriptModelSetter).SetTranscriptModel("claude-haiku-4-5")
	line := `{"type":"assistant","sessionId":"s1","message":{"model":"claude-haiku-4-5","role":"assistant","stop_reason":"end_turn","usage":{"input_tokens":1000,"output_tokens":500}}}`
	res := parser.ParseTranscriptLine(uuid.New(), line)
	for _, event := range res.Events {
		if _, ok := event.Data.(*domain.ChargeEventData); ok {
			t.Fatalf("expected no charge event without pricing, got %+v", event.Data)
		}
	}
}
