package audio_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio"
	"audio-tools/internal/audio/mocks"
)

type splitBatchRunner struct {
	probeJSON   string
	err         error
	args        []string
	paths       []string
	stdoutCalls int
	call        func(context.Context) error
}

func (r *splitBatchRunner) Run(_ context.Context, name string, _ []byte, _ ...string) ([]byte, error) {
	if name == "ffprobe" {
		return []byte(r.probeJSON), nil
	}
	r.stdoutCalls++
	return []byte("TRIM"), nil
}

func (r *splitBatchRunner) RunToFiles(ctx context.Context, _ string, args ...string) error {
	r.args = append([]string(nil), args...)
	for i, arg := range args {
		if arg == "-map" && i+4 < len(args) && args[i+2] == "-f" {
			r.paths = append(r.paths, args[i+4])
		}
	}
	if r.call != nil {
		return r.call(ctx)
	}
	if r.err != nil {
		return r.err
	}
	for i, path := range r.paths {
		if err := os.WriteFile(path, []byte("OUT"+string(rune('0'+i))), 0600); err != nil {
			return err
		}
	}
	return nil
}

func TestNormalize_AllMethods(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	for _, m := range []string{"peak", "rms", "ebu"} {
		_, err := audio.Normalize(context.Background(), []byte("X"), "wav", m, -16.0, "wav")
		require.NoError(t, err)
	}
	// Default lufs path.
	_, err := audio.Normalize(context.Background(), []byte("X"), "wav", "ebu", 0, "")
	require.NoError(t, err)
}

func TestSplit_WithBoundaries(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 0, []float64{1.0, 2.0}, "wav")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(chunks), 1)
}

func TestSplit_WithChunkSeconds(t *testing.T) {
	probeJSON := `{"streams":[{"sample_rate":"16000","channels":1,"bit_rate":"128000","codec_name":"pcm_s16le"}],"format":{"duration":"6.0","format_name":"wav","bit_rate":"128000"}}`
	fake := &mocks.FakeRunner{
		Respond: func(name string, args []string) ([]byte, error) {
			if name == "ffprobe" {
				return []byte(probeJSON), nil
			}
			return []byte("OUT"), nil
		},
	}
	swapRunner(t, fake)
	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 2.0, nil, "wav")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(chunks), 2)
}

func TestSplitUniformBatchUsesOneFileOutputInvocation(t *testing.T) {
	runner := &splitBatchRunner{probeJSON: `{"streams":[],"format":{"duration":"6.0","format_name":"wav"}}`}
	swapRunner(t, runner)
	chunks, err := audio.Split(context.Background(), []byte("SOURCE"), "wav", 2, nil, "flac")
	require.NoError(t, err)
	require.Len(t, chunks, 3)
	require.Equal(t, []string{"OUT0", "OUT1", "OUT2"}, []string{string(chunks[0].Audio), string(chunks[1].Audio), string(chunks[2].Audio)})
	require.Equal(t, []float64{0, 2, 4}, []float64{chunks[0].Start, chunks[1].Start, chunks[2].Start})
	require.Equal(t, []float64{2, 2, 2}, []float64{chunks[0].Duration, chunks[1].Duration, chunks[2].Duration})
	require.Equal(t, "audio/flac", chunks[0].ContentType)
	joined := strings.Join(runner.args, " ")
	require.Contains(t, joined, "asplit=3[in0][in1][in2]")
	require.Contains(t, joined, "atrim=start=2:duration=2,asetpts=PTS-STARTPTS[out1]")
	require.Len(t, runner.paths, 3)
	for _, path := range runner.paths {
		_, statErr := os.Stat(path)
		require.True(t, os.IsNotExist(statErr), "temporary output still exists: %s", path)
	}
}

func TestSplitUniformBatchWrapsFailureAndCleansOutputs(t *testing.T) {
	runner := &splitBatchRunner{
		probeJSON: `{"streams":[],"format":{"duration":"6.0","format_name":"wav"}}`,
		err:       errors.New("ffmpeg failed"),
	}
	swapRunner(t, runner)
	_, err := audio.Split(context.Background(), []byte("SOURCE"), "wav", 2, nil, "wav")
	require.ErrorIs(t, err, audio.ErrFfmpegExec)
	require.Len(t, runner.paths, 3)
	for _, path := range runner.paths {
		_, statErr := os.Stat(path)
		require.True(t, os.IsNotExist(statErr), "temporary output still exists: %s", path)
	}
}

func TestSplitUniformBatchPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &splitBatchRunner{
		probeJSON: `{"streams":[],"format":{"duration":"6.0","format_name":"wav"}}`,
		call: func(context.Context) error {
			cancel()
			return context.Canceled
		},
	}
	swapRunner(t, runner)
	_, err := audio.Split(ctx, []byte("SOURCE"), "wav", 2, nil, "wav")
	require.ErrorIs(t, err, context.Canceled)
	for _, path := range runner.paths {
		_, statErr := os.Stat(path)
		require.True(t, os.IsNotExist(statErr), "temporary output still exists: %s", path)
	}
}

func TestSplitAboveBatchThresholdUsesTrimRoute(t *testing.T) {
	runner := &splitBatchRunner{probeJSON: `{"streams":[],"format":{"duration":"66.0","format_name":"wav"}}`}
	swapRunner(t, runner)
	chunks, err := audio.Split(context.Background(), []byte("SOURCE"), "wav", 2, nil, "wav")
	require.NoError(t, err)
	require.Len(t, chunks, 33)
	require.Zero(t, len(runner.paths), "large request should not create batch outputs")
	require.Equal(t, 33, runner.stdoutCalls, "large request should retain one trim process per chunk")
}

func TestMerge_NoSources(t *testing.T) {
	_, err := audio.Merge(context.Background(), nil, nil, "wav", 0)
	require.Error(t, err)
}

func TestMerge_Concat(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b")}, nil, "wav", 0)
	require.NoError(t, err)
}

func TestMerge_Crossfade(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b"), []byte("c")}, nil, "wav", 0.5)
	require.NoError(t, err)
}

func TestTranscodeOpts_AllSet(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "mp3", 22050, 2, 128000)
	require.NoError(t, err)
	require.Greater(t, len(fake.Calls), 0)
}

func TestTranscodeOpts_Defaults(t *testing.T) {
	fake := mocks.NewFakeRunner(streamedWAVForTest(), nil)
	swapRunner(t, fake)
	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "", 0, 0, 0)
	require.NoError(t, err)
}

func TestProbe_OK(t *testing.T) {
	probeJSON := `{"streams":[{"sample_rate":"16000","channels":1,"bit_rate":"128000","codec_name":"pcm_s16le"}],"format":{"duration":"3.0","format_name":"wav","bit_rate":"128000","tags":{"title":"x"}}}`
	fake := &mocks.FakeRunner{
		Respond: func(name string, args []string) ([]byte, error) {
			return []byte(probeJSON), nil
		},
	}
	swapRunner(t, fake)
	m, err := audio.Probe(context.Background(), []byte("X"))
	require.NoError(t, err)
	require.Equal(t, "wav", m.Format)
	require.Equal(t, int32(16000), m.SampleRate)
}
