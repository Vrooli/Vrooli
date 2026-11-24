package audio_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio"
	"audio-tools/internal/audio/mocks"
)

// respondFake builds a FakeRunner that answers ffprobe calls with
// probeJSON (or probeErr) and every ffmpeg call with stdout.
func respondFake(stdout []byte, probeJSON string, probeErr error) *mocks.FakeRunner {
	return &mocks.FakeRunner{
		Respond: func(name string, _ []string) ([]byte, error) {
			if name == "ffprobe" {
				if probeErr != nil {
					return nil, probeErr
				}
				return []byte(probeJSON), nil
			}
			return stdout, nil
		},
	}
}

func argValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func hasSubsequence(haystack []string, want ...string) bool {
	if len(want) == 0 {
		return true
	}
	next := 0
	for _, h := range haystack {
		if h == want[next] {
			next++
			if next == len(want) {
				return true
			}
		}
	}
	return false
}

// TestRunFfmpegBaseFlags locks the portable invocation contract: every
// ffmpeg child is non-interactive, banner-free, and quiet.
func TestRunFfmpegBaseFlags(t *testing.T) {
	fake := mocks.NewFakeRunner(streamedWAVForTest(), nil)
	swapRunner(t, fake)

	_, err := audio.Transcode(context.Background(), []byte("X"))
	require.NoError(t, err)
	require.NotEmpty(t, fake.Calls)
	args := fake.Calls[0].Args
	require.True(t, hasSubsequence(args, "-nostdin", "-hide_banner", "-y", "-loglevel", "error"),
		"base flags missing or out of order: %v", args)
}

// TestRunFfmpegCancellationSurfacesContext proves cancellation is not
// reported as an ffmpeg rejection (which handlers map to
// InvalidArgument). A canceled request must map to Canceled, and no
// child process should be spawned.
func TestRunFfmpegCancellationSurfacesContext(t *testing.T) {
	fake := mocks.NewFakeRunner(nil, errors.New("signal: killed"))
	swapRunner(t, fake)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := audio.Transcode(ctx, []byte("X"))
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), "want context.Canceled, got %v", err)
	require.False(t, errors.Is(err, audio.ErrFfmpegExec), "cancellation must not look like an ffmpeg rejection")
	require.Empty(t, fake.Calls, "canceled request must not spawn ffmpeg")
}

func TestTranscodeAACUsesADTSMuxer(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "aac", 0, 0, 0)
	require.NoError(t, err)
	muxer, ok := argValue(fake.Calls[0].Args, "-f")
	require.True(t, ok)
	// ffmpeg has no "aac" muxer; raw AAC is ADTS. Requesting -f aac fails
	// with "Requested output format 'aac' is not known".
	require.Equal(t, "adts", muxer)
	require.True(t, strings.HasSuffix(fake.Calls[0].Args[len(fake.Calls[0].Args)-1], "pipe:1"))
}

func TestTranscodeOpusUsesOpusMuxer(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "opus", 0, 0, 0)
	require.NoError(t, err)
	muxer, ok := argValue(fake.Calls[0].Args, "-f")
	require.True(t, ok)
	require.Equal(t, "opus", muxer)
}

// TestTranscodeUnknownFormatIsRejectedByFfmpeg verifies an unknown output
// format is passed through to ffmpeg (whose rejection maps to
// InvalidArgument) instead of being silently rewritten to a different
// container.
func TestTranscodeUnknownFormatIsRejectedByFfmpeg(t *testing.T) {
	fake := mocks.NewFakeRunner(nil, errors.New("Requested output format 'bogus' is not known"))
	swapRunner(t, fake)

	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "bogus", 0, 0, 0)
	require.Error(t, err)
	require.True(t, errors.Is(err, audio.ErrFfmpegExec), "want ErrFfmpegExec, got %v", err)
	muxer, ok := argValue(fake.Calls[0].Args, "-f")
	require.True(t, ok)
	require.Equal(t, "bogus", muxer)
}

