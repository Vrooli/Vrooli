package audioformat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio/mocks"
)

func TestNormalizePCMFastPath(t *testing.T) {
	runner := mocks.NewFakeRunner([]byte("SHOULD-NOT-BE-USED"), nil)
	e := New(WithRunner(runner), WithFfmpegProbe(func() bool { return true }))

	in := []byte{0x01, 0x02, 0x03, 0x04}
	out, err := e.Normalize(context.Background(), CodecPCMS16LE, in)
	require.NoError(t, err)
	require.Equal(t, in, out)
	require.Empty(t, runner.Calls, "PCM fast-path must not invoke ffmpeg")
}

func TestNormalizeDecodesViaFfmpeg(t *testing.T) {
	runner := mocks.NewFakeRunner([]byte("PCMOUT"), nil)
	e := New(WithRunner(runner), WithFfmpegProbe(func() bool { return true }))

	out, err := e.Normalize(context.Background(), CodecWebM, []byte("webm-bytes"))
	require.NoError(t, err)
	require.Equal(t, "PCMOUT", string(out))

	require.Len(t, runner.Calls, 1)
	require.Equal(t, "ffmpeg", runner.Calls[0].Name)
	require.Equal(t, []byte("webm-bytes"), runner.Calls[0].Stdin)
	require.Equal(t, canonicalDecodeArgs(), runner.Calls[0].Args)
}

func TestNormalizeNoFfmpegErrors(t *testing.T) {
	runner := mocks.NewFakeRunner(nil, nil)
	e := New(WithRunner(runner), WithFfmpegProbe(func() bool { return false }))

	_, err := e.Normalize(context.Background(), CodecWebM, []byte("webm"))
	require.ErrorIs(t, err, ErrFfmpegRequired)
	require.Empty(t, runner.Calls)
}

func TestNormalizeUnknownErrors(t *testing.T) {
	e := New(WithFfmpegProbe(func() bool { return true }))
	_, err := e.Normalize(context.Background(), CodecUnknown, []byte("x"))
	require.ErrorIs(t, err, ErrUnknownFormat)
}
