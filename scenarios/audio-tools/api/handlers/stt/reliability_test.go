package stt

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/testutil/mocks"

	"github.com/vrooli/api-core/schedule"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func anyEntryContains(entries []string, substr string) bool {
	for _, e := range entries {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

// TestConnectTranscribe_RejectsOversize asserts the Connect unary Transcribe
// handler rejects audio above the shared ingress size ceiling.
func TestConnectTranscribe_RejectsOversize(t *testing.T) {
	h := NewConnectHandler(Deps{
		Chain:  sttchain.NewChain(sttchain.Options{}),
		Logger: logx.Std{},
		Clock:  schedule.System(),
	})

	_, err := h.Transcribe(context.Background(), connect.NewRequest(&sttv1.TranscribeRequest{
		Audio: make([]byte, sttpipeline.MaxAudioSize+1),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestLogPostDoneStreamError_LogsRealError asserts a late Segmenter/Selector
// error (surfacing after a Done event) is logged rather than silently
// swallowed.
func TestLogPostDoneStreamError_LogsRealError(t *testing.T) {
	fake := mocks.NewFakeLogger()
	logPostDoneStreamError(fake, "en", sttchain.ErrAllProvidersFailed)

	require.True(t, anyEntryContains(fake.Entries(), "event=stt_stream_post_done_error"),
		"post-Done error must be logged: %v", fake.Entries())
	require.True(t, anyEntryContains(fake.Entries(), "language=\"en\""),
		"log should carry session language context: %v", fake.Entries())
}

// TestLogPostDoneStreamError_IgnoresNilAndCanceled asserts the helper stays
// silent for the no-error and normal-teardown cases so the logs aren't noisy.
func TestLogPostDoneStreamError_IgnoresNilAndCanceled(t *testing.T) {
	fake := mocks.NewFakeLogger()
	logPostDoneStreamError(fake, "en", nil)
	logPostDoneStreamError(fake, "en", context.Canceled)
	require.Empty(t, fake.Entries(), "nil and context.Canceled must not log")
}