func TestTranscodeWithInputEmitsDemuxerHint(t *testing.T) {
	cases := map[string]string{"wav": "wav", "opus": "ogg", "aac": "aac"}
	for inFmt, wantDemuxer := range cases {
		t.Run(inFmt, func(t *testing.T) {
			fake := mocks.NewFakeRunner(streamedWAVForTest(), nil)
			swapRunner(t, fake)
			_, err := audio.TranscodeWithInput(context.Background(), []byte("X"), inFmt, "wav", 0, 0, 0)
			require.NoError(t, err)
			args := fake.Calls[0].Args
			// The input hint must be the -f immediately before -i pipe:0.
			idx := indexOf(args, "-i")
			require.Greater(t, idx, 0, "missing -i: %v", args)
			require.Equal(t, "-f", args[idx-2])
			require.Equal(t, wantDemuxer, args[idx-1])
		})
	}
}

func TestTranscodeWithInputOmitsHintWhenUnset(t *testing.T) {
	fake := mocks.NewFakeRunner(streamedWAVForTest(), nil)
	swapRunner(t, fake)

	_, err := audio.TranscodeWithInput(context.Background(), []byte("X"), "", "wav", 0, 0, 0)
	require.NoError(t, err)
	args := fake.Calls[0].Args
	idx := indexOf(args, "-i")
	require.Greater(t, idx, 0)
	require.NotEqual(t, "-f", args[idx-2], "unset input format must not force a demuxer: %v", args)
}

func TestTrimRejectsInvalidRanges(t *testing.T) {
	cases := map[string]struct {
		start, end float64
	}{
		"negative start":      {-1, 0},
		"negative end":        {0, -1},
		"end before start":    {5, 2},
		"zero-width window":   {2, 2},
		"nan start":           {math.NaN(), 2},
		"infinite end":        {0, math.Inf(1)},
		"infinite start":      {math.Inf(1), 0},
		"negative both sides": {-2, -1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := audio.Trim(context.Background(), []byte("X"), "wav", tc.start, tc.end, "")
			require.Error(t, err)
			require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
		})
	}
}

func TestTrimOpenEndedHasNoDuration(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	_, err := audio.Trim(context.Background(), []byte("X"), "wav", 2, 0, "")
	require.NoError(t, err)
	args := fake.Calls[0].Args
	_, hasDuration := argValue(args, "-t")
	require.False(t, hasDuration, "end=0 means EOF; no -t expected: %v", args)
}

func TestFadeFadeOutAnchoredToDuration(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"10.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	_, err := audio.Fade(context.Background(), []byte("X"), "wav", 0, 0.5, "")
	require.NoError(t, err)
	require.Len(t, fake.Calls, 2, "fade-out should probe for duration first")

	filter, ok := argValue(fake.Calls[1].Args, "-af")
	require.True(t, ok)
	require.Contains(t, filter, "afade=t=out:st=9.5:d=0.5")
	require.NotContains(t, filter, "areverse")

	// The probe must have happened before the ffmpeg encode.
	require.Equal(t, "ffprobe", fake.Calls[0].Name)
	require.Equal(t, "ffmpeg", fake.Calls[1].Name)
}

func TestFadeFadeOutFallsBackWhenProbeUnavailable(t *testing.T) {
	fake := respondFake([]byte("OUT"), "", errors.New("ffprobe exploded"))
	swapRunner(t, fake)

	_, err := audio.Fade(context.Background(), []byte("X"), "wav", 0, 0.5, "")
	require.NoError(t, err)

	filter, ok := argValue(fake.Calls[len(fake.Calls)-1].Args, "-af")
	require.True(t, ok)
	// Without a duration the reverse/fade/reverse idiom still fades the
	// tail; st= anchored on t=0 would silence everything after 0.5s.
	require.Contains(t, filter, "areverse")
	require.NotContains(t, filter, "afade=t=out")
}

func TestFadeRejectsNegativeWindows(t *testing.T) {
	for _, tc := range []struct{ in, out float64 }{{-1, 0}, {0, -0.5}, {math.NaN(), 0}} {
		_, err := audio.Fade(context.Background(), []byte("X"), "wav", tc.in, tc.out, "")
		require.Error(t, err)
		require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
	}
}

func TestVolumeRejectsNonFiniteGain(t *testing.T) {
	for _, gain := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := audio.Volume(context.Background(), []byte("X"), "wav", gain, "")
		require.Error(t, err)
		require.True(t, errors.Is(err, audio.ErrInvalidArgument), "gain=%v err=%v", gain, err)
	}
	// A finite negative gain (attenuation) stays valid.
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.Volume(context.Background(), []byte("X"), "wav", -6, "")
	require.NoError(t, err)
}

