package sttchain

import (
	"context"
	"fmt"
)

// BYOKAdapter is implemented by every BYOK STT adapter (openai-whisper,
// deepgram, ...). Registered in internal/byok with provider_id keys.
type BYOKAdapter interface {
	ID() string
	Transcribe(ctx context.Context, key string, req Request) (*Result, error)
	IsAvailable(ctx context.Context, key string) bool
	Model() string
}

// BYOKProvider routes per-request to the adapter named by req.BYOKProvider.
type BYOKProvider struct {
	registry map[string]BYOKAdapter
}

func NewBYOKProvider(registry map[string]BYOKAdapter) *BYOKProvider {
	return &BYOKProvider{registry: registry}
}

func (p *BYOKProvider) Type() ProviderTier { return TierBYOK }

func (p *BYOKProvider) IsAvailable(ctx context.Context) bool {
	return len(p.registry) > 0
}

func (p *BYOKProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	if req.BYOKKey == "" {
		return nil, fmt.Errorf("audio-tools/sttchain: BYOK key required")
	}
	if req.BYOKProvider == "" {
		return nil, ErrMissingBYOKProvider
	}
	adapter, ok := p.registry[req.BYOKProvider]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBYOKProvider, req.BYOKProvider)
	}
	res, err := adapter.Transcribe(ctx, req.BYOKKey, req)
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
