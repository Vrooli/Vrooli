package invocationreadmodel_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"

	"github.com/google/uuid"
)

// flatPricing prices every token at 1 µUSD.
type flatPricing struct{}

func (flatPricing) CalculateCost(_ context.Context, req codecs.PricingCostRequest) (*codecs.PricingCostCalculation, error) {
	total := float64(req.InputTokens+req.OutputTokens+req.CacheReadTokens+req.CacheCreationTokens) / 1_000_000
	return &codecs.PricingCostCalculation{TotalCostUSD: total, InputCostUSD: total, CostSource: "metered", Provider: "test", PricingFetchedAt: time.Now()}, nil
}

// F-W1-2: a metered codec-pipe Codex turn is charged once in the run's
// projection, from the real exec decoder's output.
func TestMeteredCodexExecTurnIsChargedOnce(t *testing.T) {
	codec, err := codecs.NewCodex(codecs.WithPricingService(flatPricing{}))
	if err != nil {
		t.Fatal(err)
	}
	runID := uuid.New()
	state := codec.NewState()
	var events []*domain.RunEvent
	for _, line := range []string{
		`{"type":"thread.started","thread_id":"01a1aaaa-0000-7000-8000-000000000001"}`,
		`{"type":"turn.started"}`,
		`{"type":"turn.completed","usage":{"input_tokens":1000,"cached_input_tokens":400,"output_tokens":100}}`,
	} {
		decoded, err := codec.DecodeStreamLine(state, runID, line)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, decoded...)
	}
	started := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	ended := started.Add(time.Minute)
	run := &domain.Run{ID: runID, CreatedAt: started, StartedAt: &started, EndedAt: &ended, Status: domain.RunStatusComplete, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "gpt-6-luna"}, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisMetered}}
	fact := invocationreadmodel.ProjectRun(run, events, ended)
	// 600 input + 400 cached + 100 output tokens at 1 µUSD each.
	if fact.MeteredChargeMicroUSD != 1100 {
		t.Fatalf("metered charge = %d µUSD, want 1100 (the turn's charge once)", fact.MeteredChargeMicroUSD)
	}
}
