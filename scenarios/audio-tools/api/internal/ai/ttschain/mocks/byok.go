// Package mocks holds the hoisted test fakes for the TTS chain.
package mocks

import (
	"context"

	"audio-tools/internal/ai/ttschain"
)

// FakeBYOK satisfies ttschain.BYOKAdapter. Tests configure ID and
// Available; Result drives the canned Synthesize response, or
// SynthesizeFn for per-call control. Streaming is off by default.
type FakeBYOK struct {
	IDStr        string
	Available    bool
	Result       *ttschain.Result
	Err          error
	SynthesizeFn func(ctx context.Context, key string, req ttschain.Request) (*ttschain.Result, error)
	Streaming    bool
}

func (f *FakeBYOK) ID() string                               { return f.IDStr }
func (f *FakeBYOK) IsAvailable(context.Context, string) bool { return f.Available }
func (f *FakeBYOK) Model() string                            { return "fake-model" }
func (f *FakeBYOK) Synthesize(ctx context.Context, key string, req ttschain.Request) (*ttschain.Result, error) {
	if f.SynthesizeFn != nil {
		return f.SynthesizeFn(ctx, key, req)
	}
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Result != nil {
		return f.Result, nil
	}
	return &ttschain.Result{Audio: []byte("byok-audio"), ContentType: "audio/mpeg"}, nil
}
func (f *FakeBYOK) StreamingCapability() bool { return f.Streaming }
func (f *FakeBYOK) SynthesizeStreaming(context.Context, string, ttschain.Request) (<-chan ttschain.AudioFrame, error) {
	return nil, nil
}

var _ ttschain.BYOKAdapter = (*FakeBYOK)(nil)
