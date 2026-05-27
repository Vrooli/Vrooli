package strategy_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/strategy"
)

// scriptedProvider returns texts[i] for the i-th Transcribe call and repeats the
// last entry once exhausted, modelling how a sliding-window transcriber's
// hypothesis evolves across windows. errAt (>=0) makes one call return an error.
func scriptedProvider(errAt int, texts ...string) *sttmocks.FakeProvider {
	p := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	i := -1
	p.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		i++
		if i == errAt {
			return nil, errors.New("provider boom")
		}
		txt := texts[len(texts)-1]
		if i < len(texts) {
			txt = texts[i]
		}
		return &sttchain.Result{Text: txt, Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"}, nil
	}
	return p
}

// windowPCM returns n non-overlapping windows of silent PCM for windowMs at
// 16 kHz mono s16le, plus tailBytes extra to optionally trigger the tail flush.
// Content is irrelevant — the provider is scripted — so zeros suffice.
func windowPCM(n, windowMs, tailBytes int) []byte {
	const sampleRate, sampleBytes = 16000, 2
	wb := sampleRate * windowMs / 1000 * sampleBytes
	return make([]byte, n*wb+tailBytes)
}

// segmentsAndFinal collects committed Segment texts (in order) and the terminal
// Done.FinalText. Requires exactly one Done, last.
func segmentsAndFinal(t *testing.T, got []sttchain.StreamEvent) (segs []string, partials []string, final string) {
	t.Helper()
	var dones int
	for _, ev := range got {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			segs = append(segs, ev.Segment.Text)
		case sttchain.StreamEventPartial:
			partials = append(partials, ev.Partial.Text)
		case sttchain.StreamEventDone:
			dones++
			final = ev.Done.FinalText
		}
	}
	require.Equal(t, 1, dones, "exactly one terminal Done")
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind, "Done must be last")
	return segs, partials, final
}

// useWindowMs picks a small window so test buffers stay tiny while keeping the
// non-overlapping (AdvanceMs == WindowMs) invariant: each window is one provider
// call and N windows of audio yields exactly N calls (+1 if a tail remains).
const useWindowMs = 100

// TestOverlapAgree_ProgressiveCommit proves LocalAgreement commits a growing
// prefix one confirmed token-run at a time, emits only the new tail per Segment,
// and reports the full committed text as FinalText.
func TestOverlapAgree_ProgressiveCommit(t *testing.T) {
	prov := scriptedProvider(-1, "the", "the quick", "the quick brown", "the quick brown")
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(windowPCM(4, useWindowMs, 0)))

	segs, partials, final := segmentsAndFinal(t, got)
	require.Equal(t, []string{"the", " quick", " brown"}, segs, "each Segment carries only the newly committed tail")
	require.Equal(t, "the quick brown", strings.TrimSpace(strings.Join(segs, "")), "segments reconstruct the committed text")
	require.Equal(t, "the quick brown", final)
	require.NotEmpty(t, partials, "the first, not-yet-agreed window emits a Partial")
}

// TestOverlapAgree_DivergenceKeepsPriorCommit proves the divergence guard: once
// a prefix is committed, a later run that agrees on a DIFFERENT continuation is
// surfaced as a Partial only and never overwrites the committed prefix.
func TestOverlapAgree_DivergenceKeepsPriorCommit(t *testing.T) {
	prov := scriptedProvider(-1, "alpha beta", "alpha beta", "alpha zeta omega", "alpha zeta omega")
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(windowPCM(4, useWindowMs, 0)))

	segs, partials, final := segmentsAndFinal(t, got)
	require.Equal(t, "alpha beta", strings.TrimSpace(strings.Join(segs, "")), "committed prefix is not overwritten by the divergent run")
	require.Equal(t, "alpha beta", final)
	for _, s := range segs {
		require.NotContains(t, s, "zeta", "divergent continuation must never commit")
	}
	require.Contains(t, partials, "alpha zeta omega", "the divergent run is surfaced as a live Partial")
}

// TestOverlapAgree_TailFlushCommitsRemainder proves the channel-close tail
// transcribe commits trailing audio that never got a second confirming window.
func TestOverlapAgree_TailFlushCommitsRemainder(t *testing.T) {
	prov := scriptedProvider(-1, "hello", "hello", "hello there")
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	// 2 full windows + a partial-window tail -> 2 window calls + 1 tail call.
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(windowPCM(2, useWindowMs, 400)))

	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, "hello there", strings.TrimSpace(strings.Join(segs, "")))
	require.Equal(t, "hello there", final)
}

// TestOverlapAgree_TailFlushNoSpuriousSegment proves the tail flush does NOT
// emit a Segment when the committed text already covers the tail transcript.
func TestOverlapAgree_TailFlushNoSpuriousSegment(t *testing.T) {
	prov := scriptedProvider(-1, "done", "done", "done")
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(windowPCM(2, useWindowMs, 400)))

	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, []string{"done"}, segs, "no spurious final Segment when committed already covers the tail")
	require.Equal(t, "done", final)
}

// TestOverlapAgree_CommitRunsThree proves CommitRuns=3 (the live operator value)
// requires three consecutive agreeing windows before committing — the first two
// only emit Partials.
func TestOverlapAgree_CommitRunsThree(t *testing.T) {
	prov := scriptedProvider(-1, "a", "a", "a")
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 3}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(windowPCM(3, useWindowMs, 0)))

	segs, partials, final := segmentsAndFinal(t, got)
	require.Equal(t, []string{"a"}, segs, "prefix commits only on the third agreeing run")
	require.Equal(t, "a", final)
	require.GreaterOrEqual(t, len(partials), 2, "the first two runs (below CommitRuns) emit Partials, not Segments")
}

// TestOverlapAgree_ProviderErrorMidStreamContinues proves a transcribe error on
// one window emits a single Error event and the stream keeps going — a later
// window still commits and the run ends with a terminal Done (errors advance,
// they do not abort).
func TestOverlapAgree_ProviderErrorMidStreamContinues(t *testing.T) {
	prov := scriptedProvider(1, "alpha", "", "alpha") // window index 1 errors
	strat := &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(windowPCM(3, useWindowMs, 0)))

	var errs int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError {
			errs++
		}
	}
	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, 1, errs, "exactly one Error event for the one failed window")
	require.Equal(t, "alpha", strings.TrimSpace(strings.Join(segs, "")), "a window after the error still commits")
	require.Equal(t, "alpha", final)
}

// TestOverlapAgree_CtxCancelStillEmitsDone proves mid-stream cancellation still
// produces a terminal Done (parity with the other strategies).
func TestOverlapAgree_CtxCancelStillEmitsDone(t *testing.T) {
	prov := scriptedProvider(-1, "x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks := make(chan sttchain.AudioChunk) // never closes
	got := runStrategy(t, ctx, &strategy.OverlapAgree{Provider: prov, WindowMs: useWindowMs, AdvanceMs: useWindowMs}, sttchain.StreamStart{}, chunks)
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}
