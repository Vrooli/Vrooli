package audio

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// probeReport is a realistic loudnorm pass-1 report, complete with the log
// lines ffmpeg surrounds it with.
const probeReport = `[Parsed_loudnorm_0 @ 0x55f0]
{
	"input_i" : "-21.75",
	"input_tp" : "-18.06",
	"input_lra" : "0.00",
	"input_thresh" : "-31.75",
	"output_i" : "-24.05",
	"output_tp" : "-20.31",
	"output_lra" : "0.00",
	"output_thresh" : "-34.05",
	"normalization_type" : "linear",
	"target_offset" : "0.05"
}
size=N/A time=00:00:00.00 bitrate=N/A speed=0x
`

// encodedReport is the pass-2 report, whose output_i is the measured
// integrated loudness of the produced audio.
const encodedReport = `[Parsed_loudnorm_0 @ 0x55f1]
{
	"input_i" : "-21.75",
	"input_tp" : "-18.06",
	"input_lra" : "0.00",
	"input_thresh" : "-31.75",
	"output_i" : "-16.10",
	"output_tp" : "-12.31",
	"output_lra" : "0.00",
	"output_thresh" : "-26.10",
	"normalization_type" : "linear",
	"target_offset" : "0.05"
}
`

type captureCall struct {
	name  string
	stdin []byte
	args  []string
}

// captureRunner is the local StderrRunner fake for package-internal tests
// (the hoisted mocks package imports audio, so it cannot be used here).
type captureRunner struct {
	calls   []captureCall
	respond func(name string, args []string) (stdout, stderr []byte, err error)
}

func (r *captureRunner) Run(ctx context.Context, name string, stdin []byte, args ...string) ([]byte, error) {
	stdout, _, err := r.RunWithStderr(ctx, name, stdin, args...)
	return stdout, err
}

func (r *captureRunner) RunWithStderr(_ context.Context, name string, stdin []byte, args ...string) ([]byte, []byte, error) {
	r.calls = append(r.calls, captureCall{
		name:  name,
		stdin: append([]byte(nil), stdin...),
		args:  append([]string(nil), args...),
	})
	return r.respond(name, args)
}

func swapCaptureRunner(t *testing.T, r *captureRunner) {
	t.Helper()
	restore := SetFfmpegAvailableForTest(true, true)
	prev := DefaultRunner
	DefaultRunner = r
	t.Cleanup(func() {
		DefaultRunner = prev
		restore()
	})
}

