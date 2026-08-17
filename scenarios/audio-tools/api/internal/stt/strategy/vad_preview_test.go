package strategy_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

// VADSegment is the default strategy on every host without a CUDA GPU, because
// kyutai-stt is the only streaming provider and it requires one. Before the
// preview lane existed this strategy emitted text only at silence boundaries,
// so an operator speaking continuously saw nothing at all until they paused —
// which is every macOS host, every Windows host, and every GPU-less Linux host.
//
// These tests pin the two halves of that fix: previews must appear during
// continuous speech, and their cost must not grow with how long the operator
// has been talking.

// recordingProvider captures every request the strategy issues. Preview calls
// run on their own goroutines, so the recorder is mutex-guarded.
type recordingProvider struct {
	mu       sync.Mutex
	audioLen []int
	fail     bool
}

func (r *recordingProvider) record(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.audioLen = append(r.audioLen, n)
}

func (r *recordingProvider) lengths() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.audioLen...)
}

func newRecordingProvider(t *testing.T, rec *recordingProvider, text string) sttchain.Provider {
	t.Helper()
	prov := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	prov.TranscribeFn = func(_ context.Context, req sttchain.Request) (*sttchain.Result, error) {
		rec.record(len(req.Audio))
		if rec.fail {
			return nil, errors.New("provider unavailable")
		}
		return &sttchain.Result{Text: text, Tier: sttchain.TierLocal, ProviderID: "fake", ModelID: "fake"}, nil
	}
	return prov
}

// continuousVoiced sends n chunks of chunkMs voiced PCM with no silence, so no
// segment boundary can ever fire during the stream.
func continuousVoiced(n, chunkMs int) <-chan sttchain.AudioChunk {
	ch := make(chan sttchain.AudioChunk, n)
	for i := 0; i < n; i++ {
		ch <- sttchain.AudioChunk{Audio: testaudio.SineSamples(440, chunkMs)}
	}
	close(ch)
	return ch
}

func partialTexts(events []sttchain.StreamEvent) []string {
	var out []string
	for _, ev := range events {
		if ev.Kind == sttchain.StreamEventPartial && ev.Partial != nil {
			out = append(out, ev.Partial.Text)
		}
	}
	return out
}

// TestVADSegmenter_EmitsPartialsDuringContinuousSpeech is the regression this
// lane exists for: text must reach the operator before they stop talking.
func TestVADSegmenter_EmitsPartialsDuringContinuousSpeech(t *testing.T) {
	rec := &recordingProvider{}
	strat := &strategy.VADSegmenter{
		Provider:          newRecordingProvider(t, rec, "live words"),
		SilenceMs:         5000,
		PreRollMs:         0,
		TrailingPadMs:     0,
		PreviewIntervalMs: 100,
		PreviewWindowMs:   500,
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(20, 100))

	require.NotEmpty(t, partialTexts(got), "continuous speech must produce live text before any silence boundary")
}

// TestVADSegmenter_NoPartialsWhenPreviewDisabled keeps the lane a lever rather
// than a mandate: an operator trading live text for provider load gets exactly
// the old behaviour back.
func TestVADSegmenter_NoPartialsWhenPreviewDisabled(t *testing.T) {
	rec := &recordingProvider{}
	strat := &strategy.VADSegmenter{
		Provider:          newRecordingProvider(t, rec, "live words"),
		SilenceMs:         5000,
		PreviewIntervalMs: 0,
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(20, 100))

	require.Empty(t, partialTexts(got), "previews must be off when the interval lever is zero")
}

