package pipeline

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
)

// captureWhisper records the multipart upload's filename + body so the
// test can assert what the substrate sent to Whisper.
func captureWhisper(t *testing.T, gotFilename, gotBody *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(32 << 20)
		if f := r.MultipartForm; f != nil {
			if files := f.File["audio_file"]; len(files) > 0 {
				*gotFilename = files[0].Filename
				fh, _ := files[0].Open()
				b, _ := io.ReadAll(fh)
				*gotBody = string(b)
			}
		}
		_, _ = w.Write([]byte(`{"text":"transcribed"}`))
	}))
}

func TestTranscribeBytesPCMWrapsWAV(t *testing.T) {
	var filename, body string
	srv := captureWhisper(t, &filename, &body)
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	text, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "en", "")
	require.NoError(t, err)
	require.Equal(t, "transcribed", text)
	require.Equal(t, "recording.wav", filename)
	require.True(t, strings.HasPrefix(body, "RIFF"), "PCM must be wrapped as WAV before upload")
}

func TestTranscribeBytesContainerPassthrough(t *testing.T) {
	var filename, body string
	srv := captureWhisper(t, &filename, &body)
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	raw := "fake-webm-bytes"
	text, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte(raw), "webm", "", "")
	require.NoError(t, err)
	require.Equal(t, "transcribed", text)
	require.Equal(t, "recording.webm", filename)
	require.Equal(t, raw, body, "containers pass straight to Whisper's own decoder")
}

func TestTranscribeBytesUndeclaredUnknownErrors(t *testing.T) {
	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	_, err := TranscribeBytes(context.Background(), "http://unused", http.DefaultClient, engine,
		[]byte("not recognizable audio"), "", "", "")
	require.ErrorIs(t, err, audioformat.ErrUnknownFormat)
}
