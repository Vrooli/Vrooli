package ttschain

import (
	"context"

	"audio-tools/internal/ai/chains/tiered"
)

// Stream runs a streaming synthesis session through the chain. Tier
// negotiation mirrors Execute()'s precedence, filtered by
// StreamingCapability()=true. When no streaming-capable tier accepts,
// falls back to Execute() and emits a single is_final=true frame.
//
// The returned channel is closed after the final frame.
func (c *Chain) Stream(ctx context.Context, req Request) (<-chan AudioFrame, error) {
	for _, slot := range tiered.Order(c.localFirst) {
		if c.Coordinator == nil {
			return c.bufferedFallback(ctx, req), nil
		}
		switch slot {
		case tiered.SlotLocal:
			if c.Local == nil || !c.Eligible(ctx, slot, req) || !c.Local.StreamingCapability() {
				continue
			}
			out, err := c.Local.SynthesizeStreaming(ctx, req)
			if err == nil && out != nil {
				return out, nil
			}
		case tiered.SlotBYOK:
			if c.BYOK == nil || !c.Eligible(ctx, slot, req) || !c.BYOK.StreamingCapability() {
				continue
			}
			out, err := c.BYOK.SynthesizeStreaming(ctx, req)
			if err != nil && tiered.StreamingTerminal(slot, err, ErrUnknownBYOKProvider, ErrMissingBYOKProvider, ErrInsufficientCredits) {
				return nil, err
			}
			if err == nil && out != nil {
				return out, nil
			}
		case tiered.SlotVrooli:
			if c.Vrooli == nil || !c.Eligible(ctx, slot, req) || !c.Vrooli.StreamingCapability() {
				continue
			}
			out, err := c.Vrooli.SynthesizeStreaming(ctx, req)
			// Credit exhaustion must not be masked by buffered fallback.
			if err != nil && tiered.StreamingTerminal(slot, err, ErrUnknownBYOKProvider, ErrMissingBYOKProvider, ErrInsufficientCredits) {
				return nil, err
			}
			if err == nil && out != nil {
				return out, nil
			}
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
