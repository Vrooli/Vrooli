// Package strategy holds the concrete StreamingStrategy implementations
// consumed by the Segmenter via the StrategySelector. Each strategy is
// transport-free and provider-agnostic: it receives the same chunks/
// events channels regardless of how the bytes arrived (browser WS,
// Connect bidi, in-process test fixture) and regardless of which
// concrete provider was negotiated.
package strategy

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
)

// BatchExecutor is the minimum dependency the BufferedFallback strategy
// needs from the unary chain. Defining it as an interface here keeps the
// strategy package free of a back-edge import on the concrete *Chain
// (which would otherwise cycle: strategy depends on chain depends on
// strategy).
type BatchExecutor interface {
	Execute(ctx context.Context, req sttchain.Request) (*sttchain.Result, error)
}

// BufferedFallback is the strategy the selector returns when streaming
// is disabled (stt.streaming_mode=off) or when no eligible
// (strategy, provider) pair exists for the negotiated session. It
// drains the chunk channel, concatenates the audio, runs Execute on the
// supplied BatchExecutor, and emits one Segment + one Done event with
// FellBackToUnary=true.
//
// This is the same shape as the legacy chain.go::bufferedFallback path,
// extracted so the selector owns when it fires instead of having a
// silent side-channel inside Chain.Stream.
type BufferedFallback struct {
	Executor BatchExecutor
}

// Kind reports the strategy kind for selector enforcement.
func (b *BufferedFallback) Kind() sttchain.StrategyKind {
	return sttchain.StrategyBuffered
}

// Run drains chunks, executes the unary chain on the concatenated
// audio, and writes one Segment + Done to events. Returns the first
// error encountered (already mirrored on the event channel as a
// StreamEventError) so callers can shape outer-loop logging.
//
// Run never closes events — the Segmenter owns the channel lifetime
// and closes it after the strategy returns.
func (b *BufferedFallback) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) error {
	if b.Executor == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: BufferedFallback requires a BatchExecutor")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FellBackToUnary: true}}
		return err
	}

	var buf []byte
	for {
		select {
		case <-ctx.Done():
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: ctx.Err()}
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FellBackToUnary: true}}
			return ctx.Err()
		case ch, ok := <-chunks:
			if !ok {
				goto run
			}
			buf = append(buf, ch.Audio...)
		}
	}
run:
	req := sttchain.Request{
		Audio:                   buf,
		Language:                start.Language,
		InitialPrompt:           start.InitialPrompt,
		SkipSpeakerVerification: start.SkipSpeakerVerification,
		BYOKProvider:            start.BYOKProvider,
		BYOKKey:                 start.BYOKKey,
		LPBSToken:               start.LPBSToken,
		UserIdentity:            start.UserIdentity,
	}
	res, err := b.Executor.Execute(ctx, req)
	if err != nil {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FellBackToUnary: true}}
		return err
	}
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
		Text:             res.Text,
		DetectedLanguage: res.DetectedLanguage,
		ProviderTier:     res.Tier,
		ProviderID:       res.ProviderID,
		ModelID:          res.ModelID,
		LatencyMs:        float64(res.Latency.Milliseconds()),
	}}
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
		FinalText:       res.Text,
		LockedTier:      res.Tier,
		ProviderID:      res.ProviderID,
		ModelID:         res.ModelID,
		LatencyMs:       float64(res.Latency.Milliseconds()),
		FellBackToUnary: true,
	}}
	return nil
}
