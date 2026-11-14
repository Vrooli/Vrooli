// Package mocks holds the hoisted test fakes for the STT chain. The
// FakeProvider here satisfies sttchain.Provider; selector / strategy
// tests share this one declaration instead of redeclaring per file.
package mocks

import (
	"context"
	"sync"
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

	// mu guards Calls and the configurable behaviour fields against the
	// concurrent Transcribe calls the Provider contract allows. Read Calls
	// through CallCount from a test that runs a strategy with previews.
	mu sync.Mutex
}

// NewFakeProvider constructs a FakeProvider for the given tier + traits.
func NewFakeProvider(tier sttchain.ProviderTier, traits sttchain.ProviderTraits) *FakeProvider {
	return &FakeProvider{Tier: tier, Traits_: traits}
}

// FakeProviderBuilder gives provider tests an overridable default without
// repeating an inline construction literal at every call site. Build returns
// a fresh fake, so one test cannot accidentally share call counts with another.
type FakeProviderBuilder struct {
	provider FakeProvider
}

func NewFakeProviderBuilder(tier sttchain.ProviderTier) *FakeProviderBuilder {
	return &FakeProviderBuilder{provider: FakeProvider{Tier: tier}}
}

func (b *FakeProviderBuilder) WithTraits(traits sttchain.ProviderTraits) *FakeProviderBuilder {
	b.provider.Traits_ = traits
	return b
}

func (b *FakeProviderBuilder) WithResult(result *sttchain.Result) *FakeProviderBuilder {
	b.provider.Result = result
	return b
}

func (b *FakeProviderBuilder) WithError(err error) *FakeProviderBuilder {
	b.provider.Err = err
	return b
}

func (b *FakeProviderBuilder) Build() *FakeProvider {
	provider := b.provider
	return &provider
}

func (p *FakeProvider) Type() sttchain.ProviderTier      { return p.Tier }
func (p *FakeProvider) IsAvailable(context.Context) bool { return true }
func (p *FakeProvider) Model() string                    { return "fake" }
func (p *FakeProvider) Traits() sttchain.ProviderTraits  { return p.Traits_ }
func (p *FakeProvider) Transcribe(ctx context.Context, req sttchain.Request) (*sttchain.Result, error) {
	// Providers must tolerate concurrent Transcribe calls (see the interface
	// doc): VADSegment issues preview transcriptions alongside the boundary
	// transcription on the same instance. The fake counts calls, so it needs
	// the same guarantee or `-race` reports the mock rather than the code.
	p.mu.Lock()
	p.Calls++
	fn, err, result := p.TranscribeFn, p.Err, p.Result
	p.mu.Unlock()

	if fn != nil {
		return fn(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	if result != nil {
		return result, nil
	}
	return &sttchain.Result{Text: "x", Tier: p.Tier, Latency: time.Millisecond}, nil
}

// CallCount reports Transcribe invocations without racing a concurrent call.
func (p *FakeProvider) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Calls
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
