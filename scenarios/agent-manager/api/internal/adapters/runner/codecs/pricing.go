// Package codecs — pricing.go is the runner-agnostic pricing seam.
//
// A codec whose CLI reports token usage but NOT a dollar cost (codex today;
// any future runner with the same shape) builds its cost event here. The seam
// is parameterised by RunnerType, so promoting it out of the codex codec —
// where it used to live — lets every codec share one cost-event builder rather
// than cloning codex's copy. Codecs whose CLI reports cost natively (Claude
// Code, OpenCode) build their CostEventData directly and do not use this seam.
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (Pricing seam).
package codecs

import (
	"context"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// PricingService is the optional pricing-lookup hook used when emitting cost
// events. main.go wires the concrete implementation; codecs that don't need
// pricing leave it nil and the cost-event path simply omits dollar amounts.
type PricingService interface {
	CalculateCost(ctx context.Context, req PricingCostRequest) (*PricingCostCalculation, error)
}

// PricingCostRequest carries the inputs to a pricing lookup.
type PricingCostRequest struct {
	Model               string
	RunnerType          string
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
}

// PricingCostCalculation carries the pricing service's output.
type PricingCostCalculation struct {
	InputCostUSD         float64
	OutputCostUSD        float64
	CacheReadCostUSD     float64
	CacheCreationCostUSD float64
	TotalCostUSD         float64
	CostSource           string
	Provider             string
	CanonicalModel       string
	PricingFetchedAt     time.Time
	PricingVersion       string
}

// usageTokens carries the token counts a runner reported for one cost event.
type usageTokens struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
}

// buildCostEvent constructs a CostEventData metric event from reported token
// usage, optionally enriching it with pricing-service data when pricing is
// non-nil. It is the single place a runner-agnostic cost event is built:
// without pricing the event carries token counts and CostSourceUnknown; with
// pricing it carries the resolved dollar breakdown and provenance.
func buildCostEvent(runID uuid.UUID, runnerType domain.RunnerType, pricing PricingService, model string, tokens usageTokens) *domain.RunEvent {
	costData := &domain.CostEventData{
		InputTokens:         tokens.InputTokens,
		OutputTokens:        tokens.OutputTokens,
		CacheReadTokens:     tokens.CacheReadTokens,
		CacheCreationTokens: tokens.CacheCreationTokens,
		Model:               model,
		CostSource:          domain.CostSourceUnknown,
	}

	if pricing != nil {
		ctx, cancel := context.WithTimeout(context.Background(), config.DefaultLevers().Runners.ProbeTimeout)
		defer cancel()

		calc, err := pricing.CalculateCost(ctx, PricingCostRequest{
			Model:               model,
			RunnerType:          string(runnerType),
			InputTokens:         tokens.InputTokens,
			OutputTokens:        tokens.OutputTokens,
			CacheReadTokens:     tokens.CacheReadTokens,
			CacheCreationTokens: tokens.CacheCreationTokens,
		})
		if err == nil && calc != nil {
			costData.InputCostUSD = calc.InputCostUSD
			costData.OutputCostUSD = calc.OutputCostUSD
			costData.CacheReadCostUSD = calc.CacheReadCostUSD
			costData.CacheCreationCostUSD = calc.CacheCreationCostUSD
			costData.TotalCostUSD = calc.TotalCostUSD
			costData.CostSource = calc.CostSource
			costData.PricingProvider = calc.Provider
			costData.PricingModel = calc.CanonicalModel
			if !calc.PricingFetchedAt.IsZero() {
				costData.PricingFetchedAt = &calc.PricingFetchedAt
			}
			costData.PricingVersion = calc.PricingVersion
		}
	}

	return &domain.RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: domain.EventTypeMetric,
		Timestamp: time.Now(),
		Data:      costData,
	}
}
