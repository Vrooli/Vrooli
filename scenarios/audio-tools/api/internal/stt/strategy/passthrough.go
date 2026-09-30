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
	cursor *sttchain.ConsumptionCursor,
) error {
	if p.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: Passthrough requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	backendAcknowledgements := p.Provider.Traits().BackendAcknowledgements
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	tracked := make(chan sttchain.AudioChunk)
	trackedDone := make(chan struct{})
	go func() {
		defer close(trackedDone)
		defer close(tracked)
		for {
			select {
			case <-runCtx.Done():
				return
			case chunk, ok := <-chunks:
				if !ok {
					return
				}
				select {
				case tracked <- chunk:
					// A backend-confirmed provider (Kyutai) emits coverage only
					// after its resource reports processed audio. Other native
					// providers have no such signal, so acknowledge after their
					// adapter has accepted the chunk into its input stream.
					if !backendAcknowledgements {
						cursor.Observe(chunk)
					}
				case <-runCtx.Done():
					return
				}
			}
		}
	}()
	vendor, err := p.Provider.TranscribeStreaming(runCtx, start, tracked)
	if err != nil {
		cancel()
		<-trackedDone
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	if vendor == nil {
		cancel()
		<-trackedDone
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
			cancel()
			<-trackedDone
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
	cancel()
	<-trackedDone
	if !sawDone {
		select {
		case events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
