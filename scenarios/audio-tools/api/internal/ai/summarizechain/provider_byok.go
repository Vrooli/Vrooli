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
	dispatch *tiered.BYOKDispatcher[BYOKAdapter, Result]
}

func NewBYOKProvider(registry map[string]BYOKAdapter) *BYOKProvider {
	return &BYOKProvider{
		dispatch: tiered.NewBYOKDispatcher(registry, "summarizechain", ErrMissingBYOKProvider, ErrUnknownBYOKProvider, stampBYOKResult),
	}
}

func stampBYOKResult(result *Result, adapter BYOKAdapter) {
	tiered.StampIdentity(&result.Tier, TierBYOK, &result.ProviderID, &result.ModelID, adapter.ID(), adapter.Model(), true)
}

func (p *BYOKProvider) Type() ProviderTier { return TierBYOK }

func (p *BYOKProvider) IsAvailable(context.Context) bool {
	return p != nil && p.dispatch.Configured()
}

func (p *BYOKProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	return p.dispatch.Execute(req.BYOKKey, req.BYOKProvider,
		func(adapter BYOKAdapter) (*Result, error) { return adapter.Summarize(ctx, req.BYOKKey, req) })
}

func (p *BYOKProvider) Model() string { return tiered.DispatchedModel() }
