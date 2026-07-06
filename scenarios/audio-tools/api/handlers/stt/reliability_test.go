package stt

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
	"audio-tools/internal/logx"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/testutil/mocks"

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

// TestMultipartTranscribe_RejectsOversize asserts the multipart endpoint
// rejects an upload larger than MaxAudioSize with a 4xx before invoking the
// chain (the limit was previously defined but never enforced).
func TestMultipartTranscribe_RejectsOversize(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{})
	h := MultipartTranscribeHandler(Deps{Chain: chain})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("audio", "big.wav")
	require.NoError(t, err)
	_, err = part.Write(make([]byte, sttpipeline.MaxAudioSize+1))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcribe", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code,
		"oversize upload should be rejected with 413; body=%s", rec.Body.String())
	require.Contains(t, rec.Body.String(), "maximum size")
}

// TestConnectTranscribe_RejectsOversize asserts the Connect unary Transcribe
// handler enforces the same MaxAudioSize ceiling as the multipart path.
func TestConnectTranscribe_RejectsOversize(t *testing.T) {
	h := NewConnectHandler(Deps{
		Chain:  sttchain.NewChain(sttchain.Options{}),
		Logger: logx.Std{},
		Clock:  clock.System{},
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
