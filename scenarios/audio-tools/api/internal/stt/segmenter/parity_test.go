package segmenter

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter/testaudio"
)

// newFakeBatchExecutor returns a deterministic BatchExecutor that
// produces a canned Result with the given final text. The parity
// projection (kind + ordering + final transcript) compares across
// transports/strategies, so identifying provider metadata is fixed.
func newFakeBatchExecutor(text string) *sttmocks.FakeBatchExecutor {
	return &sttmocks.FakeBatchExecutor{Result: &sttchain.Result{
		Text:       text,
		Tier:       sttchain.TierLocal,
		ProviderID: "fake-local",
		ModelID:    "fake-model",
		Latency:    1 * time.Millisecond,
	}}
}

// EventProjection is the stable shape compared across pipeline paths
// in the parity test. Provider-trace fields (latency, model id) and
// timing-sensitive fields are excluded so the projection holds across
// transports and strategies.
type EventProjection struct {
	Kind sttchain.StreamEventKind
	Text string
}

func projectEvents(seq []sttchain.StreamEvent) []EventProjection {
	out := make([]EventProjection, 0, len(seq))
	for _, ev := range seq {
		if ev.Kind == sttchain.StreamEventAcknowledgement {
			// Coverage acknowledgements are transport durability metadata, not
			// transcript output; Connect/WS adapters project them differently.
			continue
		}
		p := EventProjection{Kind: ev.Kind}
		switch ev.Kind {
		case sttchain.StreamEventPartial:
			if ev.Partial != nil {
				p.Text = ev.Partial.Text
			}
		case sttchain.StreamEventSegment:
			if ev.Segment != nil {
				p.Text = ev.Segment.Text
			}
		case sttchain.StreamEventDone:
			if ev.Done != nil {
				p.Text = ev.Done.FinalText
			}
		}
		out = append(out, p)
	}
	return out
}

// runDirect builds a Segmenter against a Chain with no providers
// enabled. StreamCandidates returns empty, the selector falls back to
// BufferedFallback, and the supplied BatchExecutor stand-in produces
// the deterministic result.
func runDirect(t *testing.T, audio []byte, cfg stt.StreamConfig) []EventProjection {
	t.Helper()
	exec := newFakeBatchExecutor("hello world")
	chain := sttchain.NewChain(sttchain.Options{})
	selector := stt.NewSelector(exec)
	seg := New(Deps{Chain: chain, Selector: selector})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chunks := make(chan sttchain.AudioChunk, 1)
	chunks <- sttchain.AudioChunk{Audio: audio}
	close(chunks)

	events := make(chan sttchain.StreamEvent, 8)
	go func() {
		_ = seg.Run(ctx, sttchain.StreamStart{}, cfg, chunks, events)
	}()

	var seq []sttchain.StreamEvent
	for ev := range events {
		seq = append(seq, ev)
	}
	return projectEvents(seq)
}

// TestStreamingParity is the load-bearing parity test the plan
// requires. Phase B asserts a single path (direct Segmenter via
// BufferedFallback) with the canonical projection; Phase C extends it
// to the Connect bidi transport; Phase D extends it to the browser WS
// transport.
func TestStreamingParity(t *testing.T) {
	defer goleak.VerifyNone(t)

	cases := []struct {
		name  string
		audio []byte
		want  []EventProjection
	}{
		{
			name:  "speech_like_buffered_fallback",
			audio: testaudio.SpeechTonePauseTone3s(),
			want: []EventProjection{
				{Kind: sttchain.StreamEventSegment, Text: "hello world"},
				{Kind: sttchain.StreamEventDone, Text: "hello world"},
			},
		},
		{
			name:  "silence_buffered_fallback",
			audio: testaudio.Silence1s(),
			want: []EventProjection{
				{Kind: sttchain.StreamEventSegment, Text: "hello world"},
				{Kind: sttchain.StreamEventDone, Text: "hello world"},
			},
		},
	}

	cfg := stt.Defaults()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			direct := runDirect(t, tc.audio, cfg)
			if !equalProjections(direct, tc.want) {
				t.Fatalf("direct projection mismatch:\n got=%+v\nwant=%+v", direct, tc.want)
			}
		})
	}
}

func equalProjections(a, b []EventProjection) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
