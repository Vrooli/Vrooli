//go:build whisper_integration

// Package eval's real-Whisper integration test. Build-tagged so it is
// EXCLUDED from the default suite (which must run without a live backend);
// the deterministic fake-provider tests in harness_test.go carry the
// default-suite coverage. Run it against a live Whisper sidecar with:
//
//	AUDIO_TOOLS_WHISPER_URL=http://127.0.0.1:9000 \
//	  go test -tags whisper_integration ./internal/eval/ -run Integration -v
//
// It replays the checked-in 1s smoke fixture through batch + vad-segment +
// overlap-agree and prints the comparison table, asserting the report
// shape and that batch (the single-pass oracle) is the WER floor.
package eval

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	voice "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/strategy"
)

func TestIntegration_SmokeFixtureThreeStrategies(t *testing.T) {
	whisperURL := os.Getenv("AUDIO_TOOLS_WHISPER_URL")
	if whisperURL == "" {
		t.Skip("set AUDIO_TOOLS_WHISPER_URL to run the real-Whisper eval integration test")
	}

	engine := audioformat.New()
	if !engine.HasFfmpeg() {
		t.Skip("ffmpeg required to decode the smoke fixture to canonical PCM")
	}

	// Decode the checked-in smoke fixture (wav) to canonical s16le mono 16k PCM.
	wavPath := filepath.Join("..", "diagnostics", "fixtures", "smoke.wav")
	wavBytes, err := os.ReadFile(wavPath)
	require.NoError(t, err)
	refBytes, err := os.ReadFile(filepath.Join("..", "diagnostics", "fixtures", "smoke_text.txt"))
	require.NoError(t, err)

	ctx := context.Background()
	pcm, err := engine.Normalize(ctx, audioformat.CodecFromString("wav"), wavBytes)
	require.NoError(t, err)

	clip := Clip{ID: "smoke", PCM: pcm, SampleRate: 16000, Reference: string(refBytes)}

	newProvider := func() *sttchain.LocalProvider {
		svc := voice.NewService(
			voice.Config{}, "", nil, "",
			voice.SpeakerConfig{}, "",
			&voice.SpeakerClient{},
			nil, new(atomic.Int64),
			whisperURL+"/asr?output=json",
			&http.Client{Timeout: 60 * time.Second}, engine,
		)
		return sttchain.NewLocalProvider(svc)
	}

	specs := []StrategySpec{
		{
			Kind: sttchain.StrategyBuffered, Label: "batch",
			BuildSession: func(clip Clip) (Session, *MeteredProvider) {
				meter := NewMeteredProvider(newProvider(), float64(clip.bytesPerSecond()))
				return func(c context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
					var audio []byte
					for ch := range chunks {
						audio = append(audio, ch.Audio...)
					}
					res, err := meter.Transcribe(c, sttchain.Request{Audio: audio, Format: "pcm_s16le"})
					text := ""
					if res != nil {
						text = res.Text
					}
					events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: text}}
					events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: text}}
					close(events)
					return err
				}, meter
			},
		},
		{
			Kind: sttchain.StrategyVADSegment, Label: "vad-segment",
			BuildSession: func(clip Clip) (Session, *MeteredProvider) {
				meter := NewMeteredProvider(newProvider(), float64(clip.bytesPerSecond()))
				strat := &strategy.VADSegmenter{Provider: meter}
				return StrategySession(func(c context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
					return strat.Run(c, sttchain.StreamStart{InputFormat: "pcm_s16le"}, chunks, events)
				}), meter
			},
		},
		{
			Kind: sttchain.StrategyOverlapAgree, Label: "overlap-agree(stall=3)",
			BuildSession: func(clip Clip) (Session, *MeteredProvider) {
				meter := NewMeteredProvider(newProvider(), float64(clip.bytesPerSecond()))
				strat := &strategy.OverlapAgree{Provider: meter, Trigger: strategy.TriggerVAD, MaxStallRejects: 3, SampleRate: 16000}
				return StrategySession(func(c context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
					return strat.Run(c, sttchain.StreamStart{InputFormat: "pcm_s16le"}, chunks, events)
				}), meter
			},
		},
	}

	opts := EvalOptions{ChunkMs: 100, QualityPass: true, RealtimeRepeats: 1, Sleep: time.Sleep}
	rep := RunEval(ctx, []Clip{clip}, specs, opts)
	require.Len(t, rep.PerStrategy, 3)

	var batchWER float64
	for _, s := range rep.PerStrategy {
		t.Logf("%-24s WER=%.3f calls=%d audio_s=%.2f rtf=%.2f latency_p50=%.0fms",
			s.Label, s.WER, s.WhisperCalls, s.WhisperAudioSeconds, s.RTF, s.FinalizationLatencyP50Ms)
		if s.Label == "batch" {
			batchWER = s.WER
		}
	}
	for _, s := range rep.PerStrategy {
		require.LessOrEqualf(t, batchWER, s.WER+1e-9, "batch must be the WER floor (oracle), %s beat it", s.Label)
	}
}
