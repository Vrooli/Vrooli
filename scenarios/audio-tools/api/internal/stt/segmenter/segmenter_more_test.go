package segmenter

import (
	"context"
	"errors"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt"
)

// drain pulls events into a slice until the channel closes.
func drain(events <-chan sttchain.StreamEvent) []sttchain.StreamEvent {
	var out []sttchain.StreamEvent
	for ev := range events {
		out = append(out, ev)
	}
	return out
}

// TestSegmenter_NilGuard asserts the nil-Deps guard fires and emits a
// terminal Error+Done pair before returning. This is the early-return
// path that protects transports from a misconfigured factory.
func TestSegmenter_NilGuard(t *testing.T) {
	seg := New(Deps{})
	chunks := make(chan sttchain.AudioChunk)
	close(chunks)
	events := make(chan sttchain.StreamEvent, 4)
	err := seg.Run(context.Background(), sttchain.StreamStart{}, stt.StreamConfig{}, chunks, events)
	if err == nil {
		t.Fatal("expected error from nil-Deps guard")
	}
	out := drain(events)
	if len(out) != 2 {
		t.Fatalf("want 2 terminal events, got %d (%+v)", len(out), out)
	}
	if out[0].Kind != sttchain.StreamEventError || out[1].Kind != sttchain.StreamEventDone {
		t.Fatalf("want Error then Done; got %v / %v", out[0].Kind, out[1].Kind)
	}
}

// TestSegmenter_ContextCancel_PropagatesToStrategy confirms that when
// the supplied ctx is cancelled before chunks close, the BufferedFallback
// strategy observes it and emits Error(ctx.Err)+Done.
func TestSegmenter_ContextCancel_PropagatesToStrategy(t *testing.T) {
	exec := &sttmocks.FakeBatchExecutor{Result: &sttchain.Result{Text: "ignored"}}
	chain := sttchain.NewChain(sttchain.Options{})
	seg := New(Deps{Chain: chain, Selector: stt.NewSelector(exec)})

	ctx, cancel := context.WithCancel(context.Background())
	chunks := make(chan sttchain.AudioChunk) // never closed → strategy blocks
	events := make(chan sttchain.StreamEvent, 4)

	runDone := make(chan error, 1)
	go func() {
		runDone <- seg.Run(ctx, sttchain.StreamStart{}, stt.StreamConfig{Mode: stt.ModeOff}, chunks, events)
	}()

	cancel()

	select {
	case err := <-runDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want ctx.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	out := drain(events)
	if len(out) < 2 {
		t.Fatalf("expected at least Error+Done, got %d", len(out))
	}
	if out[len(out)-1].Kind != sttchain.StreamEventDone {
		t.Fatalf("last event should be Done, got %v", out[len(out)-1].Kind)
	}
}

// TestSegmenter_ExecutorError surfaces as a StreamEventError followed by
// a terminal Done(FellBackToUnary=true) — proving the strategy's error
// path is plumbed through the segmenter's caller-visible channel.
func TestSegmenter_ExecutorError(t *testing.T) {
	exec := &sttmocks.FakeBatchExecutor{Err: errors.New("provider blew up")}
	chain := sttchain.NewChain(sttchain.Options{})
	seg := New(Deps{Chain: chain, Selector: stt.NewSelector(exec)})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x00}}
	close(chunks)

	events := make(chan sttchain.StreamEvent, 4)
	_ = seg.Run(ctx, sttchain.StreamStart{}, stt.StreamConfig{Mode: stt.ModeOff}, chunks, events)
	out := drain(events)
	if len(out) != 2 {
		t.Fatalf("want Error+Done, got %d (%+v)", len(out), out)
	}
	if out[0].Kind != sttchain.StreamEventError {
		t.Fatalf("first event should be Error, got %v", out[0].Kind)
	}
	if out[1].Kind != sttchain.StreamEventDone {
		t.Fatalf("second event should be Done, got %v", out[1].Kind)
	}
	if out[1].Done == nil || !out[1].Done.FellBackToUnary {
		t.Fatalf("Done.FellBackToUnary should be true on buffered-fallback error path: %+v", out[1].Done)
	}
}

// TestSegmenter_ModeOffShortCircuit confirms that StreamConfig.Mode=off
// drives the selector to BufferedFallback even when the chain has no
// providers, and the executor's canned Result reaches the consumer.
func TestSegmenter_ModeOffShortCircuit(t *testing.T) {
	exec := &sttmocks.FakeBatchExecutor{Result: &sttchain.Result{Text: "tldr"}}
	chain := sttchain.NewChain(sttchain.Options{})
	seg := New(Deps{Chain: chain, Selector: stt.NewSelector(exec)})

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01}}
	close(chunks)
	events := make(chan sttchain.StreamEvent, 4)

	_ = seg.Run(context.Background(), sttchain.StreamStart{}, stt.StreamConfig{Mode: stt.ModeOff}, chunks, events)
	out := drain(events)
	if len(out) == 0 {
		t.Fatal("expected events from mode=off short-circuit")
	}
	last := out[len(out)-1]
	if last.Kind != sttchain.StreamEventDone || last.Done == nil || last.Done.FinalText != "tldr" {
		t.Fatalf("want Done.FinalText=tldr, got %+v", last)
	}
}

func TestSegmenter_AutoFailsClosedWithoutStreamingProvider(t *testing.T) {
	run := false
	exec := &sttmocks.FakeBatchExecutor{ExecuteFn: func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		run = true
		return &sttchain.Result{Text: "must not run"}, nil
	}}
	chain := sttchain.NewChain(sttchain.Options{})
	seg := New(Deps{Chain: chain, Selector: stt.NewSelector(exec)})

	chunks := make(chan sttchain.AudioChunk)
	close(chunks)
	events := make(chan sttchain.StreamEvent, 4)
	err := seg.Run(context.Background(), sttchain.StreamStart{}, stt.StreamConfig{Mode: stt.ModeAuto}, chunks, events)
	if !errors.Is(err, stt.ErrNoEligibleProvider) {
		t.Fatalf("want ErrNoEligibleProvider, got %v", err)
	}
	out := drain(events)
	if len(out) != 2 || out[0].Kind != sttchain.StreamEventError || out[1].Kind != sttchain.StreamEventDone {
		t.Fatalf("want Error+Done, got %+v", out)
	}
	if out[1].Done == nil || out[1].Done.FellBackToUnary {
		t.Fatalf("strict auto failure must not claim unary fallback: %+v", out[1].Done)
	}
	if run {
		t.Fatal("strict auto failure must not invoke unary executor")
	}
}