func argAfter(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// hasNullOutput distinguishes the analysis pass from the encoding pass.
func hasNullOutput(args []string) bool {
	return hasAll(args, "-f", "null", "-")
}

func hasAll(args []string, needles ...string) bool {
	for _, n := range needles {
		found := false
		for _, a := range args {
			if a == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestLastJSONObjectTakesLastBalancedBlock(t *testing.T) {
	stderr := []byte(`Banner text with a stray {non-json} brace
[Parsed_loudnorm_0 @ 0x1] {"kind":"progress","note":"a } brace in a string"}
{"kind":"other"}
trailing size=N/A time=00:00:00.00`)

	block, ok := lastJSONObject(stderr)
	require.True(t, ok)
	require.JSONEq(t, `{"kind":"other"}`, string(block))

	single, ok := lastJSONObject([]byte("no json here"))
	require.False(t, ok)
	require.Nil(t, single)
}

func TestParseLoudnormReportFromNoisyStderr(t *testing.T) {
	report, err := parseLoudnormReport([]byte(probeReport))
	require.NoError(t, err)
	require.Equal(t, "-21.75", report.InputI)
	require.Equal(t, "-18.06", report.InputTP)
	require.Equal(t, "0.00", report.InputLRA)
	require.Equal(t, "-31.75", report.InputThresh)
	require.Equal(t, "0.05", report.TargetOffset)

	_, err = parseLoudnormReport([]byte("no report at all"))
	require.Error(t, err)

	// A complete but non-finite analysis (silence) is unusable for the
	// linear second pass and must be treated as a failed measurement.
	report, err = parseLoudnormReport([]byte(`{"input_i":"-inf","input_tp":"-inf","input_lra":"0.0","input_thresh":"-inf","target_offset":"0.0"}`))
	require.NoError(t, err)
	require.Error(t, validateLoudnormReport(report))
}

func TestNormalizeMeasuredTwoPassArgvAndResult(t *testing.T) {
	fake := &captureRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
		if hasNullOutput(args) {
			return nil, []byte(probeReport), nil
		}
		return []byte("ENCODED"), []byte(encodedReport), nil
	}}
	swapCaptureRunner(t, fake)

	out, measured, err := NormalizeMeasured(context.Background(), []byte("INPUT"), "wav", "ebu_r128", -16, "wav")
	require.NoError(t, err)
	require.Equal(t, []byte("ENCODED"), out)
	require.InDelta(t, -16.10, measured, 0.001)
	require.Len(t, fake.calls, 2)

	analysis, encode := fake.calls[0], fake.calls[1]
	require.Equal(t, "ffmpeg", analysis.name)
	require.Equal(t, []byte("INPUT"), analysis.stdin)
	analysisFilter, ok := argAfter(analysis.args, "-af")
	require.True(t, ok)
	require.Equal(t, "loudnorm=print_format=json", analysisFilter)
	require.True(t, hasNullOutput(analysis.args), "analysis pass must not write audio: %v", analysis.args)
	require.Equal(t, "info", valueAfter(analysis.args, "-loglevel"), "loudnorm report needs info logs")

	require.Equal(t, []byte("INPUT"), encode.stdin)
	encodeFilter, ok := argAfter(encode.args, "-af")
	require.True(t, ok)
	for _, want := range []string{
		"loudnorm=I=-16",
		"measured_I=-21.75",
		"measured_TP=-18.06",
		"measured_LRA=0.00",
		"measured_thresh=-31.75",
		"offset=0.05",
		"linear=true",
		"print_format=json",
	} {
		require.Contains(t, encodeFilter, want)
	}
	require.True(t, hasAll(encode.args, "-f", "wav", "pipe:1"))
}

func valueAfter(args []string, flag string) string {
	v, _ := argAfter(args, flag)
	return v
}

func TestNormalizeMeasuredFallsBackWhenAnalysisFails(t *testing.T) {
	for _, tc := range []struct {
		name    string
		respond func(args []string) ([]byte, []byte, error)
	}{
		{
			name: "ffmpeg error",
			respond: func(args []string) ([]byte, []byte, error) {
				if hasNullOutput(args) {
					return nil, []byte("broken pipe"), errors.New("exit status 1")
				}
				return []byte("SINGLEPASS"), nil, nil
			},
		},
		{
			name: "unparseable report",
			respond: func(args []string) ([]byte, []byte, error) {
				if hasNullOutput(args) {
					return nil, []byte("no json here"), nil
				}
				return []byte("SINGLEPASS"), nil, nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &captureRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
				return tc.respond(args)
			}}
			swapCaptureRunner(t, fake)

			out, measured, err := NormalizeMeasured(context.Background(), []byte("INPUT"), "wav", "ebu_r128", -16, "wav")
			require.NoError(t, err, "measurement failure must not fail a producible transform")
			require.Equal(t, []byte("SINGLEPASS"), out)
			require.Zero(t, measured, "unknown loudness must not be reported as the target")
			require.Len(t, fake.calls, 2)

			filter, ok := argAfter(fake.calls[1].args, "-af")
			require.True(t, ok)
			require.Equal(t, "loudnorm=I=-16", filter, "fallback keeps the single-pass filter")
		})
	}
}

