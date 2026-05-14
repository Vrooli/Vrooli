package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	voicev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/voice"

	"web-console/internal/audio"
	"web-console/internal/capabilities"
	"web-console/internal/metrics"
	intvoice "web-console/internal/voice"
)

func serverWithCapability(available bool) *Server {
	status := capabilities.StatusUnavailable
	if available {
		status = capabilities.StatusAvailable
	}
	checker := &fakeChecker{status: status, message: "test"}
	reg := capabilities.NewRegistry(capabilities.Known, map[string]capabilities.Checker{"whisper-stt": checker}, time.Minute)
	m := metrics.New()
	srv := &Server{
		capabilities: reg,
		metrics:      m,
	}
	srv.voice = intvoice.NewService(
		intvoice.DefaultConfig(), "",
		nil, "",
		intvoice.DefaultSpeakerConfig(), "",
		nil,
		reg,
		&m.VoiceSkipVerificationTotal,
		intvoice.ResolveWhisperURL(),
		audio.Transcode,
	)
	return srv
}

func TestVoiceTranscribe_WhisperUnavailable(t *testing.T) {
	srv := serverWithCapability(false)
	_, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{Audio: []byte("x")})
	if connectCode(err) != connect.CodeUnavailable {
		t.Errorf("expected CodeUnavailable, got %v (err=%v)", connectCode(err), err)
	}
}

func TestVoiceTranscribe_MissingAudio(t *testing.T) {
	srv := serverWithCapability(true)
	_, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connectCode(err))
	}
}

func TestVoiceTranscribe_Success(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("whisper mock: parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _, err := r.FormFile("audio_file")
		if err != nil {
			t.Errorf("whisper mock: missing audio_file: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hello world"})
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })

	text, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{Audio: []byte("fake audio data")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
}

func TestVoiceTranscribe_WhisperError(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })

	_, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{Audio: []byte("fake audio data")})
	if connectCode(err) != connect.CodeInternal {
		t.Errorf("expected CodeInternal, got %v", connectCode(err))
	}
}

func TestVoiceTranscribe_LanguageParam(t *testing.T) {
	var receivedURL string
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hola mundo"})
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })

	if _, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{
		Audio:    []byte("fake audio"),
		Language: "es",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedURL, "language=es") {
		t.Errorf("Whisper URL %q should contain language=es", receivedURL)
	}
}

func TestVoiceTranscribe_AutoDetectLanguage(t *testing.T) {
	var receivedURL string
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hello"})
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })

	if _, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{Audio: []byte("fake audio")}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(receivedURL, "language=") {
		t.Errorf("Whisper URL %q should NOT contain language= for auto-detect", receivedURL)
	}
}

// TestVoiceTranscribe_SkipSpeakerVerificationBypassesFilter: a request with
// skip_speaker_verification=true skips the verification gate entirely. The
// metric counter increments exactly once per bypass.
func TestVoiceTranscribe_SkipSpeakerVerificationBypassesFilter(t *testing.T) {
	var whisperCalls int
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		whisperCalls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "retried transcript"})
	}))
	defer whisper.Close()

	// Speaker server should never be consulted with bypass active.
	speaker := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("speaker verification endpoint called with bypass active: %s", r.URL.Path)
	}))
	defer speaker.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })
	srv.voice.SetSpeakerConfig(intvoice.SpeakerConfig{
		Enabled: true, ProfileIDs: []string{"default"}, Threshold: 0.85,
		Mode: "filter", RejectBehavior: "drop",
	})
	srv.voice.SetSpeakerClient(&intvoice.SpeakerClient{
		BaseURL: speaker.URL,
		Client:  speaker.Client(),
	})

	text, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{
		Audio:                   []byte("fake audio data"),
		SkipSpeakerVerification: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "retried transcript" {
		t.Fatalf("text = %q, want %q", text, "retried transcript")
	}
	if whisperCalls != 1 {
		t.Fatalf("whisper calls = %d, want 1", whisperCalls)
	}
	if got := srv.metrics.VoiceSkipVerificationTotal.Load(); got != 1 {
		t.Fatalf("VoiceSkipVerificationTotal = %d, want 1", got)
	}
}

// TestVoiceTranscribe_SkipVerificationCounterOnlyWhenBypassActive confirms
// the metric does not drift when bypass is not set.
func TestVoiceTranscribe_SkipVerificationCounterOnlyWhenBypassActive(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hello"})
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })
	if _, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{Audio: []byte("fake")}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := srv.metrics.VoiceSkipVerificationTotal.Load(); got != 0 {
		t.Fatalf("VoiceSkipVerificationTotal = %d, want 0", got)
	}
}

func TestVoiceTranscribe_SpeakerVerificationRejectsAudio(t *testing.T) {
	var whisperCalls int
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		whisperCalls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "should not be used"})
	}))
	defer whisper.Close()

	speaker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/verify" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse verify form: %v", err)
		}
		writeJSON(w, http.StatusOK, intvoice.SpeakerVerifyResult{
			ProfileID:    "default",
			Matched:      false,
			Score:        0.42,
			Threshold:    0.85,
			DurationMs:   12.5,
			Backend:      "nemo-titanet",
			Model:        "titanet_large",
			AudioSeconds: 1.8,
		})
	}))
	defer speaker.Close()

	srv := serverWithCapability(true)
	srv.voice.SetWhisperURL(whisper.URL + "/asr?output=json")
	srv.voice.SetTranscode(func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil })
	srv.voice.SetSpeakerConfig(intvoice.SpeakerConfig{
		Enabled:                     true,
		ProfileIDs:                  []string{"default"},
		Threshold:                   0.85,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
	})
	srv.voice.SetSpeakerClient(&intvoice.SpeakerClient{
		BaseURL: speaker.URL,
		Client:  speaker.Client(),
	})

	text, err := callVoiceTranscribe(t, srv, &voicev1.TranscribeRequest{Audio: []byte("fake audio data")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Fatalf("text = %q, want empty string", text)
	}
	if whisperCalls != 0 {
		t.Fatalf("whisper calls = %d, want 0", whisperCalls)
	}
}
