package ingress_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	afmocks "audio-tools/internal/audioformat/mocks"
	"audio-tools/internal/stt/ingress"
)

// DenoiseEnhancer pumps inbound PCM through the FilterRunner and forwards the
// filtered frames. With an identity fake process the bytes round-trip, and the
// ffmpeg argv must declare canonical PCM in AND out plus the -af filter.
func TestDenoiseEnhancer_StreamsThroughFilter(t *testing.T) {
	runner := &afmocks.FakeProcessRunner{} // identity transform
	eng := audioformat.New(
		audioformat.WithProcessRunner(runner),
		audioformat.WithFfmpegProbe(func() bool { return true }),
	)
	d := ingress.DenoiseEnhancer{Runner: eng, Filter: "afftdn=nf=-20"}

	// Even-byte (whole-int16-sample) chunks so the frame aligner emits them.
	out, cleanup, err := d.Process(context.Background(), feed([]byte{1, 2, 3, 4}, []byte{5, 6}))
	require.NoError(t, err)
	defer cleanup()

	got := drain(t, out)
	// Identity transform → same bytes back (frame-aligned), order preserved.
	var flat []byte
	for _, c := range got {
		flat = append(flat, c...)
	}
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, flat)

	// The filter process must be started with canonical PCM in+out and the -af.
	require.Len(t, runner.Calls, 1)
	args := runner.Calls[0].Args
	require.Contains(t, args, "-af")
	require.Contains(t, args, "afftdn=nf=-20")
	require.Contains(t, args, "s16le")
	require.Contains(t, args, "16000")
}

// When ffmpeg is unavailable, NewStreamFilter errors; the enhancer surfaces it
// so the Segmenter (which only adds denoise when ffmpeg is present) treats it
// as a genuine failure rather than silently no-opping.
func TestDenoiseEnhancer_FfmpegUnavailableErrors(t *testing.T) {
	eng := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	d := ingress.DenoiseEnhancer{Runner: eng}
	_, _, err := d.Process(context.Background(), feed([]byte{1, 2}))
	require.ErrorIs(t, err, audioformat.ErrFfmpegRequired)
}

// The default filter is applied when none is set.
func TestDenoiseEnhancer_DefaultFilter(t *testing.T) {
	runner := &afmocks.FakeProcessRunner{}
	eng := audioformat.New(
		audioformat.WithProcessRunner(runner),
		audioformat.WithFfmpegProbe(func() bool { return true }),
	)
	d := ingress.DenoiseEnhancer{Runner: eng}
	out, cleanup, err := d.Process(context.Background(), feed([]byte{1, 2}))
	require.NoError(t, err)
	defer cleanup()
	drain(t, out)
	require.Contains(t, runner.Calls[0].Args, ingress.DefaultDenoiseFilter)
}
