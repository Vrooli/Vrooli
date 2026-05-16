package sttchain

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// fakeStreamProvider is a Provider whose streaming/availability/Transcribe
// behaviors are controlled by the test. It satisfies the full Provider
// interface so it can stand in for Local/BYOK/Vrooli in narrow tests.
type fakeStreamProvider struct {
	tier         ProviderTier
	available    bool
	streaming    bool
	transcribeFn func(context.Context, Request) (*Result, error)
	streamFn     func(context.Context, StreamStart, <-chan AudioChunk) (<-chan StreamEvent, error)
}

func (p *fakeStreamProvider) Type() ProviderTier                                   { return p.tier }
func (p *fakeStreamProvider) IsAvailable(context.Context) bool                     { return p.available }
func (p *fakeStreamProvider) Model() string                                        { return "fake" }
func (p *fakeStreamProvider) StreamingCapability() bool                            { return p.streaming }
func (p *fakeStreamProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	if p.transcribeFn != nil {
		return p.transcribeFn(ctx, req)
	}
	return &Result{Text: "buffered", Tier: p.tier, ProviderID: "fake", ModelID: "fake", Latency: 1 * time.Millisecond}, nil
}
func (p *fakeStreamProvider) TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error) {
	if p.streamFn != nil {
		return p.streamFn(ctx, start, chunks)
	}
	return nil, nil
}

// TestStream_BufferedFallback_NoStreamingCapableProvider verifies that
// the chain falls back to Execute()+synthetic events when nothing can
// stream.
func TestStream_BufferedFallback_NoStreamingCapableProvider(t *testing.T) {
	defer goleak.VerifyNone(t)

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

// TestStream_ClientCancelDrains verifies that cancelling the context
// closes the event channel without leaking goroutines.
func TestStream_ClientCancelDrains(t *testing.T) {
	defer goleak.VerifyNone(t)

	chain := NewChain(Options{EnableLocal: false, EnableBYOK: false, EnableVrooli: false})
	chunks := make(chan AudioChunk)
	ctx, cancel := context.WithCancel(context.Background())
	events, err := chain.Stream(ctx, StreamStart{}, chunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cancel()
	// Drain to allow the goroutine to exit cleanly. The fallback path
	// will see ctx.Done() and emit an error + close.
	for range events {
	}
	close(chunks)
}
