package sttchain

import (
	"context"

	"audio-tools/internal/ai/chains/tiered"
)

// seam: BYOKAdapter is the STT BYOK-adapter seam (SEAMS.md row
// "sttchain.BYOKAdapter"). Production wires concrete vendor adapters
// from internal/byok; tests wire fakes.
//
// BYOKAdapter is implemented by every BYOK STT adapter (openai-whisper,
// deepgram, ...). Registered in internal/byok with provider_id keys.
type BYOKAdapter interface {
	ID() string
	Transcribe(ctx context.Context, key string, req Request) (*Result, error)
	IsAvailable(ctx context.Context, key string) bool
	Model() string

	// StreamingCapability reports whether the adapter can stream natively
	// (e.g. Deepgram WS, OpenAI Realtime). Adapters that return false are
	// skipped during stream-start negotiation.
	StreamingCapability() bool

	// TranscribeStreaming opens a streaming session. The contract mirrors
	// Provider.TranscribeStreaming. Adapters returning StreamingCapability=
	// false must return (nil, nil) here.
	TranscribeStreaming(ctx context.Context, key string, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error)
}

// BYOKProvider routes per-request to the adapter named by req.BYOKProvider.
type BYOKProvider struct {
	registry map[string]BYOKAdapter
	dispatch *tiered.BYOKDispatcher[BYOKAdapter, Result]
}

func NewBYOKProvider(registry map[string]BYOKAdapter) *BYOKProvider {
	return &BYOKProvider{
		registry: registry,
		dispatch: tiered.NewBYOKDispatcher(registry, "sttchain", ErrMissingBYOKProvider, ErrUnknownBYOKProvider, stampBYOKResult),
	}
}

// stampBYOKResult keeps the capability-specific Result shape local while the
// shared helper owns the tier/provider/model stamping sequence.
func stampBYOKResult(result *Result, adapter BYOKAdapter) {
	tiered.StampIdentity(&result.Tier, TierBYOK, &result.ProviderID, &result.ModelID, adapter.ID(), adapter.Model(), true)
}

func (p *BYOKProvider) Type() ProviderTier { return TierBYOK }

func (p *BYOKProvider) IsAvailable(context.Context) bool {
	return p != nil && p.dispatch.Configured()
}

func (p *BYOKProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	return p.dispatch.Execute(req.BYOKKey, req.BYOKProvider,
		func(adapter BYOKAdapter) (*Result, error) { return adapter.Transcribe(ctx, req.BYOKKey, req) })
}

func (p *BYOKProvider) Model() string { return tiered.DispatchedModel() }

// Traits reports the BYOK tier as streaming-capable when at least one
// registered adapter declares streaming support. The actual per-request
// capability is gated by the adapter selected by req.BYOKProvider; the
// selector resolves Provider-level traits at session start and the
// per-adapter dispatch happens inside TranscribeStreaming. Strategies
// is left empty so the selector applies the global compatibility
// matrix per the resolved adapter shape.
func (p *BYOKProvider) Traits() ProviderTraits {
	if p == nil {
		return ProviderTraits{}
	}
	stream := false
	for _, a := range p.registry {
		if a != nil && a.StreamingCapability() {
			stream = true
			break
		}
	}
	strategies := []StrategyKind{StrategyVADSegment, StrategyBuffered}
	if stream {
		strategies = append(strategies, StrategyPassthrough)
	}
	return ProviderTraits{Batch: true, Stream: stream, Strategies: strategies}
}

// TranscribeStreaming dispatches to the per-provider streaming adapter.
// Returns ErrUnknownBYOKProvider when the adapter is not registered;
// returns (nil, nil) when the resolved adapter declined streaming so the
// chain falls back to the next tier or unary mode.
func (p *BYOKProvider) TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	adapter, err := p.dispatch.Resolve(start.BYOKKey, start.BYOKProvider)
	if err != nil {
		return nil, err
	}
	if !adapter.StreamingCapability() {
		return nil, nil
	}
	return adapter.TranscribeStreaming(ctx, start.BYOKKey, start, chunks)
}
