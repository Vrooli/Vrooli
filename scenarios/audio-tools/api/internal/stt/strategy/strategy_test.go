package strategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/segmenter/testaudio"
)

// fakeProvider satisfies sttchain.Provider with test-controllable
// Transcribe and TranscribeStreaming behaviour.
type fakeProvider struct {
	tier        sttchain.ProviderTier
	calls       int
	transcribe  func(ctx context.Context, req sttchain.Request) (*sttchain.Result, error)
	stream      func(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error)
	traits      sttchain.ProviderTraits
}

func (f *fakeProvider) Type() sttchain.ProviderTier             { return f.tier }
func (f *fakeProvider) IsAvailable(context.Context) bool        { return true }
func (f *fakeProvider) Model() string                           { return "fake" }
func (f *fakeProvider) Traits() sttchain.ProviderTraits         { return f.traits }
func (f *fakeProvider) Transcribe(ctx context.Context, req sttchain.Request) (*sttchain.Result, error) {
	f.calls++
	if f.transcribe != nil {
		return f.transcribe(ctx, req)
	}
	return &sttchain.Result{Text: "ok", Tier: f.tier, ProviderID: "fake", ModelID: "fake-1", Latency: time.Millisecond}, nil
}
func (f *fakeProvider) TranscribeStreaming(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	if f.stream != nil {
		return f.stream(ctx, start, chunks)
	}
	return nil, nil
}

// drainEvents reads every event from `out` until it closes; returns the
// flat list. Caller is responsible for ensuring the strategy goroutine
// closes the channel.
func drainEvents(t *testing.T, out <-chan sttchain.StreamEvent) []sttchain.StreamEvent {
	t.Helper()
	var got []sttchain.StreamEvent
	for ev := range out {
		got = append(got, ev)
	}
	return got
}

// runStrategy spawns strat.Run in a goroutine, closes events when Run
// returns, and returns all events.
func runStrategy(t *testing.T, ctx context.Context, strat Strategy, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) []sttchain.StreamEvent {
	t.Helper()
	events := make(chan sttchain.StreamEvent, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = strat.Run(ctx, start, chunks, events)
		close(events)
	}()
	out := drainEvents(t, events)
	<-done
	return out
}

func chunksFrom(audio []byte) <-chan sttchain.AudioChunk {
	ch := make(chan sttchain.AudioChunk, 1)
	ch <- sttchain.AudioChunk{Audio: audio}
	close(ch)
	return ch
}

// ----- BufferedFallback -----

type fakeExecutor struct {
	res *sttchain.Result
	err error
}

func (f *fakeExecutor) Execute(context.Context, sttchain.Request) (*sttchain.Result, error) {
	return f.res, f.err
}

func TestBufferedFallback_HappyPathEmitsSegmentThenDone(t *testing.T) {
	strat := &BufferedFallback{Executor: &fakeExecutor{
		res: &sttchain.Result{Text: "hello world", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: 5 * time.Millisecond},
	}}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom([]byte("audio")))

	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventSegment, got[0].Kind)
	require.Equal(t, "hello world", got[0].Segment.Text)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
	require.True(t, got[1].Done.FellBackToUnary)
	require.Equal(t, "hello world", got[1].Done.FinalText)
}

func TestBufferedFallback_NoExecutorEmitsErrorThenDone(t *testing.T) {
	got := runStrategy(t, context.Background(), &BufferedFallback{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestBufferedFallback_ExecutorErrorPropagated(t *testing.T) {
	strat := &BufferedFallback{Executor: &fakeExecutor{err: errors.New("boom")}}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom([]byte("x")))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.EqualError(t, got[0].Error, "boom")
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
	require.True(t, got[1].Done.FellBackToUnary)
}

func TestBufferedFallback_CtxCancelEndsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Open chunk channel that never closes; ctx cancel must drive exit.
	chunks := make(chan sttchain.AudioChunk)
	got := runStrategy(t, ctx, &BufferedFallback{Executor: &fakeExecutor{res: &sttchain.Result{}}}, sttchain.StreamStart{}, chunks)
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
}

// ----- Passthrough -----

func TestPassthrough_NoProviderEmitsErrorThenDone(t *testing.T) {
	got := runStrategy(t, context.Background(), &Passthrough{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestPassthrough_ForwardsProviderEvents(t *testing.T) {
	prov := &fakeProvider{
		tier:   sttchain.TierBYOK,
		traits: sttchain.ProviderTraits{Stream: true},
		stream: func(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
			out := make(chan sttchain.StreamEvent, 3)
			out <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: "hel"}}
			out <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "hello"}}
			out <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello"}}
			close(out)
			return out, nil
		},
	}
	got := runStrategy(t, context.Background(), &Passthrough{Provider: prov}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 3)
	require.Equal(t, sttchain.StreamEventPartial, got[0].Kind)
	require.Equal(t, sttchain.StreamEventSegment, got[1].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[2].Kind)
}

