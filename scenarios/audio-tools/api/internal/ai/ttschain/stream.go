package ttschain

import (
	"context"
	"errors"

	"audio-tools/internal/ai/chains/tiered"
)

// Stream runs a streaming synthesis session through the chain. Tier
// negotiation mirrors Execute()'s precedence, filtered by
// StreamingCapability()=true. When no streaming-capable tier accepts,
// falls back to Execute() and emits a single is_final=true frame.
//
// The returned channel is closed after the final frame.
func (c *Chain) Stream(ctx context.Context, req Request) (<-chan AudioFrame, error) {
	if c.byok != nil && c.coord.Eligible(ctx, tiered.SlotBYOK, req) && c.byok.StreamingCapability() {
		out, err := c.byok.SynthesizeStreaming(ctx, req)
		if err != nil {
			if errors.Is(err, ErrUnknownBYOKProvider) || errors.Is(err, ErrMissingBYOKProvider) {
				return nil, err
			}
		} else if out != nil {
			return out, nil
		}
	}
	if c.vrooli != nil && c.coord.Eligible(ctx, tiered.SlotVrooli, req) && c.vrooli.StreamingCapability() {
		out, err := c.vrooli.SynthesizeStreaming(ctx, req)
		if err == nil && out != nil {
			return out, nil
		}
	}
	if c.local != nil && c.coord.Eligible(ctx, tiered.SlotLocal, req) && c.local.StreamingCapability() {
		out, err := c.local.SynthesizeStreaming(ctx, req)
		if err == nil && out != nil {
			return out, nil
		}
	}
	return c.bufferedFallback(ctx, req), nil
}

// bufferedFallback runs Execute() once and emits a single is_final=true
// frame carrying the full audio.
func (c *Chain) bufferedFallback(ctx context.Context, req Request) <-chan AudioFrame {
	out := make(chan AudioFrame, 1)
	go func() {
		defer close(out)
		res, err := c.Execute(ctx, req)
		if err != nil {
			out <- AudioFrame{IsFinal: true, Err: err}
			return
		}
		out <- AudioFrame{
			Audio:       res.Audio,
			ContentType: res.ContentType,
			IsFinal:     true,
			Tier:        res.Tier,
			ProviderID:  res.ProviderID,
			ModelID:     res.ModelID,
			VoiceUsed:   res.VoiceUsed,
			Latency:     res.Latency,
			ContentHash: res.ContentHash,
		}
	}()
	return out
}
