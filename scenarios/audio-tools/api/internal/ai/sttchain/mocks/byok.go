package mocks

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// FakeBYOK satisfies sttchain.BYOKAdapter. Tests configure ID and
// Available; Result/Err drives the canned Transcribe response, or
// TranscribeFn for per-call behavior.
type FakeBYOK struct {
	IDStr        string
	Available    bool
	Result       *sttchain.Result
	Err          error
	TranscribeFn func(ctx context.Context, key string, req sttchain.Request) (*sttchain.Result, error)
	Streaming    bool
	StreamFn     func(ctx context.Context, key string, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error)
}

func (f *FakeBYOK) ID() string                               { return f.IDStr }
func (f *FakeBYOK) IsAvailable(context.Context, string) bool { return f.Available }
func (f *FakeBYOK) Model() string                            { return "fake-model" }
func (f *FakeBYOK) Transcribe(ctx context.Context, key string, req sttchain.Request) (*sttchain.Result, error) {
	if f.TranscribeFn != nil {
		return f.TranscribeFn(ctx, key, req)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Result != nil {
		return f.Result, nil
	}
	return &sttchain.Result{Text: "byok", Tier: sttchain.TierBYOK}, nil
}

// StreamingCapability defaults to false; tests that need streaming set
// Streaming=true and provide StreamFn.
func (f *FakeBYOK) StreamingCapability() bool { return f.Streaming }

// TranscribeStreaming delegates to StreamFn when set, otherwise returns
// (nil, nil) per the contract for non-streaming adapters.
func (f *FakeBYOK) TranscribeStreaming(ctx context.Context, key string, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	if f.StreamFn != nil {
		return f.StreamFn(ctx, key, start, chunks)
	}
	return nil, nil
}

var _ sttchain.BYOKAdapter = (*FakeBYOK)(nil)
