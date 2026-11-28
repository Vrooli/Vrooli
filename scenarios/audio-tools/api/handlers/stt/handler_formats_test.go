package stt

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestGetSupportedFormats_ReportsFullVocabularyAndFfmpegOn(t *testing.T) {
	c := newSTTRuntimeClient(t, Deps{
		Engine: audioformat.New(audioformat.WithFfmpegProbe(func() bool { return true })),
	})
	resp, err := c.GetSupportedFormats(context.Background(), connect.NewRequest(&sttv1.GetSupportedFormatsRequest{}))
	require.NoError(t, err)
	require.ElementsMatch(t, []commonv1.AudioFormat{
		commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE,
		commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		commonv1.AudioFormat_AUDIO_FORMAT_MP3,
		commonv1.AudioFormat_AUDIO_FORMAT_FLAC,
		commonv1.AudioFormat_AUDIO_FORMAT_OGG,
		commonv1.AudioFormat_AUDIO_FORMAT_WEBM,
		commonv1.AudioFormat_AUDIO_FORMAT_OPUS,
		commonv1.AudioFormat_AUDIO_FORMAT_AAC,
	}, resp.Msg.GetAcceptedFormats())
	require.True(t, resp.Msg.GetFfmpegAvailable())
	require.Equal(t, int32(audioformat.CanonicalSampleRate), resp.Msg.GetCanonicalSampleRateHz())
	require.Equal(t, int32(audioformat.CanonicalChannels), resp.Msg.GetCanonicalChannels())
}

// The accepted-format vocabulary is independent of ffmpeg (batch decodes
// containers via Whisper); only the ffmpeg flag flips.
func TestGetSupportedFormats_FfmpegOffStillReportsVocabulary(t *testing.T) {
	c := newSTTRuntimeClient(t, Deps{
		Engine: audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false })),
	})
	resp, err := c.GetSupportedFormats(context.Background(), connect.NewRequest(&sttv1.GetSupportedFormatsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetAcceptedFormats(), 8)
	require.False(t, resp.Msg.GetFfmpegAvailable())
}

// A nil Engine must not panic — the handler falls back to a default engine.
func TestGetSupportedFormats_NilEngineFallsBack(t *testing.T) {
	c := newSTTRuntimeClient(t, Deps{})
	resp, err := c.GetSupportedFormats(context.Background(), connect.NewRequest(&sttv1.GetSupportedFormatsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetAcceptedFormats(), 8)
}
