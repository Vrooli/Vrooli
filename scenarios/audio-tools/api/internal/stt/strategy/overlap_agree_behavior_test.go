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

// scriptedProvider returns texts[i] for the i-th Transcribe call and
// repeats the last entry once exhausted. errAt (>=0) makes one call
// return an error. Words is always empty — tests that rely on
// word-aligned advance use scriptedWordsProvider.
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

// scriptedHypothesis is one call's worth of mocked Whisper output: text
// plus per-word end timestamps in seconds. Word boundaries drive
// OverlapAgree's committedAudioBytes advance.
type scriptedHypothesis struct {
	text      string
	wordEnds  []float64 // seconds, one per whitespace-delimited word in text
	receivedN *int      // optional: incremented with len(req.Audio) every call
	gotAudio  *[][]byte // optional: appends a copy of req.Audio on every call
}

// scriptedWordsProvider returns one scriptedHypothesis per call,
// repeating the last entry once exhausted. Each hypothesis fans out the
// text into TimedWord entries with start=0 and the supplied end
// timestamps, which is what OverlapAgree's word-aligned advance reads.
func scriptedWordsProvider(scripts ...scriptedHypothesis) *sttmocks.FakeProvider {
	p := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	i := -1
	p.TranscribeFn = func(_ context.Context, req sttchain.Request) (*sttchain.Result, error) {
		i++
		idx := i
		if idx >= len(scripts) {
			idx = len(scripts) - 1
		}
		s := scripts[idx]
		if s.gotAudio != nil {
			cp := make([]byte, len(req.Audio))
			copy(cp, req.Audio)
			*s.gotAudio = append(*s.gotAudio, cp)
		}
		if s.receivedN != nil {
			*s.receivedN = len(req.Audio)
		}
		words := strings.Fields(s.text)
		out := make([]sttchain.TimedWord, len(words))
		for k, w := range words {
			end := 0.0
			if k < len(s.wordEnds) {
				end = s.wordEnds[k]
			}
			start := 0.0
			if k > 0 && k-1 < len(s.wordEnds) {
				start = s.wordEnds[k-1]
			}
			out[k] = sttchain.TimedWord{Word: w, Start: start, End: end, Prob: 0.9}
		}
		return &sttchain.Result{Text: s.text, Words: out, Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base"}, nil
	}
	return p
}

// segmentsAndFinal collects committed Segment texts (in order) and the
// terminal Done.FinalText. Requires exactly one Done, last.
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

// useWindowMs picks a small window so test buffers stay tiny while
// keeping AdvanceMs == WindowMs (one chunk per iteration trigger).
const useWindowMs = 100

// TestOverlapAgree_GrowingBufferCommitsDuringStream proves the new
// algorithm emits multiple Segment events MID-UTTERANCE — not at
// end-of-stream as the prior sliding-window implementation did. Each
// scripted hypothesis is the model's current best for the
// growing-buffer audio prefix; LocalAgreement-2 commits each newly
// agreed token-run as a separate Segment.
func TestOverlapAgree_GrowingBufferCommitsDuringStream(t *testing.T) {
	prov := scriptedProvider(-1, "hello", "hello world", "hello world how", "hello world how")
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(4, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	// Find the index of the Done event — every Segment we collected
	// arrived before it, by construction of segmentsAndFinal.
	require.GreaterOrEqual(t, len(segs), 2, "at least two Segments must arrive BEFORE Done (incremental commits)")
	require.Equal(t, "hello world how", strings.TrimSpace(strings.Join(segs, "")), "segments reconstruct the committed text")
	require.Equal(t, "hello world how", final)
}

// TestOverlapAgree_NoReEmissionAcrossCommits asserts the concatenation of
// every Segment.Text equals the Done.FinalText (modulo whitespace) — i.e.
// no committed word is ever re-emitted, no committed word is ever lost.
// This is the contract a downstream consumer needs to render a final
// transcript by joining Segments instead of waiting for Done.
func TestOverlapAgree_NoReEmissionAcrossCommits(t *testing.T) {
	prov := scriptedProvider(-1, "hello", "hello world", "hello world how", "hello world how")
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(4, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, normalizeWS(final), normalizeWS(strings.Join(segs, "")),
		"Done.FinalText must equal the whitespace-normalized concatenation of Segment.Text values")
}

// TestOverlapAgree_WordAlignedAdvance proves committedAudioBytes
// advances to the END timestamp of the last committed word so the next
// iteration's audio starts immediately after the committed material.
// The scripted provider records the audio slice each call received; we
// assert call N+1's slice is exactly call N's slice minus the
// word-end-aligned prefix.
func TestOverlapAgree_WordAlignedAdvance(t *testing.T) {
	const sampleRate, sampleBytes = 16000, 2
	// Each chunk is one useWindowMs = 100ms = 3200 bytes at 16kHz s16le.
	// We word-end the first call's only word ("hello") at 0.05s; call 2
	// agrees on "hello" and the algorithm commits — advance must move
	// the cursor to 0.05*16000*2 = 1600 bytes. Call 3's audio must
	// therefore be (4 chunks accumulated by then) - 1600 bytes long.
	var calls [][]byte
	prov := scriptedWordsProvider(
		scriptedHypothesis{text: "hello", wordEnds: []float64{0.05}, gotAudio: &calls},
		scriptedHypothesis{text: "hello", wordEnds: []float64{0.05}, gotAudio: &calls},
		scriptedHypothesis{text: "hello world", wordEnds: []float64{0.05, 0.15}, gotAudio: &calls},
		scriptedHypothesis{text: "hello world", wordEnds: []float64{0.05, 0.15}, gotAudio: &calls},
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2, SampleRate: sampleRate, UseWordTimestampAdvance: true}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(4, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	require.GreaterOrEqual(t, len(segs), 1)
	require.Equal(t, "hello world", final)
	// Calls captured: at least 3 (the first triggers Partial, the
	// second commits "hello" and advances, the third sees post-advance
	// audio). The tail flush after channel close adds one more.
	require.GreaterOrEqual(t, len(calls), 3)
	expectedAdvanceBytes := int(0.05*float64(sampleRate)) * sampleBytes // 1600
	chunkBytes := sampleRate * useWindowMs / 1000 * sampleBytes         // 3200
	// After call 2 commits, committedAudioBytes = 1600. By the time
	// call 3 runs, two more chunks have arrived: pcm=2*chunkBytes=6400
	// at call 2, plus one more chunk before call 3 (chunk #3 triggers
	// it) → pcm=9600. Call 3 receives pcm[1600:] = 8000 bytes.
	require.Equal(t, 3*chunkBytes-expectedAdvanceBytes, len(calls[2]),
		"call 3 audio must start AFTER the committed word's end timestamp")
}

// TestOverlapAgree_DivergenceKeepsPriorCommit_InStream proves the
// in-stream divergence guard: while audio is still streaming, once a
// prefix is committed, an immediately-following hypothesis that agrees
// on a DIFFERENT continuation is surfaced as a Partial only — it does
// NOT overwrite the committed prefix as a mid-stream commit. This is
// the "the model wandered" rejection case, distinct from the
// channel-close tail flush (which always emits).
//
// To assert the in-stream rejection without the tail flush masking it,
// the audio ends with a hypothesis that AGREES with committed (the
// canonical "no new content" terminal state), so no tail Segment is
// produced. Tail-flush emission is covered by the dedicated tail tests.
func TestOverlapAgree_DivergenceKeepsPriorCommit_InStream(t *testing.T) {
	prov := scriptedProvider(-1,
		"alpha beta", "alpha beta", // commits "alpha beta"
		"alpha zeta omega", "alpha zeta omega", // divergent: in-stream → rejected as Partial
		"alpha beta", "alpha beta", // model recovers; commit stays "alpha beta"
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(6, useWindowMs, 0))

	segs, partials, _ := segmentsAndFinal(t, got)
	for _, s := range segs {
		require.NotContains(t, s, "zeta", "divergent continuation must never commit mid-stream")
	}
	require.Equal(t, "alpha beta", strings.TrimSpace(strings.Join(segs, "")), "committed prefix is not overwritten by the divergent run")
	require.Contains(t, partials, "alpha zeta omega", "the divergent run is surfaced as a live Partial")
}

// TestOverlapAgree_TailFlushEmitsEvenOnDivergence proves the
// channel-close tail flush is UNCONDITIONAL: unsettled audio at end of
// stream is always emitted (modulo prompt-regurg dedupe), even if its
// content doesn't agree with committed via overlap. This is the
// "never lose audio" guarantee — long utterances that never reach
// in-stream agreement must still surface their content when the
// stream ends.
func TestOverlapAgree_TailFlushEmitsEvenOnDivergence(t *testing.T) {
	prov := scriptedProvider(-1,
		"alpha beta", "alpha beta", // commits "alpha beta" in-stream
		"completely different content at end", // tail flush call: no overlap with committed
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(2, useWindowMs, 400))

	segs, _, final := segmentsAndFinal(t, got)
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Contains(t, joined, "alpha beta", "in-stream commit must be present")
	require.Contains(t, joined, "completely different content at end", "tail-flush divergent content must be appended, not dropped")
	require.Equal(t, joined, final, "Done.FinalText must include the tail content")
}

// TestOverlapAgree_ForceCommitOnMaxWindowEmitsSegment proves the
// MaxWindowMs ceiling never DROPS audio. When uncommitted audio
// exceeds the ceiling (e.g., the model keeps producing divergent
// hypotheses so no natural agreement fires), the strategy force-
// commits the whole uncommitted window as a single Segment — the
// user gets the audio's transcript, not silence + an error.
func TestOverlapAgree_ForceCommitOnMaxWindowEmitsSegment(t *testing.T) {
	// Eight 100ms chunks = 800ms of audio. MaxWindowMs=300 forces a
	// commit once uncommitted audio exceeds 300ms. The scripted
	// provider returns divergent text per call (no natural
	// agreement); the force-commit takes whatever the model returns
	// at the moment the ceiling trips.
	prov := scriptedProvider(-1, "alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta")
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2, MaxWindowMs: 300}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(8, useWindowMs, 0))

	segs, _, _ := segmentsAndFinal(t, got)
	require.NotEmpty(t, segs, "MaxWindowMs trip must emit a Segment — audio must NOT be dropped silently")
	// No "forced cursor advance" Error events — the success path
	// emits a Segment, not an Error.
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError && ev.Error != nil {
			require.NotContains(t, ev.Error.Error(), "audio dropped",
				"old drop-audio path must not be reachable on the success branch")
		}
	}
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind, "stream still terminates cleanly")
}

