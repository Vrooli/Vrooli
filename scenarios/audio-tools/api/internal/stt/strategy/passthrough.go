package strategy

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
)

// Passthrough is the strategy used for native-streaming providers
// (Deepgram, Azure, Google, future LPBS). It calls
// provider.TranscribeStreaming and forwards the returned event stream
// onto the Segmenter's events channel verbatim. The strategy itself
// performs no segmentation, transcoding, or buffering — those concerns
// live inside the provider's adapter.
type Passthrough struct {
	Provider sttchain.Provider
}

// Kind reports the strategy kind for selector enforcement.
func (p *Passthrough) Kind() sttchain.StrategyKind { return sttchain.StrategyPassthrough }

// Run delegates to provider.TranscribeStreaming. The provider owns the
// goroutine that produces events; this strategy only fans them onto
// the Segmenter channel and ensures a terminal Done is emitted even
// when the provider closes its channel without one (defensive — most
// adapters will always emit Done themselves).
func (p *Passthrough) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) error {
	if p.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: Passthrough requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	vendor, err := p.Provider.TranscribeStreaming(ctx, start, chunks)
	if err != nil {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	if vendor == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: passthrough provider returned nil event channel")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	sawDone := false
	for {
		var ev sttchain.StreamEvent
		var ok bool
		select {
		case ev, ok = <-vendor:
			if !ok {
				goto vendorDrained
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		if ev.Kind == sttchain.StreamEventDone {
			sawDone = true
		}
		select {
		case events <- ev:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

vendorDrained:
	if !sawDone {
		select {
		case events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
