package segmenter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/stt"
)

// staticExtractor is a no-op ingress.TargetExtractor for wiring tests.
type staticExtractor struct{}

func (staticExtractor) Extract(_ context.Context, pcm []byte) ([]byte, error) { return pcm, nil }

// TestBuildIngress_ComposesExtractionStage proves the per-session ingress
// pipeline includes the target-extraction stage exactly when a SpeakerExtraction
// extractor was built for the session, and orders denoise before extraction.
func TestBuildIngress_ComposesExtractionStage(t *testing.T) {
	ffmpegEngine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return true }))

	t.Run("no stages when neither configured", func(t *testing.T) {
		s := New(Deps{})
		require.Nil(t, s.buildIngress(stt.StreamConfig{}))
	})

	t.Run("extraction only", func(t *testing.T) {
		s := New(Deps{SpeakerExtraction: staticExtractor{}})
		p := s.buildIngress(stt.StreamConfig{})
		require.NotNil(t, p)
		require.Equal(t, []string{"target-extraction"}, p.Names())
	})

	t.Run("denoise then extraction (ordered)", func(t *testing.T) {
		s := New(Deps{Engine: ffmpegEngine, SpeakerExtraction: staticExtractor{}})
		p := s.buildIngress(stt.StreamConfig{DenoiseEnabled: true})
		require.NotNil(t, p)
		require.Equal(t, []string{"denoise", "target-extraction"}, p.Names())
	})

	t.Run("denoise only", func(t *testing.T) {
		s := New(Deps{Engine: ffmpegEngine})
		p := s.buildIngress(stt.StreamConfig{DenoiseEnabled: true})
		require.NotNil(t, p)
		require.Equal(t, []string{"denoise"}, p.Names())
	})
}
