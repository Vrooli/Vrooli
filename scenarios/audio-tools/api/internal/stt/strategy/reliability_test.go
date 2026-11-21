package strategy_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

// heapAlloc returns live heap bytes after a synchronous collection. The
// reliability tests below use it to prove a buffer is released while the
// strategy is still running; a goroutine leak or unbounded retention would
// otherwise be invisible to the event-level assertions.
func heapAlloc() uint64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

// TestVADSegmenter_ProviderFailureDoesNotStormForcedCuts is the regression for
// call amplification during an outage: once a continuous segment reaches
// MaxSegmentMs, a failed flush must be retried on a cadence, not on every 20ms
// frame. The audio is retained either way, so the only difference should be
// provider load and error-event volume.
func TestVADSegmenter_ProviderFailureDoesNotStormForcedCuts(t *testing.T) {
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.Err = errors.New("provider unavailable")
	strat := &strategy.VADSegmenter{
		Provider:          prov,
		SilenceMs:         60000,
		PreRollMs:         0,
		TrailingPadMs:     0,
		MaxSegmentMs:      200,
		PreviewIntervalMs: 0,
	}

	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(1, 2000))

	// 2s of continuous speech at a 200ms bound is 100 frames. Retrying the
	// retained window on every frame produces ~90 requests; a 250ms retry
	// cadence produces roughly 8 plus the terminal flush.
	require.LessOrEqual(t, prov.CallCount(), 15,
		"a failing provider must not be called once per frame after MaxSegmentMs")
	var errs int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError {
			errs++
		}
	}
	require.LessOrEqual(t, errs, 15, "a retained window must not surface one error per frame")
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind, "failure must still terminate with Done")
}

// TestVADSegmenter_VadTicksSurviveForcedCutCompaction pins the frame-counter
// translation that accompanies buffer compaction. After every forced
// MaxSegmentMs cut the buffer shifts backwards; leaving lastEmitFrameIdx in the
// old coordinate makes the throttle delta negative, so sustained voiced ticks
// stall for the rest of a continuous utterance.
func TestVADSegmenter_VadTicksSurviveForcedCutCompaction(t *testing.T) {
	rec := &recordingProvider{unique: true}
	strat := &strategy.VADSegmenter{
		Provider:          newRecordingProvider(t, rec, "continuous"),
		SilenceMs:         60000,
		PreRollMs:         100,
		TrailingPadMs:     0,
		MaxSegmentMs:      1000,
		PreviewIntervalMs: 0,
	}

	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(1, 4000))

	var voicedTicks int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventVadState && ev.VadState != nil && ev.VadState.Voiced {
			voicedTicks++
		}
	}
	// 4s of continuous speech crosses four forced cuts. The cadence is ~2Hz
	// while voiced, so normal operation emits several ticks after the first
	// cut; stale frame coordinates emitted the first tick and then went quiet.
	require.GreaterOrEqual(t, voicedTicks, 4,
		"voiced VAD ticks must keep their cadence after forced-cut buffer compaction")
}

