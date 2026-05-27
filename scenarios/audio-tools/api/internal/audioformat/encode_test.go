package audioformat

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio/mocks"
)

func TestEncodeWAVNativeNoFfmpeg(t *testing.T) {
	runner := mocks.NewFakeRunner(nil, nil)
	e := New(WithRunner(runner), WithFfmpegProbe(func() bool { return false }))

	pcm := []byte{0x01, 0x00, 0xFF, 0x7F}
	out, ct, err := e.Encode(context.Background(), pcm, OutputWAV)
	require.NoError(t, err)
	require.Equal(t, "audio/wav", ct)
	require.Empty(t, runner.Calls, "WAV egress must be native, no ffmpeg")

	require.Equal(t, "RIFF", string(out[0:4]))
	require.Equal(t, "WAVE", string(out[8:12]))
	require.Equal(t, "data", string(out[36:40]))
	require.Equal(t, uint32(len(pcm)), binary.LittleEndian.Uint32(out[40:44]))
	require.Equal(t, pcm, out[44:])
	require.Equal(t, uint32(CanonicalSampleRate), binary.LittleEndian.Uint32(out[24:28]))
}

func TestEncodeFfmpegFormats(t *testing.T) {
	for _, f := range []OutputFormat{OutputMP3, OutputOpus, OutputFLAC} {
		t.Run(string(f), func(t *testing.T) {
			runner := mocks.NewFakeRunner([]byte("ENC"), nil)
			e := New(WithRunner(runner), WithFfmpegProbe(func() bool { return true }))

			out, ct, err := e.Encode(context.Background(), []byte{0, 0}, f)
			require.NoError(t, err)
			require.Equal(t, "ENC", string(out))
			require.Equal(t, f.ContentType(), ct)
			require.Len(t, runner.Calls, 1)
			require.Equal(t, encodeArgs(f), runner.Calls[0].Args)
		})
	}
}

func TestEncodeNoFfmpegNonWAVErrors(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return false }))
	_, _, err := e.Encode(context.Background(), []byte{0, 0}, OutputMP3)
	require.ErrorIs(t, err, ErrFfmpegRequired)
}

func TestEncodeUnsupportedFormat(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return true }))
	_, _, err := e.Encode(context.Background(), []byte{0, 0}, OutputFormat("aiff"))
	require.ErrorIs(t, err, ErrUnsupportedOutput)
}
