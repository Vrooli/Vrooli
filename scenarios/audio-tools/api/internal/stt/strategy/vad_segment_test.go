package strategy_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

// vadStateEvents extracts the VadState ticks from a strategy event run.
func vadStateEvents(events []sttchain.StreamEvent) []*sttchain.VadStateEvent {
	var out []*sttchain.VadStateEvent
	for _, ev := range events {
		if ev.Kind == sttchain.StreamEventVadState && ev.VadState != nil {
			out = append(out, ev.VadState)
		}
	}
	return out
}

// TestVADSegmenter_EmitsVadStateOnTransition asserts that a speech→silence
// transition produces at least one vad-state tick with Voiced=false, and
// that voiced frames produce at least one Voiced=true tick.
func TestVADSegmenter_EmitsVadStateOnTransition(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "segment", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake", Latency: time.Millisecond}, nil
	}
	strat := &strategy.VADSegmenter{Provider: prov}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(testaudio.SpeechLike()))

	ticks := vadStateEvents(got)
	require.NotEmpty(t, ticks, "expected at least one vad-state tick on a speech→silence run")

	var voicedSeen, silenceSeen bool
	var prevSeq uint64
	for i, ev := range ticks {
		if ev.Voiced {
			voicedSeen = true
		} else {
			silenceSeen = true
		}
		require.Equal(t, int64(1200), ev.SilenceTimeoutMs, "tick #%d must echo SilenceMs=1200 default", i)
		// Monotonic per-stream tick sequence.
		require.Greater(t, ev.TickSeq, prevSeq, "tick #%d sequence must be strictly increasing", i)
		prevSeq = ev.TickSeq
	}
	require.True(t, voicedSeen, "expected at least one Voiced=true tick during sine-tone segment")
	require.True(t, silenceSeen, "expected at least one Voiced=false tick during the 700 ms silence segment")
}

// TestVADSegmenter_NoVadStateBeforeFirstVoice asserts that pure-silence
// input produces zero vad-state events — the plan §8 contract:
// "no event before first speech in a segment".
func TestVADSegmenter_NoVadStateBeforeFirstVoice(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	strat := &strategy.VADSegmenter{Provider: prov}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(testaudio.Silence()))

	ticks := vadStateEvents(got)
	require.Empty(t, ticks, "pure-silence input must not emit any vad-state ticks before first voiced frame")
}

// TestVADSegmenter_VadStateThrottledInSilence asserts the throttle ceiling
// for sustained silence: at most ~one tick per ~50 ms of silence (the
// throttle is wall-clock, so we tolerate small jitter from CI machines).
// The fixture is 100 ms sine + 1000 ms silence — that's 50 silent frames at
// 20 ms/frame. We assert the silence-side tick count stays modest.
func TestVADSegmenter_VadStateThrottledInSilence(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "x", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	// Use a long SilenceMs so the segment doesn't flush mid-run; we just
	// want to count emission cadence.
	strat := &strategy.VADSegmenter{Provider: prov, SilenceMs: 5000}
	audio := append(testaudio.SineSamples(440, 100), testaudio.SilenceSamples(1000)...)
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	ticks := vadStateEvents(got)
	// 1000 ms of silence at ≤20 Hz max ceiling = ≤20 silence ticks. Allow
	// a small headroom for the leading voiced tick + the transition tick.
	require.LessOrEqual(t, len(ticks), 25, "throttle ceiling exceeded: got %d ticks", len(ticks))
}

