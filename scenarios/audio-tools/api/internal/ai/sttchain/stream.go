package sttchain

import (
	"context"
	"errors"
	"fmt"

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
	appendLocal := func() {
		if local := c.resolveLocalEngine(start.EngineID); local != nil && c.enableLocal && local.IsAvailable(ctx) {
			out = append(out, local)
		}
	}
	if c.localFirst {
		appendLocal()
	}
	if c.byok != nil && c.Eligible(ctx, tiered.SlotBYOK, req) {
		out = append(out, c.byok)
	}
	if c.vrooli != nil && c.Eligible(ctx, tiered.SlotVrooli, req) {
		out = append(out, c.vrooli)
	}
	// Local tier: resolve the engine-specific provider (Whisper vs Kyutai)
	// from StreamStart.EngineID, then gate on its own availability. The
	// embedded coordinator's SlotLocal eligibility only knows about Whisper,
	// so the resolved provider's IsAvailable is the authority on the
	// streaming path.
	if !c.localFirst {
		appendLocal()
	}
	return out
}

// LocalEngineAvailable reports whether the Local-tier engine with the given
// manifest id can serve right now (its backing resource is reachable). It backs
// the admin engine picker's per-engine availability without the handler having
// to know which resource an engine maps to — the chain owns the providers.
func (c *Chain) LocalEngineAvailable(ctx context.Context, engineID string) bool {
	if !c.enableLocal {
		return false
	}
	p := c.resolveLocalEngine(engineID)
	return p != nil && p.IsAvailable(ctx)
}

// resolveLocalEngine returns the Local-tier provider that serves the given
// engine id on the streaming path. Unknown/empty ids fall back to the default
// Local provider (Whisper). Returns nil only when no Local provider exists.
func (c *Chain) resolveLocalEngine(engineID string) Provider {
	if engineID != "" {
		if p, ok := c.localEngines[engineID]; ok && p != nil {
			return p
		}
	}
	if c.local == nil {
		return nil
	}
	return c.local
}

// Stream runs a streaming transcription session through the chain.
// Tier negotiation mirrors Execute()'s precedence, filtered by
// Traits().Stream=true. When no streaming-capable tier accepts, falls
// back to buffered mode (drain chunks, run Execute on concatenated
// audio, emit synthetic Segment+Done) unless RequireStreaming is set.
// RequireStreaming is the production fail-closed boundary: native provider
// failure must be visible immediately rather than becoming an unbounded
// whole-turn buffer.
//
// The returned channel is closed after the final Done. Caller must drain.
func (c *Chain) Stream(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	req := Request{BYOKKey: start.BYOKKey, LPBSToken: start.LPBSToken}
	if c.localFirst {
		if local := c.resolveLocalEngine(start.EngineID); local != nil && c.enableLocal && local.IsAvailable(ctx) && local.Traits().Stream {
			out, err := local.TranscribeStreaming(ctx, start, chunks)
			if err != nil && start.RequireStreaming {
				return nil, fmt.Errorf("%w: local: %v", ErrStreamingUnavailable, err)
			}
			if err == nil && out != nil {
				return out, nil
			}
			if start.RequireStreaming && err == nil {
				return nil, fmt.Errorf("%w: local returned no event channel", ErrStreamingUnavailable)
			}
		}
	}

	if c.byok != nil && c.Eligible(ctx, tiered.SlotBYOK, req) && c.byok.Traits().Stream {
		out, err := c.byok.TranscribeStreaming(ctx, start, chunks)
		if err != nil {
			if start.RequireStreaming {
				return nil, fmt.Errorf("%w: byok: %v", ErrStreamingUnavailable, err)
			}
			if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
				return nil, err
			}
		} else if out != nil {
			return out, nil
		} else if start.RequireStreaming {
			return nil, fmt.Errorf("%w: byok returned no event channel", ErrStreamingUnavailable)
		}
	}
	if c.vrooli != nil && c.Eligible(ctx, tiered.SlotVrooli, req) && c.vrooli.Traits().Stream {
		out, err := c.vrooli.TranscribeStreaming(ctx, start, chunks)
		if err != nil && start.RequireStreaming {
			return nil, fmt.Errorf("%w: vrooli: %v", ErrStreamingUnavailable, err)
		}
		if err == nil && out != nil {
			return out, nil
		}
		if start.RequireStreaming && err == nil {
			return nil, fmt.Errorf("%w: vrooli returned no event channel", ErrStreamingUnavailable)
		}
	}
	if !c.localFirst {
		if local := c.resolveLocalEngine(start.EngineID); local != nil && c.enableLocal && local.IsAvailable(ctx) && local.Traits().Stream {
			out, err := local.TranscribeStreaming(ctx, start, chunks)
			if err != nil && start.RequireStreaming {
				return nil, fmt.Errorf("%w: local: %v", ErrStreamingUnavailable, err)
			}
			if err == nil && out != nil {
				return out, nil
			}
			if start.RequireStreaming && err == nil {
				return nil, fmt.Errorf("%w: local returned no event channel", ErrStreamingUnavailable)
			}
		}
	}

	if start.RequireStreaming {
		return nil, ErrStreamingUnavailable
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
