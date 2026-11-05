package tts_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	ttsH "audio-tools/handlers/tts"
	"audio-tools/internal/audioformat"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

func TestTTS_GetSupportedFormats_ReportsFullVocabulary(t *testing.T) {
	c := newServer(t, ttsH.Deps{
		Engine: audioformat.New(audioformat.WithFfmpegProbe(func() bool { return true })),
	})
	resp, err := c.GetSupportedFormats(context.Background(), connect.NewRequest(&ttsv1.GetSupportedFormatsRequest{}))
	require.NoError(t, err)
	require.ElementsMatch(t, []commonv1.ResponseFormat{
		commonv1.ResponseFormat_RESPONSE_FORMAT_MP3,
		commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
		commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS,
		commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC,
	}, resp.Msg.GetEmittedFormats())
	require.True(t, resp.Msg.GetFfmpegAvailable())
}

// The synthesis engine encodes containers itself, so the emitted-format
// vocabulary does not shrink when the substrate's own ffmpeg is absent.
func TestTTS_GetSupportedFormats_FfmpegOffStillReportsVocabulary(t *testing.T) {
	c := newServer(t, ttsH.Deps{
		Engine: audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false })),
	})
	resp, err := c.GetSupportedFormats(context.Background(), connect.NewRequest(&ttsv1.GetSupportedFormatsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEmittedFormats(), 4)
	require.False(t, resp.Msg.GetFfmpegAvailable())
}

func TestTTS_GetSupportedFormats_NilEngineFallsBack(t *testing.T) {
	c := newServer(t, ttsH.Deps{})
	resp, err := c.GetSupportedFormats(context.Background(), connect.NewRequest(&ttsv1.GetSupportedFormatsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEmittedFormats(), 4)
}
