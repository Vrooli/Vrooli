// Package codecs — pricing.go is the runner-agnostic pricing seam.
//
// A codec whose CLI reports token usage but NOT a dollar cost (codex today;
// any future runner with the same shape) builds its usage event here. Charge
// is a separate optional event, so pricing failure cannot erase consumption.
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (Pricing seam).
package codecs

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

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
	ChargeReason         string
	ComponentSources     map[string]string
	PricingFetchedAt     time.Time
	PricingVersion       string
	PriceBookRevision    string
}

// usageTokens carries the token counts a runner reported for one cost event.
type usageTokens struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
}

// buildCostEvents always returns usage and returns a charge when pricing
// resolves. The two events intentionally share the metric event type but have
// explicit payload discriminators.
func buildCostEvents(runID uuid.UUID, runnerType domain.RunnerType, pricing PricingService, model string, tokens usageTokens, billing ...domain.BillingSnapshot) []*domain.RunEvent {
	modelWasBlank := strings.TrimSpace(model) == ""
	if strings.TrimSpace(model) == "" {
		model = "unknown"
	}
	if modelWasBlank {
		obs.Component("runner-codec").Warn("usage event has no model", obs.KeyRunID, runID.String(), obs.KeyRunnerType, string(runnerType))
	}
	usageData := &domain.UsageEventData{
		PayloadKind:         domain.PayloadKindUsage,
		InputTokens:         tokens.InputTokens,
		OutputTokens:        tokens.OutputTokens,
		CacheReadTokens:     tokens.CacheReadTokens,
		CacheCreationTokens: tokens.CacheCreationTokens,
		Model:               model,
		RunnerType:          string(runnerType),
	}
	usageEvent := &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now(), Data: usageData}
	events := []*domain.RunEvent{usageEvent}

	if pricing == nil {
		return events
	}
	basis := domain.ChargeBasisMetered
	if len(billing) > 0 && billing[0].Mode != "" {
		basis = billing[0].EffectiveBasis()
	}
	if basis == domain.ChargeBasisSubscription || basis == domain.ChargeBasisLocal {
		zero := int64(0)
		return append(events, &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now(), Data: &domain.ChargeEventData{PayloadKind: domain.PayloadKindCharge, Basis: basis, AmountMicroUSD: &zero, Currency: "USD", Model: model, RunnerType: string(runnerType)}})
	}
	if basis != domain.ChargeBasisMetered {
		return append(events, &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now(), Data: &domain.ChargeEventData{PayloadKind: domain.PayloadKindCharge, Basis: basis, Currency: "USD", Model: model, RunnerType: string(runnerType), ChargeReason: "billing_basis_unknown"}})
	}

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
	basis = domain.ChargeBasisUnpriced
	chargeReason := "model_unpriced"
	if modelWasBlank {
		chargeReason = "model_unlabelled"
	}
	if err == nil && calc != nil {
		basis = chargeBasisForCalculation(calc)
		chargeReason = calc.ChargeReason
		if chargeReason == "" && basis == domain.ChargeBasisUnpriced {
			chargeReason = "model_unpriced"
		}
	}
	charge := &domain.ChargeEventData{
		PayloadKind:  domain.PayloadKindCharge,
		Basis:        basis,
		Currency:     "USD",
		Model:        model,
		RunnerType:   string(runnerType),
		ChargeReason: chargeReason,
	}
	if basis != domain.ChargeBasisUnpriced && calc != nil {
		amount := int64(calc.TotalCostUSD*1_000_000 + 0.5)
		charge.AmountMicroUSD = &amount
	}
	events = append(events, &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeMetric, Timestamp: time.Now(), Data: charge})

	return events
}

func markUsageTurn(events []*domain.RunEvent, turnIndex int) []*domain.RunEvent {
	for _, event := range events {
		if event == nil {
			continue
		}
		if usage, ok := event.Data.(*domain.UsageEventData); ok {
			usage.TurnIndex = turnIndex
		}
	}
	return events
}

func chargeBasisForCostSource(source string) domain.ChargeBasis {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "metered", "provider_api", "manual_override", "historical_average", "pricing_table_estimate":
		return domain.ChargeBasisMetered
	case "subscription":
		return domain.ChargeBasisSubscription
	case "local":
		return domain.ChargeBasisLocal
	case "unpriced":
		return domain.ChargeBasisUnpriced
	default:
		return domain.ChargeBasisUnknown
	}
}

func chargeBasisForCalculation(calc *PricingCostCalculation) domain.ChargeBasis {
	if len(calc.ComponentSources) == 0 {
		return chargeBasisForCostSource(calc.CostSource)
	}
	for _, source := range calc.ComponentSources {
		switch strings.ToLower(strings.TrimSpace(source)) {
		case "manual_override", "provider_api", "historical_average", "metered":
			return domain.ChargeBasisMetered
		case "subscription":
			return domain.ChargeBasisSubscription
		case "local":
			return domain.ChargeBasisLocal
		}
	}
	return domain.ChargeBasisUnknown
}