// TestVADSegmenter_VadStateBeforeFlushReachesTimeout asserts that the last
// vad-state tick before a segment flush carries SilenceElapsedMs close to
// SilenceMs (within one frame). This is the property the UI ring relies on
// to visibly complete at the segment-cut moment.
func TestVADSegmenter_VadStateBeforeFlushReachesTimeout(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "seg", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	// 700 ms is the default; the SpeechLike fixture has exactly 700 ms of
	// silence between two tones, so the first silence run is what we want.
	strat := &strategy.VADSegmenter{Provider: prov}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(testaudio.SpeechLike()))

	// Find the last silence tick that came before the first Segment event.
	var lastSilenceBeforeSeg *sttchain.VadStateEvent
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventSegment {
			break
		}
		if ev.Kind == sttchain.StreamEventVadState && ev.VadState != nil && !ev.VadState.Voiced {
			lastSilenceBeforeSeg = ev.VadState
		}
	}
	require.NotNil(t, lastSilenceBeforeSeg, "expected at least one silence tick before the segment flush")
	// Allow one frame of slack (20 ms).
	require.GreaterOrEqual(t, lastSilenceBeforeSeg.SilenceElapsedMs, int64(700-20),
		"last silence tick must reach close to SilenceMs (got %d)", lastSilenceBeforeSeg.SilenceElapsedMs)
}

// TestVADSegmenter_EmitsTickAtOrPastTimeout is the auto-stop SSOT regression.
// The client's decideAutoStop fires ONLY on a tick with
// SilenceElapsedMs >= SilenceTimeoutMs. The emit throttle once skipped that
// exact threshold-crossing tick (the last emitted elapsed landed a frame
// short, e.g. 2960 of 3000) while the segment cut reset the silence counter on
// the same frame — so the client never observed the threshold and the
// auto-stop ring hung visually full without ever stopping. Unlike
// "...ReachesTimeout" above (which only checks "close to"), this asserts the
// strict >= contract the decision boundary depends on.
func TestVADSegmenter_EmitsTickAtOrPastTimeout(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "seg", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	strat := &strategy.VADSegmenter{Provider: prov, SilenceMs: 1000}
	// 100 ms tone, then 1400 ms silence — well past the 1000 ms threshold, so
	// the silence counter crosses silenceFramesNeeded and the cut fires.
	audio := append(testaudio.SineSamples(440, 100), testaudio.SilenceSamples(1400)...)
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	ticks := vadStateEvents(got)
	require.NotEmpty(t, ticks, "expected vad-state ticks")
	reached := false
	for _, ev := range ticks {
		require.Greater(t, ev.SilenceTimeoutMs, int64(0), "timeout must be positive")
		if ev.SilenceElapsedMs >= ev.SilenceTimeoutMs {
			reached = true
		}
	}
	require.True(t, reached,
		"auto-stop contract: a vad-state tick must reach SilenceElapsedMs >= SilenceTimeoutMs at the silence threshold (decideAutoStop depends on it)")
}

// TestVADSegmenter_SetsSilenceTimedOutAtTimeout is the latch-signal contract.
// The client latches one-shot auto-stop on the self-describing SilenceTimedOut
// flag rather than re-deriving the threshold from a float compare inside a
// 250 ms freshness window (the old wedge: the single threshold tick aged out
// of the window before the RAF loop consumed it, and no further ticks arrive
// after the segment cut). This asserts the server sets SilenceTimedOut exactly
// on the threshold-crossing tick, and never before the threshold is reached.
func TestVADSegmenter_SetsSilenceTimedOutAtTimeout(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "seg", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	strat := &strategy.VADSegmenter{Provider: prov, SilenceMs: 1000}
	audio := append(testaudio.SineSamples(440, 100), testaudio.SilenceSamples(1400)...)
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	ticks := vadStateEvents(got)
	require.NotEmpty(t, ticks, "expected vad-state ticks")
	timedOutSeen := false
	for i, ev := range ticks {
		if ev.SilenceTimedOut {
			timedOutSeen = true
			require.GreaterOrEqual(t, ev.SilenceElapsedMs, ev.SilenceTimeoutMs,
				"tick #%d set SilenceTimedOut but elapsed (%d) is below timeout (%d)",
				i, ev.SilenceElapsedMs, ev.SilenceTimeoutMs)
		} else {
			require.Less(t, ev.SilenceElapsedMs, ev.SilenceTimeoutMs,
				"tick #%d left SilenceTimedOut=false but elapsed (%d) already reached timeout (%d)",
				i, ev.SilenceElapsedMs, ev.SilenceTimeoutMs)
		}
	}
	require.True(t, timedOutSeen,
		"auto-stop latch contract: exactly one threshold-crossing tick must carry SilenceTimedOut=true")
}
