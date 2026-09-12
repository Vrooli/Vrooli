// Package wiring owns production composition of Agent Manager services.
package wiring

import (
	"context"
	"fmt"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/pricing"
)

// Runners contains the production registry and concrete runners that need
// follow-up composition (for example protected-sandbox launchers).
type Runners struct {
	Registry    *runner.DefaultRegistry
	Claude      *runnercore.Runner
	Codex       *runnercore.Runner
	OpenCode    *runnercore.Runner
	Grok        *runnercore.Runner
	Antigravity *runnercore.Runner
}

// NewRunners registers every supported coding-agent runner. Codec creation
// failures become unavailable stub runners so the API can start and accurately
// report the missing runtime instead of failing bootstrap.
func NewRunners(pricingServices ...codecs.PricingService) Runners {
	registry := runner.NewRegistry()
	hostLauncher := runner.NewHostLauncher()
	result := Runners{Registry: registry}
	var pricingService codecs.PricingService
	if len(pricingServices) > 0 {
		pricingService = pricingServices[0]
	}
	register := func(name string, runnerType domain.RunnerType, build func() (*runnercore.Runner, error), target **runnercore.Runner) {
		built, err := build()
		if err != nil {
			obs.Logger().Warn(name+" codec construction failed", obs.KeyRunnerType, string(runnerType), obs.KeyError, err.Error())
			if err := registry.Register(runner.NewStubRunner(runnerType, fmt.Sprintf("%s runner failed to initialize: %v", name, err))); err != nil {
				obs.Logger().Warn("stub "+name+" runner registration failed", obs.KeyRunnerType, string(runnerType), obs.KeyError, err.Error())
			}
			return
		}
		*target = built
		if err := registry.Register(built); err != nil {
			obs.Logger().Warn(name+" runner registration failed", obs.KeyRunnerType, string(runnerType), obs.KeyError, err.Error())
		}
		if available, message := built.IsAvailable(context.Background()); available {
			obs.Logger().Info("runner available", obs.KeyRunnerType, string(runnerType))
		} else {
			obs.Logger().Warn("runner unavailable", obs.KeyRunnerType, string(runnerType), obs.KeyMessage, message)
		}
	}
	register("Claude Code", domain.RunnerTypeClaudeCode, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewClaude()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Claude)
	register("Codex", domain.RunnerTypeCodex, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewCodex(codecs.WithPricingService(pricingService))
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Codex)
	register("OpenCode", domain.RunnerTypeOpenCode, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewOpenCode()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.OpenCode)
	register("Grok", domain.RunnerTypeGrok, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewGrok()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Grok)
	register("Antigravity", domain.RunnerTypeAntigravity, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewAntigravity()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Antigravity)
	return result
}

// pricingCodecAdapter keeps the runner codec seam narrower than the full
// pricing control surface. Production composition owns this translation so
// codecs never depend on pricing repositories or HTTP providers.
type pricingCodecAdapter struct {
	service pricing.Service
}

func (a pricingCodecAdapter) CalculateCost(ctx context.Context, req codecs.PricingCostRequest) (*codecs.PricingCostCalculation, error) {
	if a.service == nil {
		return nil, fmt.Errorf("pricing service is unavailable")
	}
	calc, err := a.service.CalculateCost(ctx, pricing.CostRequest{
		Model:               req.Model,
		RunnerType:          req.RunnerType,
		InputTokens:         req.InputTokens,
		OutputTokens:        req.OutputTokens,
		CacheReadTokens:     req.CacheReadTokens,
		CacheCreationTokens: req.CacheCreationTokens,
	})
	if err != nil || calc == nil {
		return nil, err
	}
	return &codecs.PricingCostCalculation{
		InputCostUSD:         calc.InputCostUSD,
		OutputCostUSD:        calc.OutputCostUSD,
		CacheReadCostUSD:     calc.CacheReadCostUSD,
		CacheCreationCostUSD: calc.CacheCreationCostUSD,
		TotalCostUSD:         calc.TotalCostUSD,
		CostSource:           chargeSourceFromComponents(calc.ComponentSources),
		Provider:             calc.Provider,
		CanonicalModel:       calc.CanonicalModel,
		ChargeReason:         calc.ChargeReason,
		PriceBookRevision:    calc.PriceBookRevision,
		ComponentSources:     pricingComponentSources(calc.ComponentSources),
		PricingFetchedAt:     calc.PricingFetchedAt,
		PricingVersion:       calc.PricingVersion,
	}, nil
}

func chargeSourceFromComponents(sources map[pricing.PricingComponent]pricing.PricingSource) string {
	if len(sources) == 0 {
		return "unpriced"
	}
	for _, source := range sources {
		switch source {
		case pricing.SourceManualOverride, pricing.SourceProviderAPI, pricing.SourceHistoricalAverage:
			return "metered"
		case pricing.PricingSource("subscription"):
			return "subscription"
		case pricing.PricingSource("local"):
			return "local"
		}
	}
	return "unknown"
}

func pricingComponentSources(sources map[pricing.PricingComponent]pricing.PricingSource) map[string]string {
	if len(sources) == 0 {
		return nil
	}
	result := make(map[string]string, len(sources))
	for component, source := range sources {
		result[string(component)] = string(source)
	}
	return result
}
