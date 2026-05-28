package strategy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

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
func runStrategy(t *testing.T, ctx context.Context, strat strategy.Strategy, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) []sttchain.StreamEvent {
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

// chunksOfWindows sends n windows of windowMs PCM (silent zeros) followed
// by tailBytes of trailing audio, then closes. The OverlapAgree
// growing-buffer trigger fires once per chunk that crosses the advance
// threshold, so this is the canonical shape for behaviour tests that
// expect one Whisper call per scripted text.
func chunksOfWindows(n, windowMs, tailBytes int) <-chan sttchain.AudioChunk {
	const sampleRate, sampleBytes = 16000, 2
	wb := sampleRate * windowMs / 1000 * sampleBytes
	ch := make(chan sttchain.AudioChunk, n+1)
	for i := 0; i < n; i++ {
		ch <- sttchain.AudioChunk{Audio: make([]byte, wb)}
	}
	if tailBytes > 0 {
		ch <- sttchain.AudioChunk{Audio: make([]byte, tailBytes)}
	}
	close(ch)
	return ch
}

// voicedFrame returns a frameMs PCM frame at sampleRate that the
// frameRMS analyzer will classify as voiced (RMS well above the
// default 250 silence threshold). Filled with constant amplitude
// 1000.
func voicedFrame(sampleRate, frameMs int) []byte {
	bs := sampleRate * frameMs / 1000 * 2
	out := make([]byte, bs)
	for i := 0; i < bs; i += 2 {
		// little-endian int16 = 1000
		out[i] = 0xE8
		out[i+1] = 0x03
	}
	return out
}

// silentFrame returns frameMs PCM of zeros — RMS=0, classified silent.
func silentFrame(sampleRate, frameMs int) []byte {
	return make([]byte, sampleRate*frameMs/1000*2)
}

// voicedThenSilent returns one chunk consisting of `voicedMs` of
// voiced audio followed by `silenceMs` of silence. The RMS analyzer
// will see the voiced frames, then the silence frames, and trigger a
// VAD boundary when silenceMs ≥ the configured silence threshold.
func voicedThenSilent(sampleRate, voicedMs, silenceMs int) []byte {
	const frameMs = 20
	var out []byte
	for ms := 0; ms < voicedMs; ms += frameMs {
		out = append(out, voicedFrame(sampleRate, frameMs)...)
	}
	for ms := 0; ms < silenceMs; ms += frameMs {
		out = append(out, silentFrame(sampleRate, frameMs)...)
	}
	return out
}

// ----- BufferedFallback -----

func TestBufferedFallback_HappyPathEmitsSegmentThenDone(t *testing.T) {
	strat := &strategy.BufferedFallback{Executor: &sttmocks.FakeBatchExecutor{
		Result: &sttchain.Result{Text: "hello world", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: 5 * time.Millisecond},
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
	got := runStrategy(t, context.Background(), &strategy.BufferedFallback{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestBufferedFallback_ExecutorErrorPropagated(t *testing.T) {
	strat := &strategy.BufferedFallback{Executor: &sttmocks.FakeBatchExecutor{Err: errors.New("boom")}}
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
	got := runStrategy(t, ctx, &strategy.BufferedFallback{Executor: &sttmocks.FakeBatchExecutor{Result: &sttchain.Result{}}}, sttchain.StreamStart{}, chunks)
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
}

// ----- Passthrough -----

func TestPassthrough_NoProviderEmitsErrorThenDone(t *testing.T) {
	got := runStrategy(t, context.Background(), &strategy.Passthrough{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestPassthrough_ForwardsProviderEvents(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierBYOK, sttchain.ProviderTraits{Stream: true})
	prov.StreamFn = func(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		out := make(chan sttchain.StreamEvent, 3)
		out <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: "hel"}}
		out <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "hello"}}
		out <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello"}}
		close(out)
		return out, nil
	}
	got := runStrategy(t, context.Background(), &strategy.Passthrough{Provider: prov}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 3)
	require.Equal(t, sttchain.StreamEventPartial, got[0].Kind)
	require.Equal(t, sttchain.StreamEventSegment, got[1].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[2].Kind)
}

func TestPassthrough_SynthesisesMissingDone(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.StreamFn = func(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		out := make(chan sttchain.StreamEvent, 1)
		out <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "x"}}
		close(out)
		return out, nil
	}
	got := runStrategy(t, context.Background(), &strategy.Passthrough{Provider: prov}, sttchain.StreamStart{}, chunksFrom(nil))
	require.GreaterOrEqual(t, len(got), 2)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// ----- VADSegmenter -----

func TestVADSegmenter_SilenceEmitsNoSegments(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	strat := &strategy.VADSegmenter{Provider: prov}
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
	require.Equal(t, 0, prov.Calls)
}

func TestVADSegmenter_SpeechLikeEmitsAtLeastOneSegment(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "segment", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: time.Millisecond}, nil
	}
	strat := &strategy.VADSegmenter{Provider: prov}
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
	got := runStrategy(t, context.Background(), &strategy.VADSegmenter{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestVADSegmenter_CtxCancelStillEmitsDone(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks := make(chan sttchain.AudioChunk)
	got := runStrategy(t, ctx, &strategy.VADSegmenter{Provider: prov}, sttchain.StreamStart{}, chunks)
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// ----- OverlapAgree -----

func TestOverlapAgree_NoProviderEmitsErrorThenDone(t *testing.T) {
	got := runStrategy(t, context.Background(), &strategy.OverlapAgree{}, sttchain.StreamStart{}, chunksFrom(nil))
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
}

func TestOverlapAgree_SilenceProducesOnlyDone(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "", Tier: sttchain.TierLocal}, nil
	}
	// Small audio (under WindowMs at default 16kHz/2s) — no windows fire,
	// final tail transcribe sees no committed text. Done is terminal.
	got := runStrategy(t, context.Background(), &strategy.OverlapAgree{Provider: prov}, sttchain.StreamStart{}, chunksFrom(testaudio.Silence()))
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

func TestOverlapAgree_AgreementCommitsSegment(t *testing.T) {
	// Provider returns "hello world" on every call. With CommitRuns=2,
	// the second window agrees on the full prefix and we should see a
	// Segment "hello world" + final Done with FinalText.
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "hello world", Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"}, nil
	}
	// Make enough audio for multiple windows: 4 seconds of tone at 16kHz
	// → 128_000 bytes; window=32_000 bytes (2s), advance=16_000 bytes (1s).
	audio := testaudio.SineSamples(440, 4000)
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: 2000, CommitRuns: 2, AdvanceMs: 1000}
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
