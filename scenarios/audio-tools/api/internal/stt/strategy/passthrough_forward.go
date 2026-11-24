package strategy

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// forwardProviderEvents relays the provider stream and owns terminal Done
// handling. On cancellation, it tears down the tracked input feeder before
// emitting Done so provider events cannot arrive afterward.
func forwardProviderEvents(
	ctx context.Context,
	vendor <-chan sttchain.StreamEvent,
	events chan<- sttchain.StreamEvent,
	cancel context.CancelFunc,
	trackedDone <-chan struct{},
) error {
	sawDone := false
	forwardedDone := false
	terminateOnCancel := func() error {
		cancel()
		<-trackedDone
		if !forwardedDone {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		}
		return ctx.Err()
	}
	for {
		var ev sttchain.StreamEvent
		var ok bool
		select {
		case ev, ok = <-vendor:
			if !ok {
				goto vendorDrained
			}
		case <-ctx.Done():
			return terminateOnCancel()
		}
		if ev.Kind == sttchain.StreamEventDone {
			sawDone = true
		}
		select {
		case events <- ev:
			if ev.Kind == sttchain.StreamEventDone {
				forwardedDone = true
			}
		case <-ctx.Done():
			return terminateOnCancel()
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
