package segmenter

import (
	"context"
	"testing"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt"
)

// runBuffered drives a single-chunk buffered-fallback session (Mode=off)
// with the given canned provider result and StreamConfig, returning the
// drained events.
func runBuffered(t *testing.T, res *sttchain.Result, cfg stt.StreamConfig) []sttchain.StreamEvent {
	t.Helper()
	exec := &sttmocks.FakeBatchExecutor{Result: res}
	chain := sttchain.NewChain(sttchain.Options{})
	seg := New(Deps{Chain: chain, Selector: stt.NewSelector(exec)})

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: []byte{0x01}}
	close(chunks)
	events := make(chan sttchain.StreamEvent, 8)

	cfg.Mode = stt.ModeOff
	_ = seg.Run(context.Background(), sttchain.StreamStart{}, cfg, chunks, events)
	return drain(events)
}

func segmentsOf(events []sttchain.StreamEvent) []*sttchain.SegmentEvent {
	var segs []*sttchain.SegmentEvent
	for _, ev := range events {
		if ev.Kind == sttchain.StreamEventSegment {
			segs = append(segs, ev.Segment)
		}
	}
	return segs
}

func doneOf(t *testing.T, events []sttchain.StreamEvent) *sttchain.DoneEvent {
	t.Helper()
	last := events[len(events)-1]
	if last.Kind != sttchain.StreamEventDone || last.Done == nil {
		t.Fatalf("last event must be Done, got %+v", last)
	}
	return last.Done
}

// TestSegmenter_EgressDropsHallucination proves the text-domain stage drops
// a "thank you for watching" segment end-to-end and rebuilds FinalText.
func TestSegmenter_EgressDropsHallucination(t *testing.T) {
	out := runBuffered(t,
		&sttchain.Result{Text: "thank you for watching"},
		stt.StreamConfig{HallucinationFilterEnabled: true},
	)
	if segs := segmentsOf(out); len(segs) != 0 {
		t.Fatalf("hallucinated segment should be dropped, got %d segments", len(segs))
	}
	if got := doneOf(t, out).FinalText; got != "" {
		t.Fatalf("FinalText should be rebuilt to empty after drop, got %q", got)
	}
}

// TestSegmenter_EgressDropsLowConfidence proves the signal-domain stage
// drops a segment whose confidence signals cross both thresholds.
func TestSegmenter_EgressDropsLowConfidence(t *testing.T) {
	out := runBuffered(t,
		&sttchain.Result{
			Text:       "phantom words",
			Confidence: &sttchain.Confidence{NoSpeechProb: 0.9, AvgLogProb: -2.0},
		},
		stt.StreamConfig{NoSpeechThreshold: 0.6, LogProbThreshold: -1.0},
	)
	if segs := segmentsOf(out); len(segs) != 0 {
		t.Fatalf("low-confidence segment should be dropped, got %d segments", len(segs))
	}
	if got := doneOf(t, out).FinalText; got != "" {
		t.Fatalf("FinalText should be rebuilt to empty after drop, got %q", got)
	}
}

// TestSegmenter_EgressEmitsConfidentSpeech proves a confident, non-hallucination
// segment survives the gate, and that the gate-only fields are stripped
// before reaching the consumer.
func TestSegmenter_EgressEmitsConfidentSpeech(t *testing.T) {
	out := runBuffered(t,
		&sttchain.Result{
			Text:       "the quick brown fox",
			Confidence: &sttchain.Confidence{NoSpeechProb: 0.1, AvgLogProb: -0.3},
		},
		stt.StreamConfig{HallucinationFilterEnabled: true, NoSpeechThreshold: 0.6, LogProbThreshold: -1.0},
	)
	segs := segmentsOf(out)
	if len(segs) != 1 {
		t.Fatalf("confident speech should survive, got %d segments", len(segs))
	}
	if segs[0].Text != "the quick brown fox" {
		t.Fatalf("unexpected text: %q", segs[0].Text)
	}
	if segs[0].Confidence != nil || segs[0].Audio != nil {
		t.Fatalf("gate-only fields must be stripped before the wire: %+v", segs[0])
	}
	if got := doneOf(t, out).FinalText; got != "the quick brown fox" {
		t.Fatalf("unexpected FinalText %q", got)
	}
}
