// Package ingress is the pre-recognition audio-enhancement stage for streaming
// STT: the single seam inbound audio passes through AFTER it is normalized to
// canonical PCM and BEFORE any strategy (or the server-side VAD) consumes it.
//
// It is the symmetric counterpart of internal/stt/egress: where egress gates
// finished segments (text/confidence/speaker domains), ingress cleans the raw
// audio every strategy and the VAD then see (audio domain, pre-recognition).
// Like egress, the Segmenter constructs one Pipeline per session and wraps the
// canonical-PCM chunk channel with it; strategies never call it directly.
//
// All enhancers operate on canonical PCM (s16le / 16 kHz / mono) in and out, so
// the stage composes cleanly with the audioformat substrate and never changes
// the contract the VAD/Whisper depend on.
package ingress

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// Enhancer is one pre-recognition audio-enhancement stage operating on a live
// stream of canonical-PCM chunks.
//
// seam: Enhancer is the ingress audio-enhancement seam (SEAMS.md row
// "ingress.Enhancer"). Production wires the ffmpeg-backed DenoiseEnhancer built
// from the StreamConfig + engine capability; tests substitute fakes from
// internal/stt/ingress/mocks.
type Enhancer interface {
	Name() string
	// Process consumes canonical-PCM chunks from in and returns a new channel of
	// enhanced canonical-PCM chunks plus a cleanup func that releases the stage's
	// resources (a subprocess, goroutines). The returned channel MUST close when
	// in closes or ctx fires. A returned error is terminal for the session — the
	// caller emits an error + Done — so an enhancer that is merely unavailable
	// must be omitted from the Pipeline by the builder, not error here.
	Process(ctx context.Context, in <-chan sttchain.AudioChunk) (<-chan sttchain.AudioChunk, func(), error)
}

// Pipeline chains enhancers in order: the first enhancer's output feeds the
// next. An empty (or nil) Pipeline is the identity — it returns the input
// channel unchanged with a no-op cleanup, so wiring it always is free.
type Pipeline struct {
	enhancers []Enhancer
}

// NewPipeline builds a Pipeline from an ordered enhancer list.
func NewPipeline(enhancers ...Enhancer) *Pipeline {
	return &Pipeline{enhancers: enhancers}
}

// Names reports the ordered enhancer names — for telemetry and tests.
func (p *Pipeline) Names() []string {
	if p == nil {
		return nil
	}
	names := make([]string, 0, len(p.enhancers))
	for _, e := range p.enhancers {
		names = append(names, e.Name())
	}
	return names
}

// Process wires every enhancer in order. On the first enhancer error it rolls
// back the cleanups of the stages already started and returns the error (no
// channel). A nil/empty Pipeline returns in unchanged.
func (p *Pipeline) Process(ctx context.Context, in <-chan sttchain.AudioChunk) (<-chan sttchain.AudioChunk, func(), error) {
	if p == nil || len(p.enhancers) == 0 {
		return in, func() {}, nil
	}
	cur := in
	cleanups := make([]func(), 0, len(p.enhancers))
	rollback := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	for _, e := range p.enhancers {
		out, cleanup, err := e.Process(ctx, cur)
		if err != nil {
			rollback()
			return nil, func() {}, err
		}
		cleanups = append(cleanups, cleanup)
		cur = out
	}
	return cur, rollback, nil
}
