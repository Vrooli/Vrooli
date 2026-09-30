package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/testutil/mocks"
)

// containsEntry reports whether any recorded log line contains substr.
func containsEntry(entries []string, substr string) bool {
	for _, e := range entries {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

// TestTranscribeBytes_LogsEntryAndResponse asserts the whisper call path
// emits the structured request/response breadcrumbs so a hang or empty
// result is diagnosable from the logs.
func TestTranscribeBytes_LogsEntryAndResponse(t *testing.T) {
	fake := mocks.NewFakeLogger()
	prev := SetPackageLogger(fake)
	t.Cleanup(func() { SetPackageLogger(prev) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"text":"hello there"}`))
	}))
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	_, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "en", "", false)
	require.NoError(t, err)

	entries := fake.Entries()
	require.True(t, containsEntry(entries, "event=whisper_request"), "missing request log: %v", entries)
	require.True(t, containsEntry(entries, "event=whisper_response"), "missing response log: %v", entries)
	require.True(t, containsEntry(entries, "empty=false"), "non-empty transcript should log empty=false: %v", entries)
}

// TestTranscribeBytes_LogsEmptyTranscript asserts the explicit empty=true
// flag fires when Whisper returns a 200 OK with a whitespace-only transcript
// — the exact silent-loss failure this logging exists to surface.
func TestTranscribeBytes_LogsEmptyTranscript(t *testing.T) {
	fake := mocks.NewFakeLogger()
	prev := SetPackageLogger(fake)
	t.Cleanup(func() { SetPackageLogger(prev) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"text":"   "}`))
	}))
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	tr, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "", "", false)
	require.NoError(t, err)
	require.Equal(t, "   ", tr.Text)

	require.True(t, containsEntry(fake.Entries(), "empty=true"),
		"whitespace-only transcript must log empty=true: %v", fake.Entries())
}

// TestTranscribeBytes_LogsNon200 asserts a non-200 Whisper response is logged
// with context before the error is returned.
func TestTranscribeBytes_LogsNon200(t *testing.T) {
	fake := mocks.NewFakeLogger()
	prev := SetPackageLogger(fake)
	t.Cleanup(func() { SetPackageLogger(prev) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	_, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "", "", false)
	require.Error(t, err)
	require.True(t, containsEntry(fake.Entries(), "event=whisper_error reason=non_200"),
		"non-200 must log an error breadcrumb: %v", fake.Entries())
}
