package strategy_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

// strategyEventShapes projects events onto the fields both repeated runs must
// agree on. Latency, ids, and timing are intentionally excluded: the shape is
// what a consumer can observe about the strategy's decisions.
func strategyEventShapes(events []sttchain.StreamEvent) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			text := ""
			if ev.Segment != nil {
				text = ev.Segment.Text
			}
			out = append(out, "segment:"+text)
		case sttchain.StreamEventPartial:
			text := ""
			if ev.Partial != nil {
				text = ev.Partial.Text
			}
			out = append(out, "partial:"+text)
		case sttchain.StreamEventDone:
			text := ""
			if ev.Done != nil {
				text = ev.Done.FinalText
			}
			out = append(out, "done:"+text)
		default:
			out = append(out, string(ev.Kind))
		}
	}
	return out
}

func fixedTextProvider(text string) *sttmocks.FakeProvider {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: text, Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	return prov
}

// TestOverlapAgree_ZeroValueInstanceIsReusable is the regression for Run
// mutating its receiver through applyDefaults. The strategy is documented as
// stateless across sessions, so a zero-value instance must not acquire
// defaults behind the caller's back, and a second run must make the same
// decisions as the first.
func TestOverlapAgree_ZeroValueInstanceIsReusable(t *testing.T) {
	strat := &strategy.OverlapAgree{Provider: fixedTextProvider("hello world")}
	audio := testaudio.SineSamples(440, 4000)

	first := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))
	require.Zero(t, strat.SampleRate, "Run must not write resolved defaults back to the receiver")
	require.Zero(t, strat.WindowMs)
	require.Zero(t, strat.AdvanceMs)

	second := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	require.Equal(t, strategyEventShapes(first), strategyEventShapes(second),
		"the same instance must produce identical event shapes on a second session")
}

// TestVADSegmenter_ZeroValueInstanceIsReusable is the VADSegmenter counterpart
// of the reuse regression above.
func TestVADSegmenter_ZeroValueInstanceIsReusable(t *testing.T) {
	strat := &strategy.VADSegmenter{Provider: fixedTextProvider("hello world")}
	audio := testaudio.SineSamples(440, 4000)

	first := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))
	require.Zero(t, strat.SampleRate, "Run must not write resolved defaults back to the receiver")
	require.Zero(t, strat.SilenceMs)
	require.Zero(t, strat.PreviewWindowMs)

	second := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	require.Equal(t, strategyEventShapes(first), strategyEventShapes(second),
		"the same instance must produce identical event shapes on a second session")
}

// TestOverlapAgree_ConcurrentRunsDoNotShareMutableConfig proves the same
// instance can actually be used by two live sessions. Before defaults moved to
// per-run state, both Runs wrote the receiver's config fields, which -race
// reports as a data race even though the resolved values happened to match.
func TestOverlapAgree_ConcurrentRunsDoNotShareMutableConfig(t *testing.T) {
	strat := &strategy.OverlapAgree{Provider: fixedTextProvider("hello world")}
	audio := testaudio.SineSamples(440, 4000)

	results := make([][]string, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			events := make(chan sttchain.StreamEvent, 32)
			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = strat.Run(context.Background(), sttchain.StreamStart{}, chunksFrom(audio), events, nil)
				close(events)
			}()
			for ev := range events {
				results[index] = append(results[index], strategyEventShapes([]sttchain.StreamEvent{ev})...)
			}
			<-done
		}(i)
	}
	wg.Wait()

	require.Equal(t, results[0], results[1], "concurrent sessions must produce identical event shapes")
}