// TestOverlapAgree_TailFlushEmitsRemainder proves the channel-close
// tail transcribe commits trailing audio that never got a second
// confirming iteration. This covers the last fragment of an utterance
// that ends before LocalAgreement-N can confirm it.
func TestOverlapAgree_TailFlushEmitsRemainder(t *testing.T) {
	prov := scriptedProvider(-1, "hello", "hello", "hello there")
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(2, useWindowMs, 400))

	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, "hello there", strings.TrimSpace(strings.Join(segs, "")))
	require.Equal(t, "hello there", final)
}

// TestOverlapAgree_TailFlushNoSpuriousSegment proves the tail flush
// does NOT emit a Segment when the committed text already covers the
// tail transcript.
func TestOverlapAgree_TailFlushNoSpuriousSegment(t *testing.T) {
	prov := scriptedProvider(-1, "done", "done", "done")
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(2, useWindowMs, 400))

	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, []string{"done"}, segs, "no spurious final Segment when committed already covers the tail")
	require.Equal(t, "done", final)
}

// TestOverlapAgree_CommitRunsThree proves CommitRuns=3 requires three
// consecutive agreeing iterations before committing — the first two only
// emit Partials.
func TestOverlapAgree_CommitRunsThree(t *testing.T) {
	prov := scriptedProvider(-1, "a", "a", "a")
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 3}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(3, useWindowMs, 0))

	segs, partials, final := segmentsAndFinal(t, got)
	require.Equal(t, []string{"a"}, segs, "prefix commits only on the third agreeing iteration")
	require.Equal(t, "a", final)
	require.GreaterOrEqual(t, len(partials), 2, "the first two iterations (below CommitRuns) emit Partials, not Segments")
}

