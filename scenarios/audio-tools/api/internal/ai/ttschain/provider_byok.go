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
}

func NewBYOKProvider(registry map[string]BYOKAdapter) *BYOKProvider {
	return &BYOKProvider{registry: registry}
}

func (p *BYOKProvider) Type() ProviderTier { return TierBYOK }

func (p *BYOKProvider) IsAvailable(ctx context.Context) bool { return len(p.registry) > 0 }

func (p *BYOKProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	return tiered.ExecuteBYOK(p.registry, req.BYOKKey, req.BYOKProvider, "ttschain", ErrMissingBYOKProvider, ErrUnknownBYOKProvider,
		func(adapter BYOKAdapter) (*Result, error) { return adapter.Synthesize(ctx, req.BYOKKey, req) },
		func(result *Result, adapter BYOKAdapter) {
			result.Tier, result.ProviderID = TierBYOK, adapter.ID()
			if result.ModelID == "" {
				result.ModelID = adapter.Model()
			}
		})
}

func (p *BYOKProvider) Model() string { return "byok-dispatched" }

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
	adapter, err := tiered.ResolveBYOKAdapter(p.registry, req.BYOKKey, req.BYOKProvider, "ttschain", ErrMissingBYOKProvider, ErrUnknownBYOKProvider)
	if err != nil {
		return nil, err
	}
	if !adapter.StreamingCapability() {
		return nil, nil
	}
	return adapter.SynthesizeStreaming(ctx, req.BYOKKey, req)
}