// TestOverlapAgree_CancelFlushesTail is the regression for cancellation
// dropping the final tail. The channel-close path transcribes audio that no
// agreement/boundary committed; the ctx.Done path used to emit a clean Done
// without it, so a request cancel mid-utterance reported success while the
// last words were never committed.
func TestOverlapAgree_CancelFlushesTail(t *testing.T) {
	firstCall := make(chan struct{})
	var calls atomic.Int32
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(ctx context.Context, _ sttchain.Request) (*sttchain.Result, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if calls.Add(1) == 1 {
			close(firstCall)
			return &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal}, nil
		}
		return &sttchain.Result{Text: "hello world", Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider:    prov,
		Trigger:     strategy.TriggerStopwatch,
		WindowMs:    1000,
		AdvanceMs:   500,
		CommitRuns:  2,
		MaxWindowMs: 60000,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan sttchain.StreamEvent, 32)
	chunks := make(chan sttchain.AudioChunk)
	done := make(chan error, 1)
	go func() {
		defer close(events)
		done <- strat.Run(ctx, sttchain.StreamStart{}, chunks, events, nil)
	}()

	// One second of audio runs the first settle attempt but commits nothing
	// (CommitRuns=2). The tail is therefore present when cancellation arrives.
	chunks <- sttchain.AudioChunk{Audio: testaudio.SineSamples(440, 1000)}
	select {
	case <-firstCall:
	case <-time.After(2 * time.Second):
		t.Fatal("first settle attempt did not run")
	}
	cancel()

	got := drainEvents(t, events)
	require.ErrorIs(t, <-done, context.Canceled)
	require.GreaterOrEqual(t, calls.Load(), int32(2), "cancellation must attempt the tail transcription")

	var segments []string
	var final string
	for _, ev := range got {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			if ev.Segment != nil {
				segments = append(segments, ev.Segment.Text)
			}
		case sttchain.StreamEventDone:
			if ev.Done != nil {
				final = ev.Done.FinalText
			}
		}
	}
	require.Equal(t, []string{"hello world"}, segments, "the uncommitted tail must be emitted before Done")
	require.Equal(t, "hello world", final)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// TestOverlapAgree_CancelTailFlushRespectsDeadline proves the cancellation
// flush is bounded by the caller's own deadline. A provider that only returns
// when its context is done must not hold teardown past that deadline, and the
// attempt must actually have been made.
func TestOverlapAgree_CancelTailFlushRespectsDeadline(t *testing.T) {
	firstCall := make(chan struct{})
	var calls atomic.Int32
	var flushed atomic.Bool
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(ctx context.Context, _ sttchain.Request) (*sttchain.Result, error) {
		if calls.Add(1) == 1 {
			close(firstCall)
			return &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal}, nil
		}
		// The tail attempt waits for its bounded context instead of returning
		// immediately, so the test measures the flush budget.
		<-ctx.Done()
		flushed.Store(true)
		return nil, ctx.Err()
	}
	strat := &strategy.OverlapAgree{
		Provider:    prov,
		Trigger:     strategy.TriggerStopwatch,
		WindowMs:    1000,
		AdvanceMs:   500,
		CommitRuns:  2,
		MaxWindowMs: 60000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()
	events := make(chan sttchain.StreamEvent, 32)
	chunks := make(chan sttchain.AudioChunk)
	done := make(chan error, 1)
	go func() {
		defer close(events)
		done <- strat.Run(ctx, sttchain.StreamStart{}, chunks, events, nil)
	}()

	chunks <- sttchain.AudioChunk{Audio: testaudio.SineSamples(440, 1000)}
	select {
	case <-firstCall:
	case <-time.After(2 * time.Second):
		t.Fatal("first settle attempt did not run")
	}
	start := time.Now()
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("cancellation blocked past the bounded flush budget")
	}
	drainEvents(t, events)

	require.True(t, flushed.Load(), "the bounded tail attempt must still reach the provider")
	require.Less(t, time.Since(start), 2500*time.Millisecond,
		"teardown must respect the caller's deadline instead of hanging")
}

// TestVADSegmenter_CancelFlushesTail is the VAD counterpart. Its cancellation
// path already called flushSegment, but with the already-cancelled session
// context, so a ctx-honouring provider failed the request and the tail was
// still dropped. The flush must use a bounded, detached context.
func TestVADSegmenter_CancelFlushesTail(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(ctx context.Context, _ sttchain.Request) (*sttchain.Result, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return &sttchain.Result{Text: "tail", Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.VADSegmenter{
		Provider:          prov,
		SilenceMs:         60000,
		MaxSegmentMs:      60000,
		PreviewIntervalMs: 0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan sttchain.StreamEvent, 32)
	chunks := make(chan sttchain.AudioChunk)
	done := make(chan error, 1)
	go func() {
		defer close(events)
		done <- strat.Run(ctx, sttchain.StreamStart{}, chunks, events, nil)
	}()

	// Voice the buffer without crossing the (deliberately huge) silence or
	// max-segment bounds, so only the cancellation flush can emit it.
	chunks <- sttchain.AudioChunk{Audio: testaudio.VoicedSamples(500, 1000)}
	cancel()

	got := drainEvents(t, events)
	require.ErrorIs(t, <-done, context.Canceled)

	var segments []string
	var final string
	for _, ev := range got {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			if ev.Segment != nil {
				segments = append(segments, ev.Segment.Text)
			}
		case sttchain.StreamEventDone:
			if ev.Done != nil {
				final = ev.Done.FinalText
			}
		}
	}
	require.Equal(t, []string{"tail"}, segments, "cancellation must flush voiced audio with a usable context")
	require.Equal(t, "tail", final)
}
