package audioformat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilitiesWithFfmpeg(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return true }))
	require.ElementsMatch(t,
		[]Codec{CodecPCMS16LE, CodecWAV, CodecMP3, CodecFLAC, CodecOGG, CodecWebM, CodecOpus, CodecAAC},
		e.Accepts())
	require.ElementsMatch(t,
		[]OutputFormat{OutputMP3, OutputWAV, OutputOpus, OutputFLAC},
		e.Emits())
	require.True(t, e.CanEmit(OutputOpus))
}

func TestCapabilitiesWithoutFfmpeg(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return false }))
	// PCM always accepted (fast-path); container codecs need ffmpeg.
	require.Equal(t, []Codec{CodecPCMS16LE}, e.Accepts())
	// WAV always emittable (native header); the rest need ffmpeg.
	require.Equal(t, []OutputFormat{OutputWAV}, e.Emits())
	require.False(t, e.CanEmit(OutputMP3))
	require.True(t, e.CanEmit(OutputWAV))
}
