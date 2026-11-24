package audio_test

import (
	"context"
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio"
	"audio-tools/internal/audio/mocks"
)

func TestMergeCrossfadeMapsFinalLabel(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	sources := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	_, err := audio.Merge(context.Background(), sources, nil, "wav", 0.5)
	require.NoError(t, err)

	filter, mapLabel := filterAndMap(t, lastFfmpegCall(t, fake).Args)
	require.Equal(t, "[a2]", mapLabel)
	require.Contains(t, filter, "acrossfade")
	require.True(t, strings.HasSuffix(filter, "[a2]"), "filter must emit the mapped label: %s", filter)
}

func TestMergeConcatMapsConcatOutput(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b")}, nil, "wav", 0)
	require.NoError(t, err)

	filter, mapLabel := filterAndMap(t, lastFfmpegCall(t, fake).Args)
	require.Equal(t, "[a]", mapLabel)
	require.Contains(t, filter, "concat=n=2:v=0:a=1[a]")
}

func TestMergeHonorsInputFormats(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	_, err := audio.Merge(context.Background(),
		[][]byte{[]byte("a"), []byte("b")}, []string{"wav", "opus"}, "wav", 0)
	require.NoError(t, err)

	args := fake.Calls[0].Args
	require.True(t, hasSubsequence(args, "-f", "wav", "-i"), "first source hint missing: %v", args)
	// Declared Opus arrives in an Ogg container; there is no "opus"
	// demuxer.
	require.True(t, hasSubsequence(args, "-f", "ogg", "-i"), "opus source hint missing: %v", args)
}

func TestMergeRejectsEmptySource(t *testing.T) {
	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), {}}, nil, "wav", 0)
	require.Error(t, err)
	require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
}

func TestMergeRejectsInvalidCrossfade(t *testing.T) {
	for _, crossfade := range []float64{-1, math.NaN(), math.Inf(1)} {
		_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b")}, nil, "wav", crossfade)
		require.Error(t, err)
		require.True(t, errors.Is(err, audio.ErrInvalidArgument), "crossfade=%v err=%v", crossfade, err)
	}
}

func TestMergeRemovesTempFilesOnSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b")}, nil, "wav", 0)
	require.NoError(t, err)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "merge leaked temp inputs")
}

func TestMergeRemovesTempFilesOnRunnerError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	fake := mocks.NewFakeRunner(nil, errors.New("ffmpeg failed"))
	swapRunner(t, fake)

	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b")}, nil, "wav", 0)
	require.Error(t, err)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "merge leaked temp inputs after a failed run")
}

func TestProbeTolerantOfMissingValues(t *testing.T) {
	const probe = `{"streams":[{"codec_name":"pcm_s16le","sample_rate":"N/A","channels":0,"bit_rate":"N/A"}],"format":{"format_name":"wav","duration":"N/A","bit_rate":"N/A"}}`
	fake := mocks.NewFakeRunner([]byte(probe), nil)
	swapRunner(t, fake)

	meta, err := audio.Probe(context.Background(), []byte("X"))
	require.NoError(t, err, "ffprobe's N/A values must not fail metadata extraction")
	require.Zero(t, meta.DurationSeconds)
	require.Zero(t, meta.SampleRate)
	require.Zero(t, meta.Bitrate)
	require.Equal(t, "pcm_s16le", meta.Codec)
	require.Nil(t, meta.Tags)
}

func TestProbeSelectsAudioStreamAndHidesBanner(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte(`{"streams":[],"format":{}}`), nil)
	swapRunner(t, fake)

	_, err := audio.Probe(context.Background(), []byte("X"))
	require.NoError(t, err)
	args := fake.Calls[0].Args
	require.True(t, hasSubsequence(args, "-select_streams", "a:0"), "%v", args)
	require.Contains(t, args, "-hide_banner")
	// ffprobe rejects -nostdin outright; input is a file, so stdin is
	// never read.
	require.NotContains(t, args, "-nostdin")
}

func TestProbeNonFiniteDurationBecomesZero(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte(`{"streams":[],"format":{"duration":"inf","format_name":"wav"}}`), nil)
	swapRunner(t, fake)

	meta, err := audio.Probe(context.Background(), []byte("X"))
	require.NoError(t, err)
	require.Zero(t, meta.DurationSeconds)
}

func TestProbeContextCanceled(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte(`{}`), nil)
	swapRunner(t, fake)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := audio.Probe(ctx, []byte("X"))
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), "want context.Canceled, got %v", err)
}

func TestSplitExplicitBoundariesReportLastChunkDuration(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"10.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 0, []float64{2, 5}, "wav")
	require.NoError(t, err)
	require.Len(t, chunks, 3)
	require.Equal(t, []float64{2, 3, 5}, []float64{chunks[0].Duration, chunks[1].Duration, chunks[2].Duration},
		"the open-ended final chunk must report duration = input duration - start")
}

func TestSplitExplicitBoundariesSurviveProbeFailure(t *testing.T) {
	fake := respondFake([]byte("OUT"), "", errors.New("ffprobe exploded"))
	swapRunner(t, fake)

	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 0, []float64{2}, "wav")
	require.NoError(t, err, "an unavailable probe must not fail an explicit-boundary split")
	require.Len(t, chunks, 2)
	require.Equal(t, 2.0, chunks[0].Duration)
	require.Equal(t, 0.0, chunks[1].Duration, "unknown final duration is reported as unknown, not guessed")
}
