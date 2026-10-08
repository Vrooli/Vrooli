package strategy_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

// recorderProvider captures every Request the segmenter dispatches so
// boundary-accuracy tests can assert on the audio bytes and prompts
// sent to Whisper.
type recorderProvider struct {
	mu       sync.Mutex
	requests []sttchain.Request
	replies  []string
	idx      int
}

func (r *recorderProvider) Transcribe(_ context.Context, req sttchain.Request) (*sttchain.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	text := "seg"
	if r.idx < len(r.replies) {
		text = r.replies[r.idx]
	}
	r.idx++
	return &sttchain.Result{Text: text, Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
}
func (r *recorderProvider) Type() sttchain.ProviderTier      { return sttchain.TierLocal }
func (r *recorderProvider) IsAvailable(context.Context) bool { return true }
func (r *recorderProvider) Model() string                    { return "fake" }
func (r *recorderProvider) Traits() sttchain.ProviderTraits {
	return sttchain.ProviderTraits{Batch: true}
}

func (r *recorderProvider) TranscribeStreaming(context.Context, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	return nil, nil
}

var _ sttchain.Provider = (*recorderProvider)(nil)

const sampleBytes = 2

// TestVADSegmenter_PreRollOverlapsBetweenSegments asserts that
// segment N+1's audio begins PreRollMs before the start of voiced
// content — i.e., the new segment overlaps the trailing PCM of the
// previous one.
func TestVADSegmenter_PreRollOverlapsBetweenSegments(t *testing.T) {
	prov := &recorderProvider{replies: []string{"hello", "world"}}
	strat := &strategy.VADSegmenter{
		Provider:      prov,
		SilenceMs:     200,
		PreRollMs:     300,
		TrailingPadMs: 0,
		SilenceRMS:    100,
	}
	// 100ms tone | 400ms silence | 100ms tone | trailing 400ms silence.
	audio := testaudio.SineSamples(440, 100)
	audio = append(audio, testaudio.SilenceSamples(400)...)
	audio = append(audio, testaudio.SineSamples(660, 100)...)
	audio = append(audio, testaudio.SilenceSamples(400)...)

	_ = runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	require.GreaterOrEqual(t, len(prov.requests), 2, "expected ≥2 segments, got %d", len(prov.requests))
	// Segment 2's audio length should exceed the voiced 100ms (=3200 bytes)
	// by roughly PreRollMs * 32 bytes/ms = 9600 bytes. Allow slack for
	// frame quantisation and trailing-pad zero-fill.
	seg2 := prov.requests[1].Audio
	require.Greater(t, len(seg2), 100*16000/1000*sampleBytes,
		"segment 2 should be larger than the bare 100ms voiced window when PreRollMs=300")
}

// TestVADSegmenter_TrailingPadKeepsRealSilence asserts that segment N's
// emitted audio extends past the last voiced frame by TrailingPadMs of
// silence — Whisper deals with natural decay better than a flush cut.
func TestVADSegmenter_TrailingPadKeepsRealSilence(t *testing.T) {
	prov := &recorderProvider{replies: []string{"x"}}
	strat := &strategy.VADSegmenter{
		Provider:      prov,
		SilenceMs:     200,
		TrailingPadMs: 200,
		PreRollMs:     0,
		SilenceRMS:    100,
	}
	audio := testaudio.SineSamples(440, 100)
	audio = append(audio, testaudio.SilenceSamples(600)...)
	_ = runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	require.Len(t, prov.requests, 1)
	// Voiced is ~100ms = 3200 bytes; with TrailingPadMs=200 the segment
	// should reach at least ~300ms = 9600 bytes (allow small slack).
	require.GreaterOrEqual(t, len(prov.requests[0].Audio), 8000,
		"trailing pad should extend the segment past the voiced edge; got %d bytes", len(prov.requests[0].Audio))
}

// TestVADSegmenter_InitialPromptRolloverAndDedup asserts that segment 2
// receives the previous segment's last-K words as initial_prompt AND
// that the emitted SegmentEvent.Text for segment 2 is deduped against
// what was already committed.
func TestVADSegmenter_InitialPromptRolloverAndDedup(t *testing.T) {
	// Provider returns overlapping text: "hello world" then "world goodbye".
	// After dedup, segment 2 should emit only "goodbye".
	prov := &recorderProvider{replies: []string{"hello world", "world goodbye"}}
	strat := &strategy.VADSegmenter{
		Provider:           prov,
		SilenceMs:          200,
		PreRollMs:          0,
		TrailingPadMs:      0,
		InitialPromptWords: 5,
		SilenceRMS:         100,
	}
	audio := testaudio.SineSamples(440, 100)
	audio = append(audio, testaudio.SilenceSamples(400)...)
	audio = append(audio, testaudio.SineSamples(660, 100)...)
	audio = append(audio, testaudio.SilenceSamples(400)...)

	events := runStrategy(t, context.Background(), strat, sttchain.StreamStart{InitialPrompt: ""}, chunksFrom(audio))

	require.GreaterOrEqual(t, len(prov.requests), 2)
	// Segment 2's initial_prompt should include the last words of segment 1.
	require.Contains(t, prov.requests[1].InitialPrompt, "hello world",
		"segment 2 initial_prompt should carry segment 1's tail; got %q", prov.requests[1].InitialPrompt)

	// Walk emitted SegmentEvents and concatenate; full result should be
	// "hello world goodbye" — "world" must appear once, not twice.
	var emitted []string
	for _, ev := range events {
		if ev.Kind == sttchain.StreamEventSegment && ev.Segment != nil {
			emitted = append(emitted, ev.Segment.Text)
		}
	}
	require.GreaterOrEqual(t, len(emitted), 2)
	joined := strings.Join(emitted, " ")
	require.Equal(t, 1, strings.Count(strings.ToLower(joined), "world"),
		"dedup should remove the duplicate 'world'; got %q", joined)
}

func TestVADSegmenter_DoesNotTreatARepeatedPhraseAsPreRoll(t *testing.T) {
	phrase := "alpha beta gamma delta epsilon zeta"
	prov := &recorderProvider{replies: []string{phrase, phrase + " new"}}
	strat := &strategy.VADSegmenter{
		Provider:           prov,
		SilenceMs:          200,
		PreRollMs:          0,
		TrailingPadMs:      0,
		InitialPromptWords: 0,
		SilenceRMS:         100,
	}
	audio := testaudio.SineSamples(440, 100)
	audio = append(audio, testaudio.SilenceSamples(400)...)
	audio = append(audio, testaudio.SineSamples(660, 100)...)
	audio = append(audio, testaudio.SilenceSamples(400)...)

	events := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))
	var emitted []string
	for _, ev := range events {
		if ev.Kind == sttchain.StreamEventSegment && ev.Segment != nil {
			emitted = append(emitted, ev.Segment.Text)
		}
	}
	joined := strings.Join(emitted, " ")
	require.Equal(t, 2, strings.Count(strings.ToLower(joined), "alpha"),
		"a repeated phrase is real speech when no audio pre-roll exists: %q", joined)
	require.Contains(t, joined, "new")
}

