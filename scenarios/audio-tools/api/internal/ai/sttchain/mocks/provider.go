// Package mocks holds the hoisted test fakes for the STT chain. The
// FakeProvider here satisfies sttchain.Provider; selector / strategy
// tests share this one declaration instead of redeclaring per file.
package mocks

import (
	"context"
	"time"

	"audio-tools/internal/ai/sttchain"
)

// FakeProvider satisfies sttchain.Provider. Tests configure Tier and
// Traits via the constructor; per-call behavior can be overridden by
// setting TranscribeFn / StreamFn, otherwise Result / Err drive a
// canonical canned response. Calls counts Transcribe invocations.
type FakeProvider struct {
	Tier         sttchain.ProviderTier
	Traits_      sttchain.ProviderTraits
	Result       *sttchain.Result
	Err          error
	TranscribeFn func(ctx context.Context, req sttchain.Request) (*sttchain.Result, error)
	StreamFn     func(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error)
	Calls        int
}

// NewFakeProvider constructs a FakeProvider for the given tier + traits.
func NewFakeProvider(tier sttchain.ProviderTier, traits sttchain.ProviderTraits) *FakeProvider {
	return &FakeProvider{Tier: tier, Traits_: traits}
}

func (p *FakeProvider) Type() sttchain.ProviderTier      { return p.Tier }
func (p *FakeProvider) IsAvailable(context.Context) bool { return true }
func (p *FakeProvider) Model() string                    { return "fake" }
func (p *FakeProvider) Traits() sttchain.ProviderTraits  { return p.Traits_ }
func (p *FakeProvider) Transcribe(ctx context.Context, req sttchain.Request) (*sttchain.Result, error) {
	p.Calls++
	if p.TranscribeFn != nil {
		return p.TranscribeFn(ctx, req)
	}
	if p.Err != nil {
		return nil, p.Err
	}
	if p.Result != nil {
		return p.Result, nil
	}
	return &sttchain.Result{Text: "x", Tier: p.Tier, Latency: time.Millisecond}, nil
}

func (p *FakeProvider) TranscribeStreaming(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	if p.StreamFn != nil {
		return p.StreamFn(ctx, start, chunks)
	}
	return nil, nil
}

// FakeBatchExecutor satisfies the strategy.BatchExecutor seam used by
// the buffered-fallback strategy to dispatch into the unary chain.
// Tests set Result/Err for a single canned response, or ExecuteFn for
// per-call customization.
type FakeBatchExecutor struct {
	Result    *sttchain.Result
	Err       error
	ExecuteFn func(ctx context.Context, req sttchain.Request) (*sttchain.Result, error)
}

func (f *FakeBatchExecutor) Execute(ctx context.Context, req sttchain.Request) (*sttchain.Result, error) {
	if f.ExecuteFn != nil {
		return f.ExecuteFn(ctx, req)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Result != nil {
		return f.Result, nil
	}
	return &sttchain.Result{Text: "x", Latency: time.Millisecond}, nil
}

// Compile-time guarantees.
var _ sttchain.Provider = (*FakeProvider)(nil)
