package ttschain

import (
	"context"
	"fmt"
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
	if req.BYOKKey == "" {
		return nil, fmt.Errorf("audio-tools/ttschain: BYOK key required")
	}
	if req.BYOKProvider == "" {
		return nil, ErrMissingBYOKProvider
	}
	adapter, ok := p.registry[req.BYOKProvider]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBYOKProvider, req.BYOKProvider)
	}
	res, err := adapter.Synthesize(ctx, req.BYOKKey, req)
	if err != nil {
		return nil, err
	}
	res.Tier = TierBYOK
	res.ProviderID = adapter.ID()
	if res.ModelID == "" {
		res.ModelID = adapter.Model()
	}
	return res, nil
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
	if req.BYOKKey == "" {
		return nil, fmt.Errorf("audio-tools/ttschain: BYOK key required")
	}
	if req.BYOKProvider == "" {
		return nil, ErrMissingBYOKProvider
	}
	adapter, ok := p.registry[req.BYOKProvider]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBYOKProvider, req.BYOKProvider)
	}
	if !adapter.StreamingCapability() {
		return nil, nil
	}
	return adapter.SynthesizeStreaming(ctx, req.BYOKKey, req)
}