// TestVADSegmenter_LongLeadingSilenceDoesNotRetainWholeTurn is the regression
// for unbounded growth before the first voiced frame. The transport resets its
// idle deadline on every audio frame, so a silent open microphone can run for
// minutes; no segment cut fires until speech arrives. Retaining all of that
// silence makes memory grow with wall-clock session length.
func TestVADSegmenter_LongLeadingSilenceDoesNotRetainWholeTurn(t *testing.T) {
	const (
		silenceMs = 5 * 60 * 1000
		chunkMs   = 100
		speechMs  = 200
	)
	silenceChunk := testaudio.SilenceSamples(chunkMs)
	before := heapAlloc()
	var measured atomic.Uint64
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		measured.CompareAndSwap(0, heapAlloc())
		return &sttchain.Result{Text: "after silence", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	strat := &strategy.VADSegmenter{
		Provider:          prov,
		SilenceMs:         1200,
		PreRollMs:         800,
		PreviewIntervalMs: 0,
	}

	// Stream audio in capture-sized chunks. A single 5-minute chunk would let
	// the trailing voiced frame set hasVoiced before the trim ever runs; real
	// capture delivers silence while the strategy is still waiting for speech.
	chunksIn := make(chan sttchain.AudioChunk)
	go func() {
		defer close(chunksIn)
		for sent := 0; sent < silenceMs; sent += chunkMs {
			chunksIn <- sttchain.AudioChunk{Audio: silenceChunk}
		}
		chunksIn <- sttchain.AudioChunk{Audio: testaudio.SineSamples(440, speechMs)}
	}()

	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksIn)

	require.NotZero(t, measured.Load(), "the final voiced segment must have been transcribed")
	delta := measured.Load() - before
	// The whole turn is ~9.6MB. The bounded buffer keeps roughly a second of
	// pre-roll around the first voiced frame.
	require.Less(t, delta, uint64(4<<20),
		"leading silence must be evicted instead of retained for the whole turn; retained %d bytes", delta)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// TestOverlapAgree_LongSessionReleasesCommittedPCM proves the growing-window
// strategy drops audio behind its coverage cursor. The algorithm is bounded in
// transcription terms by MaxWindowMs, but the raw PCM slice previously grew for
// the entire turn and was retained until Run returned.
func TestOverlapAgree_LongSessionReleasesCommittedPCM(t *testing.T) {
	const chunks = 150 // 150 * 2s = 5 minutes of audio
	chunk := testaudio.SilenceSamples(2000)

	before := heapAlloc()
	var armed atomic.Bool
	var measured atomic.Uint64
	var calls atomic.Int64
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		n := calls.Add(1)
		if armed.Load() {
			measured.CompareAndSwap(0, heapAlloc())
		}
		return &sttchain.Result{
			Text: fmt.Sprintf("word%d", n), Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake",
		}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider:    prov,
		Trigger:     strategy.TriggerStopwatch,
		WindowMs:    100,
		AdvanceMs:   100,
		CommitRuns:  2,
		MaxWindowMs: 500,
	}

	chunksIn := make(chan sttchain.AudioChunk)
	go func() {
		for i := 0; i < chunks; i++ {
			chunksIn <- sttchain.AudioChunk{Audio: chunk}
		}
		// Arm only after the last chunk has been handed to the strategy, so
		// the measurement includes the full turn's PCM.
		armed.Store(true)
		close(chunksIn)
	}()

	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksIn)

	require.NotZero(t, measured.Load(), "the session must have transcribed audio")
	delta := measured.Load() - before
	// Before compaction the raw PCM slice held the full ~9.6MB turn even
	// though only about one MaxWindowMs of it was uncommitted.
	require.Less(t, delta, uint64(4<<20),
		"committed PCM must be released while the session is still running; retained %d bytes", delta)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// TestPassthrough_CtxCancelStillEmitsTerminalDone pins the strategy contract:
// cancellation is a stream end and must still produce a Done. Without it the
// transport cannot distinguish a drained tail from a dropped one.
func TestPassthrough_CtxCancelStillEmitsTerminalDone(t *testing.T) {
	started := make(chan struct{})
	prov := sttmocks.NewFakeProvider(sttchain.TierBYOK, sttchain.ProviderTraits{Stream: true})
	prov.StreamFn = func(ctx context.Context, _ sttchain.StreamStart, _ <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		vendor := make(chan sttchain.StreamEvent)
		close(started)
		go func() {
			<-ctx.Done()
			close(vendor)
		}()
		return vendor, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()

	// An open chunk channel models a live capture that has not ended; ctx
	// cancellation is the only teardown signal.
	chunks := make(chan sttchain.AudioChunk)
	got := runStrategy(t, ctx, &strategy.Passthrough{Provider: prov}, sttchain.StreamStart{}, chunks)

	require.NotEmpty(t, got, "cancellation must not end the event stream silently")
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind,
		"cancellation must still emit a terminal Done so the transport does not count a dropped tail")
}
