package eval

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/segmenter/testaudio"
)

// scriptedSession emits a caller-controlled ordered list of committed segments
// followed by a terminal Done carrying finalText, so a test can drive the exact
// commit timeline + final hypothesis the safety gates operate on through the
// real Replay -> EvalClip -> RunReport path.
func scriptedSession(meter *MeteredProvider, segments []string, finalText string) Session {
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		var audio []byte
		for ch := range chunks {
			audio = append(audio, ch.Audio...)
		}
		_, _ = meter.Transcribe(ctx, sttchain.Request{Audio: audio})
		for _, seg := range segments {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: seg}}
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: finalText}}
		close(events)
		return nil
	}
}

func gateSpec(label string, segments []string, finalText string) StrategySpec {
	return StrategySpec{
		Kind:  sttchain.StrategyVADSegment,
		Label: label,
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("x", time.Millisecond), float64(clip.bytesPerSecond()))
			return scriptedSession(meter, segments, finalText), meter
		},
	}
}

// TestRunReport_SafetyGatesFailOnInjectedFaults proves the phase-7 hard gates
// actually fail through the real harness on a committed-text retraction and on a
// threshold-sized contiguous dropped span, and that a clean run passes — the
// end-to-end wiring the per-detector unit tests do not exercise.
func TestRunReport_SafetyGatesFailOnInjectedFaults(t *testing.T) {
	opts := EvalOptions{ChunkMs: 100, QualityPass: true, DroppedSpanThresholdWords: 4}
	byLabel := func(rep EvalReport) map[string]StrategyReport {
		out := map[string]StrategyReport{}
		for _, row := range rep.PerStrategy {
			out[row.Label] = row
		}
		return out
	}

	// Retraction fixture uses a short reference so the single dropped word stays
	// well under the dropped-span threshold — isolating the retraction gate.
	retractClip := []Clip{{ID: "r1", PCM: testaudio.SilenceSamplesAtRate(16000, 1000), SampleRate: 16000, Reference: "alpha bravo charlie"}}
	// Commit "...charlie" then finalize dropping it: a previously committed token
	// is removed, which the retraction gate forbids absolutely.
	retract := gateSpec("retract", []string{"alpha bravo charlie"}, "alpha bravo")
	rRep := byLabel(RunReport(context.Background(), retractClip, []StrategySpec{retract}, opts))
	r := rRep["retract"].Safety
	require.False(t, r.Passed, "retraction must fail the safety gate")
	require.False(t, r.RetractionFree, "retraction gate should flag the removed committed token")
	require.True(t, r.DroppedSpanFree, "a single dropped word must not trip the dropped-span gate")
	require.NotEmpty(t, r.RetractionEvents)

	// Dropped-span + clean share a reference long enough to host a 4-word drop.
	reference := "alpha bravo charlie delta echo foxtrot golf"
	longClip := []Clip{{ID: "d1", PCM: testaudio.SilenceSamplesAtRate(16000, 1000), SampleRate: 16000, Reference: reference}}
	dropped := gateSpec("dropped", []string{"alpha bravo golf"}, "alpha bravo golf") // omits 4 contiguous mid words
	clean := gateSpec("clean", []string{reference}, reference)
	dRep := byLabel(RunReport(context.Background(), longClip, []StrategySpec{dropped, clean}, opts))

	d := dRep["dropped"].Safety
	require.False(t, d.Passed, "a 4-word contiguous drop must fail the safety gate")
	require.False(t, d.DroppedSpanFree, "dropped-span gate should flag the contiguous omission")
	require.True(t, d.RetractionFree, "the dropped-span fixture commits no retraction")
	require.GreaterOrEqual(t, d.MaxDroppedSpanWords, 4)

	c := dRep["clean"].Safety
	require.True(t, c.Passed, "a clean run passes both gates")
	require.True(t, c.RetractionFree)
	require.True(t, c.DroppedSpanFree)
}
