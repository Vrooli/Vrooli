package summarizechain

import (
	"context"

	"audio-tools/internal/ai/chains/tiered"
)

// seam: BYOKAdapter is the summarize BYOK-adapter seam (SEAMS.md row
// "summarizechain.BYOKAdapter"). Production wires concrete vendor
// adapters (openrouter); tests wire fakes.
type BYOKAdapter interface {
	ID() string
	Summarize(ctx context.Context, key string, req Request) (*Result, error)
	IsAvailable(ctx context.Context, key string) bool
	Model() string
}

type BYOKProvider struct {
	registry map[string]BYOKAdapter
}

func NewBYOKProvider(registry map[string]BYOKAdapter) *BYOKProvider {
	return &BYOKProvider{registry: registry}
}

func (p *BYOKProvider) Type() ProviderTier { return TierBYOK }

func (p *BYOKProvider) IsAvailable(ctx context.Context) bool {
	return tiered.RegistryConfigured(p.registry)
}

func (p *BYOKProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	return tiered.ExecuteBYOK(p.registry, req.BYOKKey, req.BYOKProvider, "summarizechain", ErrMissingBYOKProvider, ErrUnknownBYOKProvider,
		func(adapter BYOKAdapter) (*Result, error) { return adapter.Summarize(ctx, req.BYOKKey, req) },
		func(result *Result, adapter BYOKAdapter) {
			result.Tier, result.ProviderID = TierBYOK, adapter.ID()
			if result.ModelID == "" {
				result.ModelID = adapter.Model()
			}
		})
}

func (p *BYOKProvider) Model() string { return tiered.DispatchedModel() }