func TestNormalizeRejectsNonFiniteTarget(t *testing.T) {
	for _, target := range []float64{math.NaN(), math.Inf(1)} {
		_, err := audio.Normalize(context.Background(), []byte("X"), "wav", "ebu_r128", target, "")
		require.Error(t, err)
		require.True(t, errors.Is(err, audio.ErrInvalidArgument), "target=%v err=%v", target, err)
	}
}

func TestSplitZeroWindowsReturnsWholeInput(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 0, nil, "wav")
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	require.Equal(t, 0.0, chunks[0].Start)
	require.Equal(t, 0.0, chunks[0].Duration, "last chunk uses duration=0 as EOF sentinel")
	require.Equal(t, "audio/wav", chunks[0].ContentType)
	require.Len(t, fake.Calls, 1, "zero windows must not probe")
}

func TestSplitRejectsNegativeChunkSeconds(t *testing.T) {
	for _, chunk := range []float64{-1, math.NaN(), math.Inf(1)} {
		_, err := audio.Split(context.Background(), []byte("X"), "wav", chunk, nil, "wav")
		require.Error(t, err)
		require.True(t, errors.Is(err, audio.ErrInvalidArgument), "chunk=%v err=%v", chunk, err)
	}
}

func TestSplitRejectsInvalidBoundaries(t *testing.T) {
	cases := map[string][]float64{
		"negative":       {-1, 2},
		"zero":           {0, 2},
		"descending":     {5, 2},
		"duplicate":      {2, 2},
		"nan":            {math.NaN()},
		"infinite":       {math.Inf(1)},
		"second is zero": {1, 0},
	}
	for name, boundaries := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := audio.Split(context.Background(), []byte("X"), "wav", 0, boundaries, "wav")
			require.Error(t, err)
			require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
		})
	}
}

func TestSplitChunkCutsFromProbeDuration(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"8000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"6.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 2.0, nil, "wav")
	require.NoError(t, err)
	require.Len(t, chunks, 3)
	require.Equal(t, []float64{0, 2, 4}, []float64{chunks[0].Start, chunks[1].Start, chunks[2].Start})
	require.Equal(t, []float64{2, 2, 2}, []float64{chunks[0].Duration, chunks[1].Duration, chunks[2].Duration})

	// Second chunk must be the absolute [2,4) window: input -ss 2 plus a
	// 2-second output duration.
	var secondTrim []string
	for _, c := range fake.Calls {
		if c.Name != "ffmpeg" {
			continue
		}
		if ss, ok := argValue(c.Args, "-ss"); ok && ss == "2" {
			secondTrim = c.Args
			break
		}
	}
	require.NotNil(t, secondTrim, "no ffmpeg call for the second chunk: %v", fake.Calls)
	require.True(t, hasSubsequence(secondTrim, "-ss", "2", "-i", "pipe:0", "-t", "2"), "%v", secondTrim)
}

func TestSplitUnknownDurationReturnsSingleChunk(t *testing.T) {
	// ffprobe could not determine duration ("N/A"): never loop on a bogus
	// duration; return the whole input as one chunk.
	const probe = `{"streams":[],"format":{"format_name":"wav","duration":"N/A"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 1.0, nil, "wav")
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	require.Equal(t, 0.0, chunks[0].Start)
}

func TestSplitCapsChunkCount(t *testing.T) {
	const probe = `{"streams":[],"format":{"duration":"100.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	_, err := audio.Split(context.Background(), []byte("X"), "wav", 0.001, nil, "wav")
	require.Error(t, err)
	require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
}

func TestSplitContextCanceled(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := audio.Split(ctx, []byte("X"), "wav", 1, nil, "wav")
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), "want context.Canceled, got %v", err)
}

// lastFfmpegCall returns the final ffmpeg invocation, skipping any ffprobe
// calls made by crossfade validation.
func lastFfmpegCall(t *testing.T, fake *mocks.FakeRunner) mocks.Call {
	t.Helper()
	for i := len(fake.Calls) - 1; i >= 0; i-- {
		if fake.Calls[i].Name == "ffmpeg" {
			return fake.Calls[i]
		}
	}
	t.Fatalf("no ffmpeg call recorded: %v", fake.Calls)
	return mocks.Call{}
}

