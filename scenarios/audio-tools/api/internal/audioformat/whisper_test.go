package audioformat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareForWhisperPCMWrapsWAV(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return false })) // no ffmpeg needed
	pcm := []byte{0x01, 0x00, 0xFF, 0x7F}
	payload, filename, err := e.PrepareForWhisper(CodecPCMS16LE, pcm)
	require.NoError(t, err)
	require.Equal(t, "recording.wav", filename)
	require.Equal(t, "RIFF", string(payload[0:4]))
	require.Equal(t, pcm, payload[44:], "PCM bytes preserved after the 44-byte header")
}

func TestPrepareForWhisperContainerPassthrough(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return false }))
	cases := map[Codec]string{
		CodecWebM: "recording.webm",
		CodecMP3:  "recording.mp3",
		CodecOGG:  "recording.ogg",
		CodecAAC:  "recording.m4a",
	}
	for codec, wantName := range cases {
		in := []byte("container-bytes")
		payload, filename, err := e.PrepareForWhisper(codec, in)
		require.NoError(t, err)
		require.Equal(t, wantName, filename)
		require.Equal(t, in, payload, "containers pass through untouched (Whisper decodes them)")
	}
}

func TestPrepareForWhisperUnknownErrors(t *testing.T) {
	e := New()
	_, _, err := e.PrepareForWhisper(CodecUnknown, []byte("?"))
	require.ErrorIs(t, err, ErrUnknownFormat)
}