func TestNormalizeMeasuredPass2WithoutOutputReport(t *testing.T) {
	fake := &captureRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
		if hasNullOutput(args) {
			return nil, []byte(probeReport), nil
		}
		return []byte("ENCODED"), []byte("no json"), nil
	}}
	swapCaptureRunner(t, fake)

	out, measured, err := NormalizeMeasured(context.Background(), []byte("INPUT"), "wav", "", 0, "")
	require.NoError(t, err)
	require.Equal(t, []byte("ENCODED"), out)
	require.Zero(t, measured)
	// Default target when the request omits one.
	filter, _ := argAfter(fake.calls[1].args, "-af")
	require.Contains(t, filter, "loudnorm=I=-16")
}

func TestNormalizeMeasuredPeakMeasuresProducedOutput(t *testing.T) {
	fake := &captureRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
		if hasNullOutput(args) {
			return nil, []byte(strings.Replace(probeReport, "-21.75", "-9.25", 1)), nil
		}
		return []byte("PEAKED"), nil, nil
	}}
	swapCaptureRunner(t, fake)

	out, measured, err := NormalizeMeasured(context.Background(), []byte("INPUT"), "wav", "peak", -14, "wav")
	require.NoError(t, err)
	require.Equal(t, []byte("PEAKED"), out)
	require.InDelta(t, -9.25, measured, 0.001)
	require.Len(t, fake.calls, 2)

	filter, _ := argAfter(fake.calls[0].args, "-af")
	require.Equal(t, "dynaudnorm", filter)
	_, ok := argAfter(fake.calls[1].args, "-af")
	require.True(t, ok, "produced output must be measured")
}

func TestNormalizeMeasuredPeakMeasurementFailureIsZero(t *testing.T) {
	fake := &captureRunner{respond: func(_ string, args []string) ([]byte, []byte, error) {
		if hasNullOutput(args) {
			return nil, nil, errors.New("measurement exploded")
		}
		return []byte("RMSED"), nil, nil
	}}
	swapCaptureRunner(t, fake)

	out, measured, err := NormalizeMeasured(context.Background(), []byte("INPUT"), "wav", "rms", -14, "wav")
	require.NoError(t, err)
	require.Equal(t, []byte("RMSED"), out)
	require.Zero(t, measured)
}

func TestNormalizeMeasuredRejectsNonFiniteTarget(t *testing.T) {
	fake := &captureRunner{respond: func(string, []string) ([]byte, []byte, error) {
		t.Fatal("no ffmpeg invocation expected for an invalid target")
		return nil, nil, nil
	}}
	swapCaptureRunner(t, fake)

	for _, target := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, _, err := NormalizeMeasured(context.Background(), []byte("INPUT"), "wav", "ebu_r128", target, "wav")
		require.ErrorIs(t, err, ErrInvalidArgument)
	}
	require.Empty(t, fake.calls)
}

// TestIntegrationNormalizePeakReportsMeasurement covers the non-EBU paths:
// dynaudnorm produces the audio, and the reported loudness comes from a
// real measurement of that output (not the unused target).
func TestIntegrationNormalizePeakReportsMeasurement(t *testing.T) {
	requireRealFFmpeg(t)
	src := sineWAV(t, 2, 44100)

	out, measured, err := NormalizeMeasured(context.Background(), src, "wav", "peak", -14, "wav")
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.NotZero(t, measured, "peak normalization must report the produced output's loudness")
	require.Greater(t, measured, -40.0)
	require.Less(t, measured, 0.0)
}

// TestIntegrationNormalizeMeasuredHitsTarget is the real-ffmpeg check that
// the reported value is a measurement of the audio produced, close to the
// requested target.
func TestIntegrationNormalizeMeasuredHitsTarget(t *testing.T) {
	requireRealFFmpeg(t)
	src := sineWAV(t, 3, 44100)

	out, measured, err := NormalizeMeasured(context.Background(), src, "wav", "ebu_r128", -16, "wav")
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.NotZero(t, measured, "two-pass loudnorm must report a real measurement")
	require.InDelta(t, -16, measured, 2.0, "measured loudness should land near the target")

	meta, err := Probe(context.Background(), out)
	require.NoError(t, err)
	require.Greater(t, meta.DurationSeconds, 2.5)
}