// TestOverlapAgree_ProviderErrorMidStreamContinues proves a transcribe
// error on one iteration emits a single Error event and the stream
// keeps going — a later iteration still commits and the run ends with a
// terminal Done (errors advance, they do not abort).
func TestOverlapAgree_ProviderErrorMidStreamContinues(t *testing.T) {
	prov := scriptedProvider(1, "alpha", "", "alpha") // call index 1 errors
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(3, useWindowMs, 0))

	var errs int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError {
			errs++
		}
	}
	segs, _, final := segmentsAndFinal(t, got)
	require.Equal(t, 1, errs, "exactly one Error event for the one failed iteration")
	require.Equal(t, "alpha", strings.TrimSpace(strings.Join(segs, "")), "an iteration after the error still commits")
	require.Equal(t, "alpha", final)
}

// TestOverlapAgree_PromptRegurgitationNoDuplicate proves the
// mergeAgreed dedupe defense: when Whisper occasionally outputs
// `<initial_prompt> <new_words>` instead of just `<new_words>`, the
// strategy must NOT re-emit the committed prefix as a new Segment.
// This is the user-reported "duplicate text" regression the dedupe
// layer guards against. Independent of the growing-buffer rewrite —
// the defense still runs on the agreed prefix.
func TestOverlapAgree_PromptRegurgitationNoDuplicate(t *testing.T) {
	prov := scriptedProvider(-1,
		"hello world", "hello world",
		"hello world hello world how", "hello world hello world how",
		"hello world how are you", "hello world how are you",
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(6, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	joined := strings.TrimSpace(strings.Join(segs, ""))
	for _, word := range []string{"hello", "world", "how", "are", "you"} {
		count := strings.Count(" "+joined+" ", " "+word+" ")
		require.Equalf(t, 1, count, "word %q should appear once across segments, joined=%q", word, joined)
	}
	require.Equal(t, "hello world how are you", joined, "segments reconstruct the canonical text without duplication")
	require.Equal(t, "hello world how are you", final)
}

func TestOverlapAgree_BoundsRollingPromptContext(t *testing.T) {
	var prompts []string
	provider := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	provider.TranscribeFn = func(_ context.Context, req sttchain.Request) (*sttchain.Result, error) {
		prompts = append(prompts, req.InitialPrompt)
		return &sttchain.Result{
			Text:       "one two three four five six seven eight nine ten eleven twelve",
			Tier:       sttchain.TierLocal,
			ProviderID: "whisper",
			ModelID:    "base",
		}, nil
	}
	strat := &strategy.OverlapAgree{Provider: provider, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	_ = runStrategy(t, context.Background(), strat, sttchain.StreamStart{InitialPrompt: "operator hint"}, chunksOfWindows(6, useWindowMs, 0))

	require.GreaterOrEqual(t, len(prompts), 2)
	for _, prompt := range prompts[1:] {
		require.LessOrEqual(t, len(strings.Fields(prompt)), 10, "rolling prompt must stay bounded: %q", prompt)
		require.Contains(t, prompt, "operator hint")
	}
	// The committed transcript is deliberately longer than the eight-word
	// overlap budget; its oldest words must not be re-fed on later calls.
	require.NotContains(t, prompts[len(prompts)-1], "one two three four")
}

// TestOverlapAgree_SlidingWindowsWordBoundaryAlignment proves
// mergeAgreed's Case 3 (suffix↔prefix overlap) still kicks in when the
// next agreement's first words overlap the prior commit's tail words.
// Under growing-buffer this happens naturally when word-aligned advance
// fails (word timestamps absent or wrong) so consecutive hypotheses
// share head/tail words instead of being clean prefix-extensions.
func TestOverlapAgree_SlidingWindowsWordBoundaryAlignment(t *testing.T) {
	prov := scriptedProvider(-1,
		"the quick brown", "the quick brown",
		"quick brown fox jumps", "quick brown fox jumps",
		"brown fox jumps over", "brown fox jumps over",
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(6, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Equal(t, "the quick brown fox jumps over", joined, "overlap merge produces canonical sequence")
	require.Equal(t, "the quick brown fox jumps over", final)
	for _, word := range []string{"the", "quick", "brown", "fox", "jumps", "over"} {
		count := strings.Count(" "+joined+" ", " "+word+" ")
		require.Equalf(t, 1, count, "word %q should appear once, joined=%q", word, joined)
	}
}

// TestOverlapAgree_TailRegurgitationNoDuplicate proves the
// channel-close tail flush also dedupes against committed. Without
// the fix, a Whisper tail transcribe that regurgitates the prompt
// would commit a duplicated tail.
func TestOverlapAgree_TailRegurgitationNoDuplicate(t *testing.T) {
	prov := scriptedProvider(-1,
		"hello world", "hello world", // commits "hello world"
		"hello world hello world goodbye", // tail call regurgitates prompt
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(2, useWindowMs, 400))

	segs, _, final := segmentsAndFinal(t, got)
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Equal(t, "hello world goodbye", joined, "tail flush dedupes against committed; no duplicated 'hello world'")
	require.Equal(t, "hello world goodbye", final)
}

// TestOverlapAgree_CtxCancelStillEmitsDone proves mid-stream
// cancellation still produces a terminal Done (parity with the other
// strategies).
func TestOverlapAgree_CtxCancelStillEmitsDone(t *testing.T) {
	prov := scriptedProvider(-1, "x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chunks := make(chan sttchain.AudioChunk) // never closes
	got := runStrategy(t, ctx, &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs}, sttchain.StreamStart{}, chunks)
	require.NotEmpty(t, got)
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

func normalizeWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestOverlapAgree_NormalizesCaseAndPunctuationForAgreement proves the
// agreement gate ignores capitalization and trailing-punctuation
// differences across consecutive hypotheses. Whisper jitters both
// across calls (it re-decides sentence boundaries as more context
// arrives); without normalization, "Hello world" vs "hello world."
// would yield zero agreement at position 0 and the algorithm would
// never commit on real audio.
func TestOverlapAgree_NormalizesCaseAndPunctuationForAgreement(t *testing.T) {
	prov := scriptedProvider(-1,
		"Hello world",   // capitalized, no punct
		"hello world.",  // lowercased, trailing period
		"hello, World.", // mixed case + punct again
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(3, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	require.NotEmpty(t, segs, "case/punct differences must NOT block agreement — commit should happen")
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Contains(t, strings.ToLower(joined), "hello", "committed text contains 'hello' (case from first hypothesis preserved)")
	require.Contains(t, strings.ToLower(joined), "world", "committed text contains 'world'")
	require.NotEmpty(t, final)
}

// TestOverlapAgree_BoundedAgreementWindowSurvivesLongJitter proves
// the comparison window is bounded: even when each hypothesis is
// dozens of tokens long with jittery tail words, the
// MaxAgreedTokens-bounded prefix walk still finds agreement on the
// stable head and commits.
//
// Setup: stable head ("hello world how are you") with a jittery,
// growing tail across iterations. Without a bound, the agreement walk
// would attempt to match every position and fail at the jittery
// tail's first divergent token. With the bound (default 30 tokens),
// the walk stops at the head and commits cleanly.
func TestOverlapAgree_BoundedAgreementWindowSurvivesLongJitter(t *testing.T) {
	prov := scriptedProvider(-1,
		"hello world how are you alpha-1",
		"hello world how are you beta-2",
		"hello world how are you gamma-3",
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(3, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	require.NotEmpty(t, segs, "stable head must commit despite jittery tail words")
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Contains(t, joined, "hello world how are you", "stable head reaches the user")
	// Jittery suffix words may or may not appear depending on whether
	// the tail flush captures them — the contract is just that the
	// HEAD commits, which the unbounded variant could not guarantee.
	require.NotEmpty(t, final)
}

// TestOverlapAgree_PostAdvanceCommitsContinue is the regression test
// for the "only first several words" bug.
//
// Setup: scripted provider returns hypotheses with word timestamps so
// the strategy advances committedAudioBytes after the first commit.
// Then the second commit must succeed even though its agreed text
// shares NO words with the (now post-advance) committed prefix —
// because the audio is genuinely new content after the cursor moved.
//
// Before the fix: mergeAgreed's divergence detector rejected the
// post-advance agreement as a "wander", no further Segments emitted,
// only "first several words" reached the user.
func TestOverlapAgree_PostAdvanceCommitsContinue(t *testing.T) {
	const sampleRate = 16000
	prov := scriptedWordsProvider(
		// First two iterations agree on "hello world" — commit + advance
		// cursor to the end of "world" (0.15s).
		scriptedHypothesis{text: "hello world", wordEnds: []float64{0.05, 0.15}},
		scriptedHypothesis{text: "hello world", wordEnds: []float64{0.05, 0.15}},
		// After advance, the next hypotheses are over post-advance
		// audio. Their text does NOT share any prefix with "hello world".
		scriptedHypothesis{text: "how are you", wordEnds: []float64{0.05, 0.10, 0.20}},
		scriptedHypothesis{text: "how are you doing", wordEnds: []float64{0.05, 0.10, 0.20, 0.30}},
		scriptedHypothesis{text: "how are you doing today", wordEnds: []float64{0.05, 0.10, 0.20, 0.30, 0.45}},
	)
	strat := &strategy.OverlapAgree{Provider: prov, Trigger: strategy.TriggerStopwatch, WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2, SampleRate: sampleRate, UseWordTimestampAdvance: true}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(5, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	require.GreaterOrEqual(t, len(segs), 2, "at least two commits: in-stream MUST continue after cursor advance")
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Contains(t, joined, "hello world", "first commit reaches the user")
	require.Contains(t, joined, "how are", "post-advance content reaches the user (was dropped before the fix)")
	require.Equal(t, joined, final, "Done.FinalText matches the joined Segments")
}

// ─────────────────────────────────────────────────────────────────────
// Stall-fallback (overlap_max_stall_rejects)
// ─────────────────────────────────────────────────────────────────────

// TestOverlapAgree_StallFallbackFiresAtExactlyN proves the stall-fallback
// commits the freshest hypothesis tail after exactly MaxStallRejects
// CONSECUTIVE divergence-rejects — not before. Each case commits "alpha
// beta", then feeds divergent hypotheses whose only difference is the
// LAST word; the committed stall text's last word identifies precisely
// which reject triggered the commit.
func TestOverlapAgree_StallFallbackFiresAtExactlyN(t *testing.T) {
	cases := []struct {
		name       string
		n          int
		chunks     int
		texts      []string
		wantJoined string
	}{
		{
			// rejects land on calls #4 ("...q") and #5 ("...r"); N=2 fires
			// on #5 → committed stall text ends in "r".
			name:       "n2",
			n:          2,
			chunks:     5,
			texts:      []string{"alpha beta", "alpha beta", "zeta omega p", "zeta omega q", "zeta omega r"},
			wantJoined: "alpha beta zeta omega r",
		},
		{
			// rejects on #4/#5/#6; N=3 must wait for #6 ("...s"). If the
			// counter mis-fired at N=2 the text would end in "r".
			name:       "n3",
			n:          3,
			chunks:     6,
			texts:      []string{"alpha beta", "alpha beta", "zeta omega p", "zeta omega q", "zeta omega r", "zeta omega s"},
			wantJoined: "alpha beta zeta omega s",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			prov := scriptedProvider(-1, tc.texts...)
			strat := &strategy.OverlapAgree{
				Provider: prov, Trigger: strategy.TriggerStopwatch,
				WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2,
				MaxStallRejects: tc.n,
			}
			got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(tc.chunks, useWindowMs, 0))

			segs, _, final := segmentsAndFinal(t, got)
			joined := strings.TrimSpace(strings.Join(segs, ""))
			require.Equal(t, tc.wantJoined, joined,
				"stall-fallback must commit the freshest tail at exactly the Nth reject")
			require.Equal(t, tc.wantJoined, final, "Done.FinalText carries the stall-committed text")
		})
	}
}

// TestOverlapAgree_StallFallbackDisabledPreservesLegacy proves
// MaxStallRejects=0 disables the fallback entirely: the divergent
// in-stream hypotheses are NEVER committed mid-stream (only surfaced as
// Partials), exactly as before the lever existed. The audio ends with the
// model recovering to the committed prefix so the channel-close tail
// flush produces nothing — isolating the in-stream behavior.
func TestOverlapAgree_StallFallbackDisabledPreservesLegacy(t *testing.T) {
	prov := scriptedProvider(-1,
		"alpha beta", "alpha beta", // commit "alpha beta"
		"zeta omega p", "zeta omega q", "zeta omega r", // divergent: would trip the fallback if enabled
		"alpha beta", "alpha beta", // model recovers → clean tail flush
	)
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerStopwatch,
		WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2,
		MaxStallRejects: 0, // disabled
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(7, useWindowMs, 0))

	segs, partials, _ := segmentsAndFinal(t, got)
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Equal(t, "alpha beta", joined, "with the fallback disabled, divergent runs never commit mid-stream")
	for _, s := range segs {
		require.NotContains(t, s, "zeta", "no divergent content may commit when MaxStallRejects=0")
	}
	require.Contains(t, partials, "zeta omega r", "the divergent runs are still surfaced as live Partials")
}

// TestOverlapAgree_StallFallbackResetsAfterCommit proves the
// consecutive-reject counter resets on any forward commit. A reject (1),
// then a real agreement commit (reset to 0), then two fresh rejects must
// be required to fire — so the committed stall text reflects the SECOND
// batch's freshest tail, not an early mis-fire carried over from the
// first reject.
func TestOverlapAgree_StallFallbackResetsAfterCommit(t *testing.T) {
	prov := scriptedProvider(-1,
		"alpha", "alpha", // commit "alpha"
		"beta gamma p", "beta gamma q", // reject #1 (counter=1)
		"alpha epsilon", "alpha epsilon", // real commit "epsilon" → counter resets to 0
		"zeta eta p", "zeta eta q", "zeta eta r", // fresh rejects: #1, #2 → fire on "r"
	)
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerStopwatch,
		WindowMs: useWindowMs, AdvanceMs: useWindowMs, CommitRuns: 2,
		MaxStallRejects: 2,
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(9, useWindowMs, 0))

	segs, _, final := segmentsAndFinal(t, got)
	joined := strings.TrimSpace(strings.Join(segs, ""))
	require.Equal(t, "alpha epsilon zeta eta r", joined,
		"counter reset on commit: the fallback fires on the 2nd FRESH reject (\"r\"), not carried over from the pre-commit reject")
	require.NotContains(t, joined, "zeta eta q", "the first fresh reject (\"q\") must not be the commit point")
	require.Equal(t, joined, final)
}

// ─────────────────────────────────────────────────────────────────────
// Phase C: VAD-anchored triggering tests
// ─────────────────────────────────────────────────────────────────────

// vadChunks returns a channel that sends one chunk per
// voiced-then-silent segment in `segments`. Each segment is described
// by (voicedMs, silenceMs). The resulting audio drives the VAD
// trigger: a settle attempt fires at the end of each silence window.
func vadChunks(sampleRate int, segments ...[2]int) <-chan sttchain.AudioChunk {
	ch := make(chan sttchain.AudioChunk, len(segments)+1)
	for _, seg := range segments {
		ch <- sttchain.AudioChunk{Audio: voicedThenSilent(sampleRate, seg[0], seg[1])}
	}
	close(ch)
	return ch
}

// TestOverlapAgree_VAD_TriggersOnSilenceBoundary proves the VAD
// trigger calls Whisper exactly once per silence boundary detected in
// the audio stream. Two silence-bounded utterances → two scripted
// transcribe calls. Stopwatch jitter would call far more or far
// fewer times depending on chunk timing.
func TestOverlapAgree_VAD_TriggersOnSilenceBoundary(t *testing.T) {
	const sampleRate = 16000
	var calls int
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		calls++
		return &sttchain.Result{Text: "settle " + intToString(calls), Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerVAD,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, SilenceMs: 200, SilenceRMS: 250, FrameMs: 20,
	}
	// Two utterances: each 400ms voiced + 240ms silence (silence ≥ SilenceMs).
	chunks := vadChunks(sampleRate, [2]int{400, 240}, [2]int{400, 240})
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunks)
	_, _, _ = segmentsAndFinal(t, got)

	// 2 silence-triggered iterations + 1 tail flush call on close = 3 calls.
	require.GreaterOrEqual(t, calls, 2, "each silence boundary must trigger one Whisper call")
	require.LessOrEqual(t, calls, 3, "trigger must not fire mid-utterance (no stopwatch spam)")
}

// TestOverlapAgree_VAD_NoTriggerWithoutSilence proves the VAD trigger
// stays quiet during continuous voiced audio. The MaxWindowMs safety
// net is what eventually fires if speech goes too long without
// pausing — without that, latency would grow unbounded; with it,
// per-call latency is capped.
func TestOverlapAgree_VAD_NoTriggerWithoutSilence(t *testing.T) {
	const sampleRate = 16000
	var calls int
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		calls++
		return &sttchain.Result{Text: "x"}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerVAD,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, SilenceMs: 200, SilenceRMS: 250, FrameMs: 20,
		MaxWindowMs: 10000, // generous ceiling — must NOT trigger in this test
	}
	// 800ms of continuous voiced audio, no silence anywhere.
	chunks := vadChunks(sampleRate, [2]int{800, 0})
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunks)
	_, _, _ = segmentsAndFinal(t, got)

	// At most one call — the tail flush on channel close. Definitely
	// not the stopwatch-style 8 calls / 800ms / 100ms advance.
	require.LessOrEqual(t, calls, 1, "continuous voiced audio must NOT trigger mid-stream settle attempts")
}

// TestOverlapAgree_VAD_MaxWindowForceCommitsAudio proves the
// MaxWindowMs safety net force-COMMITS (does not drop) when voiced
// audio piles up without a silence boundary. The "never lose audio"
// guarantee holds even during continuous speech: the user always
// gets a Segment for what they said.
func TestOverlapAgree_VAD_MaxWindowForceCommitsAudio(t *testing.T) {
	const sampleRate = 16000
	var calls int
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		calls++
		return &sttchain.Result{Text: "voiced text " + intToString(calls), Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerVAD,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, SilenceMs: 200, SilenceRMS: 250, FrameMs: 20,
		MaxWindowMs: 500, // tight ceiling; 800ms of voice will trip it
	}
	chunks := vadChunks(sampleRate, [2]int{800, 0})
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunks)

	segs, _, final := segmentsAndFinal(t, got)
	require.NotEmpty(t, segs, "MaxWindowMs ceiling must emit a force-commit Segment, not drop the audio")
	require.NotEmpty(t, final, "Done.FinalText carries the force-committed transcript")
	require.GreaterOrEqual(t, calls, 1, "force-commit must call Whisper to obtain the transcript")
	// No "audio dropped" Error events on the success path.
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError && ev.Error != nil {
			require.NotContains(t, ev.Error.Error(), "audio dropped",
				"successful force-commit must not emit a drop-audio Error")
		}
	}
}

// TestOverlapAgree_VAD_EmptyForceResultBacksOff proves an empty provider
// result retains the pending audio without issuing one force request per
// incoming frame. The retry point advances by one settle step and the
// terminal event remains explicit.
func TestOverlapAgree_VAD_EmptyForceResultBacksOff(t *testing.T) {
	const sampleRate = 16000
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "", Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerVAD,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, SilenceMs: 200, SilenceRMS: 250, FrameMs: 20,
		MaxWindowMs: 300,
	}
	chunks := make(chan sttchain.AudioChunk, 60)
	for i := 0; i < 60; i++ {
		chunks <- sttchain.AudioChunk{Audio: voicedFrame(sampleRate, 20)}
	}
	close(chunks)
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunks)

	require.LessOrEqual(t, prov.Calls, 12, "empty force results must not trigger one provider call per 20ms frame")
	var sawError bool
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError {
			sawError = true
		}
	}
	require.True(t, sawError, "the uncommitted speech failure must remain visible")
	require.Equal(t, sttchain.StreamEventDone, got[len(got)-1].Kind)
}

