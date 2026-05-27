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