func filterAndMap(t *testing.T, args []string) (filter, mapLabel string) {
	t.Helper()
	filter, ok := argValue(args, "-filter_complex")
	require.True(t, ok)
	mapLabel, ok = argValue(args, "-map")
	require.True(t, ok)
	return filter, mapLabel
}

// TestMergeCrossfadeMapsFinalLabel is the regression for the crossfade
// graph: its output labels are [a1]..[aN], so mapping a hard-coded "[a]"
// made every crossfade merge fail with an unmatched stream.
func TestTranscodeWithInputRejectsOutOfRangeKnobs(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	cases := map[string]struct {
		sampleRate, channels, bitrate int
	}{
		"negative sample rate": {sampleRate: -1},
		"huge sample rate":     {sampleRate: 768001},
		"negative channels":    {channels: -1},
		"too many channels":    {channels: 65},
		"negative bitrate":     {bitrate: -1},
		"huge bitrate":         {bitrate: 1_000_000_001},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := audio.TranscodeWithInput(context.Background(), []byte("X"), "", "wav", tc.sampleRate, tc.channels, tc.bitrate)
			require.Error(t, err)
			require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
		})
	}
	require.Empty(t, fake.Calls, "parameter validation must run before spawning ffmpeg")
}

func TestNormalizeRejectsOutOfRangeTarget(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)

	for _, target := range []float64{-71, -4.9, 100} {
		_, err := audio.Normalize(context.Background(), []byte("X"), "wav", "ebu_r128", target, "wav")
		require.Error(t, err, "target %g", target)
		require.True(t, errors.Is(err, audio.ErrInvalidArgument), "target %g: %v", target, err)
	}
	require.Empty(t, fake.Calls, "out-of-range loudness targets must be rejected before ffmpeg")
}

func TestMergeRejectsCrossfadeLongerThanShortestSource(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"1.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	_, err := audio.Merge(context.Background(), [][]byte{[]byte("A"), []byte("B")}, []string{"wav", "wav"}, "wav", 1.5)
	require.Error(t, err)
	require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
	for _, c := range fake.Calls {
		require.Equal(t, "ffprobe", c.Name, "crossfade validation must reject before ffmpeg writes an empty stream")
	}
}

func TestMergeAcceptsCrossfadeShorterThanSource(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"1.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	out, err := audio.Merge(context.Background(), [][]byte{[]byte("A"), []byte("B")}, []string{"wav", "wav"}, "wav", 0.5)
	require.NoError(t, err)
	require.Equal(t, []byte("OUT"), out)
}

func TestTransformsRejectEmptyFfmpegOutput(t *testing.T) {
	fake := mocks.NewFakeRunner(nil, nil)
	swapRunner(t, fake)

	_, err := audio.Volume(context.Background(), []byte("X"), "wav", 0, "wav")
	require.Error(t, err, "an exit-zero ffmpeg that writes nothing must not look like a successful transform")
	require.True(t, errors.Is(err, audio.ErrFfmpegExec), "want ErrFfmpegExec, got %v", err)
}

func TestTrimRejectsStartBeyondKnownDuration(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"2.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	_, err := audio.Trim(context.Background(), []byte("X"), "wav", 2.0, 0, "wav")
	require.Error(t, err)
	require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
	for _, c := range fake.Calls {
		require.Equal(t, "ffprobe", c.Name, "validation must reject before running ffmpeg")
	}
}

func TestTrimClampsEndToKnownDuration(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"2.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	_, err := audio.Trim(context.Background(), []byte("X"), "wav", 0.5, 10, "wav")
	require.NoError(t, err)
	args := lastFfmpegCall(t, fake).Args
	require.True(t, hasSubsequence(args, "-ss", "0.5", "-i", "pipe:0", "-t", "1.5"), "%v", args)
}

func TestSplitRejectsBoundaryBeyondKnownDuration(t *testing.T) {
	const probe = `{"streams":[{"sample_rate":"16000","channels":1,"codec_name":"pcm_s16le"}],"format":{"duration":"2.0","format_name":"wav"}}`
	fake := respondFake([]byte("OUT"), probe, nil)
	swapRunner(t, fake)

	_, err := audio.Split(context.Background(), []byte("X"), "wav", 0, []float64{1, 3}, "wav")
	require.Error(t, err)
	require.True(t, errors.Is(err, audio.ErrInvalidArgument), "want ErrInvalidArgument, got %v", err)
}