// TestOverlapAgree_ForceCommitRetainsPhysicalTail proves a force commit does
// not claim coverage through the exact end of a batch transcription. Whisper
// can omit the final few words at a hard window boundary; the next request
// must re-read a small physical tail so that omission is recoverable.
func TestOverlapAgree_ForceCommitRetainsPhysicalTail(t *testing.T) {
	const sampleRate = 16000
	var requests [][]byte
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(_ context.Context, req sttchain.Request) (*sttchain.Result, error) {
		requests = append(requests, append([]byte(nil), req.Audio...))
		return &sttchain.Result{Text: "covered words", Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerStopwatch,
		WindowMs: 100, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, MaxWindowMs: 500,
	}
	// Each 800ms chunk force-commits. The second request must include the
	// retained tail from the first window, so it is larger than a fresh 800ms
	// request rather than silently starting at the previous window's end.
	runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(2, 800, 0))
	require.GreaterOrEqual(t, len(requests), 2)
	require.Greater(t, len(requests[1]), len(requests[0]),
		"the second force window must retain physical overlap from the first")
}

// TestOverlapAgree_VAD_CommitsFirstPostForceBoundary proves a successful
// force-commit can hand the cursor to the next clean VAD boundary. The
// production path must not spend a second LocalAgreement call building a
// two-hypothesis run over audio that already has a clean boundary: that call
// cannot commit before the next force window and makes the real-time lane
// slower than the audio it is consuming.
func TestOverlapAgree_VAD_CommitsFirstPostForceBoundary(t *testing.T) {
	const sampleRate = 16000
	var calls int
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		calls++
		return &sttchain.Result{Text: "boundary text " + intToString(calls), Tier: sttchain.TierLocal}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerVAD,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, SilenceMs: 200, SilenceRMS: 250, FrameMs: 20,
		MaxWindowMs: 500,
	}
	// The first utterance trips MaxWindowMs while voiced. The second ends
	// with a clean boundary immediately after that force commit.
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{},
		vadChunks(sampleRate, [2]int{800, 0}, [2]int{100, 240}))

	segs, _, final := segmentsAndFinal(t, got)
	require.GreaterOrEqual(t, len(segs), 2, "the clean post-force boundary must commit immediately")
	require.Contains(t, strings.Join(segs, " "), "boundary text 2")
	require.Contains(t, final, "boundary text 2")
	var partials int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventPartial {
			partials++
		}
	}
	require.Zero(t, partials, "a clean post-force boundary must not spend a call on an uncommittable partial")
	require.GreaterOrEqual(t, calls, 2)
}

