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
	tr, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "en", "", false)
	require.NoError(t, err)
	require.Equal(t, "transcribed", tr.Text)
	require.Equal(t, "recording.wav", filename)
	require.True(t, strings.HasPrefix(body, "RIFF"), "PCM must be wrapped as WAV before upload")
}

func TestTranscribeBytesContainerPassthrough(t *testing.T) {
	var filename, body string
	srv := captureWhisper(t, &filename, &body)
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	raw := "fake-webm-bytes"
	tr, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte(raw), "webm", "", "", false)
	require.NoError(t, err)
	require.Equal(t, "transcribed", tr.Text)
	require.Equal(t, "recording.webm", filename)
	require.Equal(t, raw, body, "containers pass straight to Whisper's own decoder")
}

func TestTranscribeBytesUndeclaredUnknownErrors(t *testing.T) {
	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	_, err := TranscribeBytes(context.Background(), "http://unused", http.DefaultClient, engine,
		[]byte("not recognizable audio"), "", "", "", false)
	require.ErrorIs(t, err, audioformat.ErrUnknownFormat)
}

// TestTranscribeBytesParsesConfidenceAndVADFilter asserts the /asr response
// segments are folded into TranscriptionResult confidence, and that the
// vad_filter flag reaches the request query when enabled.
func TestTranscribeBytesParsesConfidenceAndVADFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"text":"hello","segments":[
			{"no_speech_prob":0.2,"avg_logprob":-0.4},
			{"no_speech_prob":0.4,"avg_logprob":-0.8}]}`))
	}))
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	tr, err := TranscribeBytes(context.Background(), srv.URL+"/asr?output=json", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "en", "", true)
	require.NoError(t, err)
	require.Equal(t, "hello", tr.Text)
	require.True(t, tr.HasConfidence)
	require.InDelta(t, 0.3, tr.NoSpeechProb, 1e-9, "mean no_speech_prob")
	require.InDelta(t, -0.6, tr.AvgLogProb, 1e-9, "mean avg_logprob")
	require.Contains(t, gotQuery, "vad_filter=true")
}

// TestTranscribeBytesNoSegmentsNoConfidence asserts a response without a
// segments array yields HasConfidence=false so the signal stage is skipped.
func TestTranscribeBytesNoSegmentsNoConfidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"text":"hello"}`))
	}))
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	tr, err := TranscribeBytes(context.Background(), srv.URL+"/asr", srv.Client(), engine,
		[]byte{0x01, 0x00, 0x02, 0x00}, "pcm_s16le", "", "", false)
	require.NoError(t, err)
	require.False(t, tr.HasConfidence)
}
