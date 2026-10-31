package segmenter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/stt"
)

// buildIngress is the denoise capability gate: it returns a denoise pipeline
// only when DenoiseEnabled is set AND ffmpeg is available. "config on but no
// ffmpeg" must degrade to nil (no-op) rather than failing the session.
func TestBuildIngress_Gating(t *testing.T) {
	withFfmpeg := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return true }))
	noFfmpeg := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))

	t.Run("disabled → nil", func(t *testing.T) {
		s := &Segmenter{deps: Deps{Engine: withFfmpeg}}
		require.Nil(t, s.buildIngress(stt.StreamConfig{DenoiseEnabled: false}))
	})

	t.Run("enabled + ffmpeg → denoise pipeline", func(t *testing.T) {
		s := &Segmenter{deps: Deps{Engine: withFfmpeg}}
		p := s.buildIngress(stt.StreamConfig{DenoiseEnabled: true})
		require.NotNil(t, p)
		require.Equal(t, []string{"denoise"}, p.Names())
	})

	t.Run("enabled but no ffmpeg → nil (degrade, not fail)", func(t *testing.T) {
		s := &Segmenter{deps: Deps{Engine: noFfmpeg}}
		require.Nil(t, s.buildIngress(stt.StreamConfig{DenoiseEnabled: true}))
	})
}
