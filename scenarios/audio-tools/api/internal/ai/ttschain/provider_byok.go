package ttschain

import (
	"context"

	"audio-tools/internal/ai/chains/tiered"
)

// seam: BYOKAdapter is the TTS BYOK-adapter seam (SEAMS.md row
// "ttschain.BYOKAdapter"). Production wires concrete vendor adapters
// (openai-tts, elevenlabs, ...); tests wire fakes.
type BYOKAdapter interface {
	ID() string
	Synthesize(ctx context.Context, key string, req Request) (*Result, error)
	IsAvailable(ctx context.Context, key string) bool
	Model() string

	StreamingCapability() bool
	SynthesizeStreaming(ctx context.Context, key string, req Request) (<-chan AudioFrame, error)
}

type BYOKProvider struct {
	registry map[string]BYOKAdapter
	dispatch *tiered.BYOKDispatcher[BYOKAdapter, Result]
}

func NewBYOKProvider(registry map[string]BYOKAdapter) *BYOKProvider {
	return &BYOKProvider{
		registry: registry,
		dispatch: tiered.NewBYOKDispatcher(registry, "ttschain", ErrMissingBYOKProvider, ErrUnknownBYOKProvider, stampBYOKResult),
	}
}

func stampBYOKResult(result *Result, adapter BYOKAdapter) {
	tiered.StampIdentity(&result.Tier, TierBYOK, &result.ProviderID, &result.ModelID, adapter.ID(), adapter.Model(), true)
}

func (p *BYOKProvider) Type() ProviderTier { return TierBYOK }

func (p *BYOKProvider) IsAvailable(context.Context) bool {
	return p.AdapterConfigured()
}

// AdapterConfigured reports that at least one BYOK adapter is registered.
// This is separate from credential presence and endpoint usability.
func (p *BYOKProvider) AdapterConfigured() bool {
	return p != nil && p.dispatch != nil && p.dispatch.Configured()
}

func (p *BYOKProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	return p.dispatch.Execute(req.BYOKKey, req.BYOKProvider,
		func(adapter BYOKAdapter) (*Result, error) { return adapter.Synthesize(ctx, req.BYOKKey, req) })
}

func (p *BYOKProvider) Model() string { return tiered.DispatchedModel() }

func (p *BYOKProvider) StreamingCapability() bool {
	if p == nil {
		return false
	}
	for _, a := range p.registry {
		if a != nil && a.StreamingCapability() {
			return true
		}
	}
	return false
}

func (p *BYOKProvider) SynthesizeStreaming(ctx context.Context, req Request) (<-chan AudioFrame, error) {
	adapter, err := p.dispatch.Resolve(req.BYOKKey, req.BYOKProvider)
	if err != nil {
		return nil, err
	}
	if !adapter.StreamingCapability() {
		return nil, nil
	}
	return adapter.SynthesizeStreaming(ctx, req.BYOKKey, req)
}