// TestVADSegmenter_PreviewCostStaysBoundedAsSegmentGrows is the anti-quadratic
// gate. During continuous speech no segment commits, so the in-flight buffer
// grows for the whole session; previewing all of it would make per-preview cost
// proportional to elapsed time — the same shape as the retention defects this
// pipeline was repaired for. Every preview must read a bounded trailing window.
func TestVADSegmenter_PreviewCostStaysBoundedAsSegmentGrows(t *testing.T) {
	// One chunk carrying a segment far larger than the preview window. A single
	// chunk keeps the run deterministic: previews are single-flight, so with
	// many pre-buffered chunks how many previews fire depends on how fast the
	// loop drains them, which is scheduler-dependent. Exactly one preview and
	// one end-of-stream flush is enough to prove the clamp.
	const (
		speechMs        = 6000
		previewWindowMs = 500
	)
	rec := &recordingProvider{}
	strat := &strategy.VADSegmenter{
		Provider:          newRecordingProvider(t, rec, "live words"),
		SilenceMs:         60000,
		PreRollMs:         0,
		TrailingPadMs:     0,
		PreviewIntervalMs: 100,
		PreviewWindowMs:   previewWindowMs,
	}
	runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(1, speechMs))

	lengths := rec.lengths()
	require.NotEmpty(t, lengths, "the strategy must have called the provider")
	windowBytes := testaudio.PCMByteCount(testaudio.SampleRateHz, previewWindowMs)
	totalBytes := testaudio.PCMByteCount(testaudio.SampleRateHz, speechMs)

	// Previews run asynchronously, so a preview issued before the stream closed
	// can be recorded after the end-of-stream flush. Position therefore proves
	// nothing; size does. Exactly one call — the flush — may exceed the window,
	// because it legitimately transcribes the whole retained segment.
	require.Greater(t, totalBytes, windowBytes, "the fixture must outgrow the preview window for this test to mean anything")

	oversize, atWindow := 0, 0
	for _, n := range lengths {
		if n > windowBytes {
			oversize++
		}
		if n == windowBytes {
			atWindow++
		}
	}
	require.Equalf(t, 1, oversize,
		"only the end-of-stream flush may read more than the %d-byte window; got %d such calls across %v — a preview is reading the whole segment, so its cost grows with session length",
		windowBytes, oversize, lengths)
	// Without this the bound above would pass vacuously on a run where the
	// clamp never had to do anything.
	require.Positivef(t, atWindow,
		"the clamp must actually engage: the segment grew to %d bytes, so some preview must have been cut to exactly %d bytes; got %v",
		totalBytes, windowBytes, lengths)
}

// TestVADSegmenter_PreviewFailureIsInvisible keeps a best-effort lane from
// turning a healthy session into a visibly broken one. A preview that fails
// carries no information the operator can act on — the audio is still retained
// and the segment boundary will transcribe it properly.
func TestVADSegmenter_PreviewFailureIsInvisible(t *testing.T) {
	rec := &recordingProvider{fail: true}
	strat := &strategy.VADSegmenter{
		Provider:          newRecordingProvider(t, rec, ""),
		SilenceMs:         60000,
		PreviewIntervalMs: 100,
		PreviewWindowMs:   500,
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(10, 100))

	require.Empty(t, partialTexts(got), "a failed preview must not emit text")
	// The end-of-stream flush legitimately reports its own failure; previews
	// must not add to that count.
	var errs int
	for _, ev := range got {
		if ev.Kind == sttchain.StreamEventError {
			errs++
		}
	}
	require.LessOrEqual(t, errs, 1, "preview failures must not surface as session errors")
}

// TestVADSegmenter_NoPartialArrivesAfterDone protects the consumer's state
// machine. Interim text that lands after the turn ended can never be settled,
// so it would sit in the composer forever.
func TestVADSegmenter_NoPartialArrivesAfterDone(t *testing.T) {
	rec := &recordingProvider{}
	strat := &strategy.VADSegmenter{
		Provider:          newRecordingProvider(t, rec, "live words"),
		SilenceMs:         60000,
		PreviewIntervalMs: 100,
		PreviewWindowMs:   500,
	}
	got := runStrategy(t, context.Background(), strat, sttchain.StreamStart{}, continuousVoiced(20, 100))

	require.NotEmpty(t, got)
	doneAt := -1
	for i, ev := range got {
		if ev.Kind == sttchain.StreamEventDone {
			doneAt = i
		}
	}
	require.GreaterOrEqual(t, doneAt, 0, "the session must terminate with a done event")
	require.Equal(t, len(got)-1, doneAt, "no event may follow done")
}
