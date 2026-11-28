package audio

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	intaudio "audio-tools/internal/audio"
	audiomocks "audio-tools/internal/audio/mocks"
	"audio-tools/internal/testutil/mocks"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// TestMerge_RejectsEmptySourceInList: the proto marks each source's audio
// non-empty, so a zero-length source is a client mistake, not a 502 from
// ffmpeg.
func TestMerge_RejectsEmptySourceInList(t *testing.T) {
	withRunner(t, audiomocks.NewFakeRunner([]byte("OUT"), nil))
	c := newClient(t)
	_, err := c.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{
		Sources: []*audiov1.MergeSource{
			{Audio: []byte("A"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
			{Audio: nil, Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
		},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestMerge_MultiInputArgv(t *testing.T) {
	fake := audiomocks.NewFakeRunner([]byte("MIX"), nil)
	withRunner(t, fake)
	c := newClient(t)
	_, err := c.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{
		Sources: []*audiov1.MergeSource{
			{Audio: []byte("A"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
			{Audio: []byte("B"), Format: commonv1.AudioFormat_AUDIO_FORMAT_MP3},
			{Audio: []byte("C"), Format: commonv1.AudioFormat_AUDIO_FORMAT_FLAC},
		},
		OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_OGG,
	}))
	require.NoError(t, err)
	// One ffmpeg invocation per request; the post-transform duration probe
	// is a separate ffprobe call.
	ffmpegCalls := 0
	for _, call := range fake.Calls {
		if call.Name == "ffmpeg" {
			ffmpegCalls++
		}
	}
	require.Equal(t, 1, ffmpegCalls)
	args := fake.Calls[0].Args
	inputs := 0
	for _, a := range args {
		if a == "-i" {
			inputs++
		}
	}
	require.Equal(t, 3, inputs, "each source must be its own ffmpeg input: %v", args)
}

func TestSplit_InvalidWindowsMapToInvalidArgument(t *testing.T) {
	withRunner(t, audiomocks.NewFakeRunner([]byte("OUT"), nil))
	c := newClient(t)

	t.Run("negative chunk", func(t *testing.T) {
		_, err := c.Split(context.Background(), connect.NewRequest(&audiov1.SplitRequest{
			Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, ChunkSeconds: -1,
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("descending boundaries", func(t *testing.T) {
		_, err := c.Split(context.Background(), connect.NewRequest(&audiov1.SplitRequest{
			Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
			BoundariesSeconds: []float64{5, 2},
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("negative boundary", func(t *testing.T) {
		_, err := c.Split(context.Background(), connect.NewRequest(&audiov1.SplitRequest{
			Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
			BoundariesSeconds: []float64{-1},
		}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}

func TestTrim_InvalidRangeMapsToInvalidArgument(t *testing.T) {
	withRunner(t, audiomocks.NewFakeRunner([]byte("OUT"), nil))
	c := newClient(t)
	_, err := c.Trim(context.Background(), connect.NewRequest(&audiov1.TrimRequest{
		Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, StartSeconds: 5, EndSeconds: 2,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestTranscode_ForwardsInputFormat: the proto declares input_format and
// the handler used to drop it. Opus is the interesting case because the
// demuxer is "ogg", not "opus".
func TestTranscode_ForwardsInputFormat(t *testing.T) {
	fake := audiomocks.NewFakeRunner([]byte("OUT"), nil)
	withRunner(t, fake)
	c := newClient(t)
	res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio:        []byte("X"),
		InputFormat:  commonv1.AudioFormat_AUDIO_FORMAT_OPUS,
		OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_MP3,
	}))
	require.NoError(t, err)
	require.Equal(t, "audio/mpeg", res.Msg.GetContentType())

	args := fake.Calls[0].Args
	i := -1
	for idx, a := range args {
		if a == "-i" {
			i = idx
			break
		}
	}
	require.Greater(t, i, 1, "missing -i: %v", args)
	require.Equal(t, "-f", args[i-2])
	require.Equal(t, "ogg", args[i-1])
}

func TestTranscode_FfmpegMissingMapsToFailedPrecondition(t *testing.T) {
	restore := intaudio.SetFfmpegAvailableForTest(false, false)
	t.Cleanup(restore)
	c := newClient(t)
	_, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: []byte("X"), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestExtractMetadata_FfprobeMissingMapsToFailedPrecondition(t *testing.T) {
	restore := intaudio.SetFfmpegAvailableForTest(true, false)
	t.Cleanup(restore)
	c := newClient(t)
	_, err := c.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{
		Audio: []byte("X"),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestExtractMetadata_NoTags(t *testing.T) {
	const probe = `{"streams":[{"codec_name":"pcm_s16le","sample_rate":"16000","channels":1}],"format":{"format_name":"wav","duration":"1.0"}}`
	withRunner(t, audiomocks.NewFakeRunner([]byte(probe), nil))
	c := newClient(t)
	res, err := c.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{
		Audio: []byte("X"),
	}))
	require.NoError(t, err)
	require.NotNil(t, res.Msg.GetMetadata())
	require.Empty(t, res.Msg.GetMetadata().GetTags())
	require.Equal(t, int32(16000), res.Msg.GetMetadata().GetSampleRate())
}

// TestExtractMetadata_AliasFormatName: ffprobe reports WebM as
// "matroska,webm"; the response should still carry AUDIO_FORMAT_WEBM.
func TestExtractMetadata_AliasFormatName(t *testing.T) {
	const probe = `{"streams":[{"codec_name":"opus","sample_rate":"48000","channels":2}],"format":{"format_name":"matroska,webm","duration":"2.0"}}`
	withRunner(t, audiomocks.NewFakeRunner([]byte(probe), nil))
	c := newClient(t)
	res, err := c.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{
		Audio: []byte("X"),
	}))
	require.NoError(t, err)
	require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_WEBM, res.Msg.GetMetadata().GetFormat())
}

func TestAudioFormatFromString_AliasLists(t *testing.T) {
	cases := map[string]commonv1.AudioFormat{
		"wav":                       commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		"matroska,webm":             commonv1.AudioFormat_AUDIO_FORMAT_WEBM,
		"ogg":                       commonv1.AudioFormat_AUDIO_FORMAT_OGG,
		"mov,mp4,m4a,3gp,3g2,mj2":   commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED,
		"":                          commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED,
		"totally-unknown-container": commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED,
		"wav,wrong-order-alias":     commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}
	for in, want := range cases {
		require.Equal(t, want, audioFormatFromString(in), "format_name=%q", in)
	}
}

// TestCanceledContextMapsToCanceled: the request was canceled, so Connect
// must report Canceled rather than blaming the payload with
// InvalidArgument.
func TestCanceledContextMapsToCanceled(t *testing.T) {
	withRunner(t, audiomocks.NewFakeRunner([]byte("OUT"), nil))
	h := NewConnectHandler(Deps{Logger: &mocks.FakeLogger{}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := h.Transcode(ctx, connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: []byte("X"), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeCanceled, connect.CodeOf(err))

	_, err = h.ExtractMetadata(ctx, connect.NewRequest(&audiov1.ExtractMetadataRequest{Audio: []byte("X")}))
	require.Error(t, err)
	require.Equal(t, connect.CodeCanceled, connect.CodeOf(err))
}

func TestExpiredDeadlineMapsToDeadlineExceeded(t *testing.T) {
	withRunner(t, audiomocks.NewFakeRunner([]byte("OUT"), nil))
	h := NewConnectHandler(Deps{Logger: &mocks.FakeLogger{}})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	t.Cleanup(cancel)

	_, err := h.Transcode(ctx, connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: []byte("X"), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeDeadlineExceeded, connect.CodeOf(err))
}

func TestMapAudioErr_ContextAndInvalidArgument(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"canceled", context.Canceled, connect.CodeCanceled},
		{"wrapped canceled", errors.Join(errors.New("runner aborted"), context.Canceled), connect.CodeCanceled},
		{"deadline", context.DeadlineExceeded, connect.CodeDeadlineExceeded},
		{"invalid argument", intaudio.ErrInvalidArgument, connect.CodeInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, connect.CodeOf(mapAudioErr(tc.err)))
		})
	}
}
