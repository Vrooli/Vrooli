package eval

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/strategy"
)

func fakeProv(text string, latency time.Duration) *sttmocks.FakeProvider {
	p := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true})
	p.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: text, Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: latency}, nil
	}
	return p
}

// controlledSession drains the replayed chunks, makes ONE metered backend
// call over the concatenated audio (so the meter records calls +
// audio-seconds), then emits a single Segment + terminal Done carrying a
// caller-controlled final text. It isolates the harness's metric/report
// wiring from any strategy's commit quirks.
func controlledSession(meter *MeteredProvider, finalText string) Session {
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		var audio []byte
		for ch := range chunks {
			audio = append(audio, ch.Audio...)
		}
		_, _ = meter.Transcribe(ctx, sttchain.Request{Audio: audio})
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: finalText}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: finalText}}
		close(events)
		return nil
	}
}

func silentPCM(sampleRate int, ms int) []byte {
	return make([]byte, sampleRate*2*ms/1000)
}

func TestRunEval_AggregatesQualityAndCompute(t *testing.T) {
	clips := []Clip{
		{ID: "c1", PCM: silentPCM(16000, 1000), SampleRate: 16000, Reference: "the quick brown fox"},
		{ID: "c2", PCM: silentPCM(16000, 2000), SampleRate: 16000, Reference: "hello world"},
	}

	perfect := StrategySpec{
		Kind: sttchain.StrategyOverlapAgree, Label: "perfect",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("x", 20*time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, clip.Reference), meter
		},
	}
	// lossy drops the last reference word from each clip -> one deletion/clip.
	lossy := StrategySpec{
		Kind: sttchain.StrategyVADSegment, Label: "lossy",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("x", 10*time.Millisecond), float64(clip.bytesPerSecond()))
			words := strings.Fields(clip.Reference)
			dropped := strings.Join(words[:len(words)-1], " ")
			return controlledSession(meter, dropped), meter
		},
	}

	rep := RunEval(context.Background(), clips, []StrategySpec{perfect, lossy}, DefaultEvalOptions())
	require.True(t, rep.QualityMeasured)
	require.False(t, rep.LatencyMeasured)
	require.Len(t, rep.PerStrategy, 2)

	p := rep.PerStrategy[0]
	require.Equal(t, "perfect", p.Label)
	require.InDelta(t, 0.0, p.WER, 1e-9, "perfect strategy has zero WER")
	require.Equal(t, 6, p.RefWords, "4 + 2 reference words across both clips")
	require.Equal(t, 2, p.WhisperCalls, "one metered call per clip")
	require.InDelta(t, 3.0, p.WhisperAudioSeconds, 1e-9, "1s + 2s of audio")
	require.Greater(t, p.RTF, 0.0)
	require.Len(t, p.PerClip, 2)

	l := rep.PerStrategy[1]
	require.Equal(t, "lossy", l.Label)
	require.Equal(t, 2, l.EditCounts.Deletions, "one dropped word per clip")
	require.Equal(t, 0, l.EditCounts.Substitutions)
	require.Equal(t, 0, l.EditCounts.Insertions)
	require.InDelta(t, 2.0/6.0, l.WER, 1e-9, "micro-average WER = total edits / total ref words")
}

// TestRunEval_RealOverlapAgreePath proves the harness drives a REAL
// OverlapAgree strategy end-to-end (replay -> strategy -> events -> WER),
// metering the backend calls, without needing a live Whisper.
func TestRunEval_RealOverlapAgreePath(t *testing.T) {
	clip := Clip{ID: "c1", PCM: silentPCM(16000, 500), SampleRate: 16000, Reference: "hello world"}

	spec := StrategySpec{
		Kind: sttchain.StrategyOverlapAgree, Label: "overlap-agree",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("hello world", time.Millisecond), float64(clip.bytesPerSecond()))
			strat := &strategy.OverlapAgree{
				Provider: meter, Trigger: strategy.TriggerStopwatch,
				WindowMs: 100, AdvanceMs: 100, CommitRuns: 2, SampleRate: 16000,
			}
			return StrategySession(func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
				return strat.Run(ctx, sttchain.StreamStart{}, chunks, events)
			}), meter
		},
	}

	rep := RunEval(context.Background(), []Clip{clip}, []StrategySpec{spec}, DefaultEvalOptions())
	require.Len(t, rep.PerStrategy, 1)
	row := rep.PerStrategy[0]
	require.InDelta(t, 0.0, row.WER, 1e-9, "agreement commits the full reference -> zero WER")
	require.GreaterOrEqual(t, row.WhisperCalls, 2, "overlap-agree calls the backend per settle attempt")
	require.Equal(t, "hello world", strings.TrimSpace(row.PerClip[0].Hypothesis))
}

// TestRunEval_RealtimeProducesLatencySamples proves the real-time pass
// records one finalization-latency sample per repeat. Sleep is stubbed and
// Now is a deterministic monotone counter, so the test never actually
// sleeps and stays reproducible.
func TestRunEval_RealtimeProducesLatencySamples(t *testing.T) {
	clip := Clip{ID: "c1", PCM: silentPCM(16000, 300), SampleRate: 16000, Reference: "hello world"}
	spec := StrategySpec{
		Kind: sttchain.StrategyOverlapAgree, Label: "overlap-agree",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("hello world", time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, "hello world"), meter
		},
	}

	opts := EvalOptions{
		ChunkMs: 100, QualityPass: true, RealtimeRepeats: 3,
		Sleep: func(time.Duration) {}, // never actually sleep
	}
	rep := RunEval(context.Background(), []Clip{clip}, []StrategySpec{spec}, opts)
	require.True(t, rep.LatencyMeasured)
	require.Len(t, rep.PerStrategy[0].PerClip, 1)
	require.Len(t, rep.PerStrategy[0].PerClip[0].LatencySamplesMs, 3, "one latency sample per real-time repeat")
}
