package sttchain

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/goleak"
)

// ignoreSharedGlobals keeps goleak focused on goroutines this package creates.
// api-core/database opens a package-level foreignPinPool (sql.OpenDB) during
// init, whose connectionOpener is process-lifetime infrastructure. Ignoring
// the current snapshot is narrower than ignoring every DB opener, so a real
// leak created inside a test still fails.
func ignoreSharedGlobals() goleak.Option {
	return goleak.IgnoreCurrent()
}

// TestStream_BufferedFallback_NoStreamingCapableProvider verifies that
// the chain falls back to Execute()+synthetic events when nothing can
// stream.
func TestStream_BufferedFallback_NoStreamingCapableProvider(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreSharedGlobals())

	// Build a chain with a Local-only stub. We use the real chain shape
	// (concrete *LocalProvider etc) but with the local nil — exercising
	// the buffered fallback when none of the three concrete providers
	// is configured for streaming. Local provider returns "buffered" via
	// the unary path.
	chain := NewChain(Options{EnableLocal: false, EnableBYOK: false, EnableVrooli: false})
	chunks := make(chan AudioChunk, 1)
	chunks <- AudioChunk{Audio: []byte("ignored")}
	close(chunks)

	events, err := chain.Stream(context.Background(), StreamStart{}, chunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sawSegment, sawDone, sawError bool
	for ev := range events {
		switch ev.Kind {
		case StreamEventSegment:
			sawSegment = true
		case StreamEventDone:
			sawDone = true
			if !ev.Done.FellBackToUnary {
				t.Fatal("expected FellBackToUnary=true on fallback path")
			}
		case StreamEventError:
			sawError = true
		}
	}
	// With every tier disabled, Execute returns ErrAllProvidersFailed,
	// and the fallback path emits Error + Done.
	if !sawError {
		t.Fatalf("expected error event when all tiers disabled; saw segment=%v done=%v", sawSegment, sawDone)
	}
	if !sawDone {
		t.Fatal("expected done event")
	}
}

func TestStream_RequireStreamingFailsClosedWithoutProvider(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreSharedGlobals())

	chain := NewChain(Options{EnableLocal: false, EnableBYOK: false, EnableVrooli: false})
	chunks := make(chan AudioChunk)
	close(chunks)

	events, err := chain.Stream(context.Background(), StreamStart{RequireStreaming: true}, chunks)
	if events != nil {
		t.Fatal("strict streaming must not return a buffered event channel")
	}
	if !errors.Is(err, ErrStreamingUnavailable) {
		t.Fatalf("want ErrStreamingUnavailable, got %v", err)
	}
}

// TestStream_ClientCancelDrains verifies that cancelling the context
// closes the event channel without leaking goroutines.
func TestStream_ClientCancelDrains(t *testing.T) {
	defer goleak.VerifyNone(t, ignoreSharedGlobals())

	chain := NewChain(Options{EnableLocal: false, EnableBYOK: false, EnableVrooli: false})
	// A queued chunk can be ready at the same time as cancellation. Even if
	// the select consumes it first, the next iteration must observe cancel
	// and terminate with the documented Error + Done sequence.
	chunks := make(chan AudioChunk, 1)
	chunks <- AudioChunk{Audio: []byte("queued-before-cancel")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	events, err := chain.Stream(ctx, StreamStart{}, chunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var kinds []StreamEventKind
	for ev := range events {
		kinds = append(kinds, ev.Kind)
	}
	if len(kinds) != 2 || kinds[0] != StreamEventError || kinds[1] != StreamEventDone {
		t.Fatalf("canceled fallback must emit Error then terminal Done, got %v", kinds)
	}
}