// TestOverlapAgree_RetainsAudioAfterProviderFailure proves a transient
// provider error does not advance committedAudioBytes. A subsequent settle
// attempt can therefore transcribe the same audio and commit the recovery.
func TestOverlapAgree_RetainsAudioAfterProviderFailure(t *testing.T) {
	var calls int
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("temporary provider outage")
		}
		return &sttchain.Result{Text: "recovered speech", Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerStopwatch,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		MaxWindowMs: 2000,
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksOfWindows(5, 200, 0))

	segs, _, final := segmentsAndFinal(t, got)
	var errorsSeen int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError {
			errorsSeen++
		}
	}
	require.GreaterOrEqual(t, calls, 2, "a later settle attempt must retry retained audio")
	require.GreaterOrEqual(t, errorsSeen, 1, "the provider failure must be observable")
	require.Contains(t, strings.Join(segs, " "), "recovered speech")
	require.Contains(t, final, "recovered speech")
}

// TestOverlapAgree_VAD_TailFlushOnChannelClose proves the channel-close
// tail flush still runs in VAD mode — unsettled audio at session end
// reaches the user regardless of whether the trigger was VAD or
// stopwatch.
func TestOverlapAgree_VAD_TailFlushOnChannelClose(t *testing.T) {
	const sampleRate = 16000
	prov := scriptedProvider(-1, "first", "second", "third")
	strat := &strategy.OverlapAgree{
		Provider: prov, Trigger: strategy.TriggerVAD,
		WindowMs: 200, AdvanceMs: 100, CommitRuns: 2,
		SampleRate: sampleRate, SilenceMs: 200, SilenceRMS: 250, FrameMs: 20,
	}
	// One short utterance + 200ms silence (triggers one VAD settle),
	// then 400ms voiced with no closing silence (must be tail-flushed).
	chunks := vadChunks(sampleRate, [2]int{400, 240}, [2]int{400, 0})
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunks)

	segs, _, final := segmentsAndFinal(t, got)
	require.NotEmpty(t, segs, "VAD-mode tail flush must emit something for the trailing voiced audio")
	require.NotEmpty(t, final, "Done.FinalText must carry the committed transcript")
}

// intToString avoids importing strconv just for digit formatting.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
