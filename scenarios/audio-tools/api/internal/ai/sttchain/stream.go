package sttchain

import (
	"context"
	"errors"

	"audio-tools/internal/ai/chains/tiered"
)

// StreamCandidates returns the precedence-ordered list of providers
// eligible for a streaming session given the per-request StreamStart
// and the chain's enable flags + cached availability. Does NOT filter
// by Traits().Stream; the StrategySelector applies that filter once it
// knows which strategy is being negotiated.
//
// Returned slice is freshly allocated each call.
func (c *Chain) StreamCandidates(ctx context.Context, start StreamStart) []Provider {
	out := make([]Provider, 0, 3)
	req := Request{BYOKKey: start.BYOKKey, LPBSToken: start.LPBSToken}
	if c.byok != nil && c.coord.Eligible(ctx, tiered.SlotBYOK, req) {
		out = append(out, c.byok)
	}
	if c.vrooli != nil && c.coord.Eligible(ctx, tiered.SlotVrooli, req) {
		out = append(out, c.vrooli)
	}
	if c.local != nil && c.coord.Eligible(ctx, tiered.SlotLocal, req) {
		out = append(out, c.local)
	}
	return out
}

// Stream runs a streaming transcription session through the chain.
// Tier negotiation mirrors Execute()'s precedence, filtered by
// Traits().Stream=true. When no streaming-capable tier accepts, falls
// back to buffered mode (drain chunks, run Execute on concatenated
// audio, emit synthetic Segment+Done).
//
// The returned channel is closed after the final Done. Caller must drain.
func (c *Chain) Stream(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	req := Request{BYOKKey: start.BYOKKey, LPBSToken: start.LPBSToken}

	if c.byok != nil && c.coord.Eligible(ctx, tiered.SlotBYOK, req) && c.byok.Traits().Stream {
		out, err := c.byok.TranscribeStreaming(ctx, start, chunks)
		if err != nil {
			if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
				return nil, err
			}
		} else if out != nil {
			return out, nil
		}
	}
	if c.vrooli != nil && c.coord.Eligible(ctx, tiered.SlotVrooli, req) && c.vrooli.Traits().Stream {
		out, err := c.vrooli.TranscribeStreaming(ctx, start, chunks)
		if err == nil && out != nil {
			return out, nil
		}
	}
	if c.local != nil && c.coord.Eligible(ctx, tiered.SlotLocal, req) && c.local.Traits().Stream {
		out, err := c.local.TranscribeStreaming(ctx, start, chunks)
		if err == nil && out != nil {
			return out, nil
		}
	}

	return c.bufferedFallback(ctx, start, chunks), nil
}

// bufferedFallback consumes the chunks channel, runs the unary chain on
// the concatenated audio, and emits a single Segment + Done event
// stamped FellBackToUnary=true.
func (c *Chain) bufferedFallback(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) <-chan StreamEvent {
	out := make(chan StreamEvent, 4)
	go func() {
		defer close(out)
		var buf []byte
		for {
			select {
			case <-ctx.Done():
				out <- StreamEvent{Kind: StreamEventError, Error: ctx.Err()}
				return
			case ch, ok := <-chunks:
				if !ok {
					goto run
				}
				buf = append(buf, ch.Audio...)
			}
		}
	run:
		req := Request{
			Audio:                   buf,
			Language:                start.Language,
			InitialPrompt:           start.InitialPrompt,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			BYOKProvider:            start.BYOKProvider,
			BYOKKey:                 start.BYOKKey,
			LPBSToken:               start.LPBSToken,
			UserIdentity:            start.UserIdentity,
		}
		res, err := c.Execute(ctx, req)
		if err != nil {
			out <- StreamEvent{Kind: StreamEventError, Error: err}
			out <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{FellBackToUnary: true}}
			return
		}
		out <- StreamEvent{Kind: StreamEventSegment, Segment: &SegmentEvent{
			Text:             res.Text,
			DetectedLanguage: res.DetectedLanguage,
			ProviderTier:     res.Tier,
			ProviderID:       res.ProviderID,
			ModelID:          res.ModelID,
			LatencyMs:        float64(res.Latency.Milliseconds()),
		}}
		out <- StreamEvent{Kind: StreamEventDone, Done: &DoneEvent{
			FinalText:       res.Text,
			LockedTier:      res.Tier,
			ProviderID:      res.ProviderID,
			ModelID:         res.ModelID,
			LatencyMs:       float64(res.Latency.Milliseconds()),
			FellBackToUnary: true,
		}}
	}()
	return out
}
