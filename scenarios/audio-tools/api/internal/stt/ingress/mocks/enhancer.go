// Package mocks provides fakes for the internal/stt/ingress seams.
package mocks

import (
	"context"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/ingress"
)

// FakeEnhancer is the canonical ingress.Enhancer fake. It optionally transforms
// each chunk's bytes (Transform) so tests can assert pipeline ordering, and can
// fail at Process start (StartErr) to exercise the Pipeline's rollback path.
type FakeEnhancer struct {
	NameVal   string
	Transform func([]byte) []byte
	StartErr  error

	// Started/Cleaned count Process starts and cleanup invocations so tests can
	// assert resource lifecycle (e.g. rollback ran).
	Started int
	Cleaned int
}

// Name implements ingress.Enhancer.
func (f *FakeEnhancer) Name() string {
	if f.NameVal == "" {
		return "fake"
	}
	return f.NameVal
}

// Process implements ingress.Enhancer.
func (f *FakeEnhancer) Process(ctx context.Context, in <-chan sttchain.AudioChunk) (<-chan sttchain.AudioChunk, func(), error) {
	if f.StartErr != nil {
		return nil, nil, f.StartErr
	}
	f.Started++
	out := make(chan sttchain.AudioChunk, 16)
	go func() {
		defer close(out)
		for {
			select {
			case c, ok := <-in:
				if !ok {
					return
				}
				audio := c.Audio
				if f.Transform != nil {
					audio = f.Transform(audio)
				}
				select {
				case out <- sttchain.AudioChunk{Audio: audio}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	cleanup := func() { f.Cleaned++ }
	return out, cleanup, nil
}

var _ ingress.Enhancer = (*FakeEnhancer)(nil)

// FakeTargetExtractor is a programmable ingress.TargetExtractor. Tests set
// Transform to compute cleaned PCM from the window (default: identity) or Err
// to force a per-window failure. Windows records every window it observed so a
// test can assert how the stream was chunked.
type FakeTargetExtractor struct {
	Transform func([]byte) []byte
	Err       error
	Windows   [][]byte
}

// Extract implements ingress.TargetExtractor.
func (f *FakeTargetExtractor) Extract(_ context.Context, pcm []byte) ([]byte, error) {
	win := make([]byte, len(pcm))
	copy(win, pcm)
	f.Windows = append(f.Windows, win)
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Transform != nil {
		return f.Transform(pcm), nil
	}
	return pcm, nil
}

var _ ingress.TargetExtractor = (*FakeTargetExtractor)(nil)