func TestPassthrough_SynthesisesMissingDone(t *testing.T) {
	prov := &fakeProvider{
		stream: func(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
			out := make(chan sttchain.StreamEvent, 1)
			out <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "x"}}
			close(out)
			return out, nil
		},
	}
	got := runStrategy(t, context.Background(), &Passthrough{Provider: prov}, sttchain.StreamStart{}, chunksFrom(nil))
	require.GreaterOrEqual(t, len(got), 2)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// ----- VADSegmenter -----

func TestVADSegmenter_SilenceEmitsNoSegments(t *testing.T) {
	prov := &fakeProvider{tier: sttchain.TierLocal}
	strat := &VADSegmenter{Provider: prov}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(testaudio.Silence()))

	// All-silence input never sees a voiced frame, so flushSegment skips.
	// Expect: zero Segment events, one terminal Done.
	var segments, dones int
	for _, ev := range got {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			segments++
		case sttchain.StreamEventDone:
			dones++
		}
	}
	require.Equal(t, 0, segments, "silence must not produce segments")
	require.Equal(t, 1, dones)
	require.Equal(t, 0, prov.calls)
}

func TestVADSegmenter_SpeechLikeEmitsAtLeastOneSegment(t *testing.T) {
	prov := &fakeProvider{
		tier: sttchain.TierLocal,
		transcribe: func(context.Context, sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "segment", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: time.Millisecond}, nil
		},
	}
	strat := &VADSegmenter{Provider: prov}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(testaudio.SpeechLike()))

	var segments, dones int
	for _, ev := range got {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			segments++
		case sttchain.StreamEventDone:
			dones++
		}
	}
	require.GreaterOrEqual(t, segments, 1, "speech-like input must produce at least one segment")
	require.Equal(t, 1, dones)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

func TestVADSegmenter_NoProviderEmitsErrorThenDone(t *testing.T) {
	got := runStrategy(t, context.Background(), &VADSegmenter{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestVADSegmenter_CtxCancelStillEmitsDone(t *testing.T) {
	prov := &fakeProvider{tier: sttchain.TierLocal}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks := make(chan sttchain.AudioChunk)
	got := runStrategy(t, ctx, &VADSegmenter{Provider: prov}, sttchain.StreamStart{}, chunks)
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// ----- OverlapAgree -----

func TestOverlapAgree_NoProviderEmitsErrorThenDone(t *testing.T) {
	got := runStrategy(t, context.Background(), &OverlapAgree{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestOverlapAgree_SilenceProducesOnlyDone(t *testing.T) {
	prov := &fakeProvider{
		tier: sttchain.TierLocal,
		transcribe: func(context.Context, sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "", Tier: sttchain.TierLocal}, nil
		},
	}
	// Small audio (under WindowMs at default 16kHz/2s) — no windows fire,
	// final tail transcribe sees no committed text. Done is terminal.
	got := runStrategy(t, context.Background(), &OverlapAgree{Provider: prov}, sttchain.StreamStart{}, chunksFrom(testaudio.Silence()))
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

func TestOverlapAgree_AgreementCommitsSegment(t *testing.T) {
	// Provider returns "hello world" on every call. With CommitRuns=2,
	// the second window agrees on the full prefix and we should see a
	// Segment "hello world" + final Done with FinalText.
	prov := &fakeProvider{
		tier: sttchain.TierLocal,
		transcribe: func(context.Context, sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "hello world", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"}, nil
		},
	}
	// Make enough audio for multiple windows: 4 seconds of tone at 16kHz
	// → 128_000 bytes; window=32_000 bytes (2s), advance=16_000 bytes (1s).
	audio := testaudio.SineSamples(440, 4000)
	strat := &OverlapAgree{Provider: prov, WindowMs: 2000, CommitRuns: 2, AdvanceMs: 1000}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	var sawSegment bool
	var doneEv *sttchain.DoneEvent
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventSegment {
			sawSegment = true
		}
		if ev.Kind == sttchain.StreamEventDone {
			doneEv = ev.Done
		}
	}
	require.True(t, sawSegment, "two-run agreement should commit a Segment")
	require.NotNil(t, doneEv)
	require.Equal(t, "hello world", doneEv.FinalText)
}