// TestVADSegmenter_InitialPromptHonorsOperatorPrefix asserts that the
// operator-supplied StreamStart.InitialPrompt stays in front of the
// rolling tail.
func TestVADSegmenter_InitialPromptHonorsOperatorPrefix(t *testing.T) {
	prov := &recorderProvider{replies: []string{"alpha beta"}}
	strat := &strategy.VADSegmenter{
		Provider:           prov,
		SilenceMs:          200,
		InitialPromptWords: 5,
		SilenceRMS:         100,
	}
	audio := testaudio.SineSamples(440, 100)
	audio = append(audio, testaudio.SilenceSamples(400)...)
	_ = runStrategy(t, context.Background(), strat, sttchain.StreamStart{InitialPrompt: "domain hint"}, chunksFrom(audio))

	require.Len(t, prov.requests, 1)
	require.Equal(t, "domain hint", prov.requests[0].InitialPrompt,
		"first segment should see only operator prefix; got %q", prov.requests[0].InitialPrompt)
}

// TestVADSegmenter_DisabledLevers reproduces the pre-fix baseline.
// With all three new tunables at zero, behavior should match the
// pre-existing hard-cut + no-context strategy: segment 2 carries no
// audio overlap, no trailing pad, no initial_prompt rollover.
func TestVADSegmenter_DisabledLevers(t *testing.T) {
	prov := &recorderProvider{replies: []string{"a", "b"}}
	strat := &strategy.VADSegmenter{
		Provider:           prov,
		SilenceMs:          200,
		PreRollMs:          0,
		TrailingPadMs:      0,
		InitialPromptWords: 0,
		SilenceRMS:         100,
	}
	_ = sttmocks.NewFakeProvider // import keep
	audio := testaudio.SineSamples(440, 100)
	audio = append(audio, testaudio.SilenceSamples(400)...)
	audio = append(audio, testaudio.SineSamples(660, 100)...)
	audio = append(audio, testaudio.SilenceSamples(400)...)
	_ = runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, chunksFrom(audio))

	require.GreaterOrEqual(t, len(prov.requests), 2)
	require.Equal(t, "", prov.requests[1].InitialPrompt,
		"zero initial_prompt_words should suppress rollover")
}
