package audio

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	intaudio "audio-tools/internal/audio"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// stderrHandlerRunner is an audio.Runner that also exposes stderr, which
// loudnorm needs for its measurement reports.
type stderrHandlerRunner struct {
	calls   int
	respond func(name string, args []string) (stdout, stderr []byte, err error)
}

func (r *stderrHandlerRunner) Run(ctx context.Context, name string, stdin []byte, args ...string) ([]byte, error) {
	stdout, _, err := r.RunWithStderr(ctx, name, stdin, args...)
	return stdout, err
}

func (r *stderrHandlerRunner) RunWithStderr(_ context.Context, name string, stdin []byte, args ...string) ([]byte, []byte, error) {
	r.calls++
	return r.respond(name, args)
}

func argValueIn(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

const (
	analysisReport = `[Parsed_loudnorm_0 @ 0x1]
{
	"input_i" : "-27.00",
	"input_tp" : "-10.00",
	"input_lra" : "3.00",
	"input_thresh" : "-37.00",
	"output_i" : "-27.00",
	"output_tp" : "-10.00",
	"output_lra" : "3.00",
	"output_thresh" : "-37.00",
	"normalization_type" : "dynamic",
	"target_offset" : "0.00"
}
size=N/A time=00:00:00.00 bitrate=N/A speed=0x
`
	encodeReport = `[Parsed_loudnorm_0 @ 0x2]
{
	"input_i" : "-27.00",
	"input_tp" : "-10.00",
	"input_lra" : "3.00",
	"input_thresh" : "-37.00",
	"output_i" : "-14.80",
	"output_tp" : "-1.50",
	"output_lra" : "0.00",
	"output_thresh" : "-24.80",
	"normalization_type" : "linear",
	"target_offset" : "0.00"
}
`
)

func TestNormalize_MeasuredLufsComesFromFfmpeg(t *testing.T) {
	// The request asks for -9 LUFS; ffmpeg reports the produced audio at
	// -14.8 LUFS. The response must carry the measurement, not the target.
	runner := &stderrHandlerRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
		if argValueIn(args, "-f") == "null" {
			return nil, []byte(analysisReport), nil
		}
		return []byte("NORM"), []byte(encodeReport), nil
	}}
	withRunner(t, runner)
	c := newClient(t)

	res, err := c.Normalize(context.Background(), connect.NewRequest(&audiov1.NormalizeRequest{
		Audio:        []byte("X"),
		Format:       commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		Method:       audiov1.NormalizationMethod_NORMALIZATION_METHOD_EBU_R128,
		TargetLufs:   -9,
		OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.NoError(t, err)
	require.InDelta(t, -14.8, res.Msg.GetMeasuredLufs(), 1e-9)
	require.NotEqual(t, float64(-9), res.Msg.GetMeasuredLufs(), "measured_lufs must not echo the request target")
	require.Equal(t, []byte("NORM"), res.Msg.GetAudio())
	require.Equal(t, 2, runner.calls)
}

func TestNormalize_UnreportedLoudnessIsZero(t *testing.T) {
	// The encode succeeds but its measurement is unreadable; unknown
	// loudness must surface as 0, never as the requested target.
	runner := &stderrHandlerRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
		if argValueIn(args, "-f") == "null" {
			return nil, []byte(analysisReport), nil
		}
		return []byte("NORM"), []byte("no json here"), nil
	}}
	withRunner(t, runner)
	c := newClient(t)

	res, err := c.Normalize(context.Background(), connect.NewRequest(&audiov1.NormalizeRequest{
		Audio:      []byte("X"),
		Format:     commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		Method:     audiov1.NormalizationMethod_NORMALIZATION_METHOD_EBU_R128,
		TargetLufs: -9,
	}))
	require.NoError(t, err)
	require.Zero(t, res.Msg.GetMeasuredLufs())
	require.NotEmpty(t, res.Msg.GetAudio())
}

var (
	_ intaudio.Runner       = (*stderrHandlerRunner)(nil)
	_ intaudio.StderrRunner = (*stderrHandlerRunner)(nil)
)
