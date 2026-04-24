package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func serverWithCapability(available bool) *Server {
	status := StatusUnavailable
	if available {
		status = StatusAvailable
	}
	checker := &fakeChecker{status: status, message: "test"}
	reg := NewCapabilityRegistry(knownCapabilities, map[string]StatusChecker{"whisper-stt": checker}, time.Minute)
	return &Server{
		capabilities:   reg,
		voiceConfig:    DefaultVoiceStreamConfig(),
		whisperURL:     resolveWhisperURL(),
		transcodeAudio: defaultTranscodeAudio,
		metrics:        NewMetrics(),
	}
}

func buildMultipartRequest(t *testing.T, fieldName, fileName string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	req := httptest.NewRequest("POST", "/api/v1/voice/transcribe", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestVoiceTranscribe_WhisperUnavailable(t *testing.T) {
	srv := serverWithCapability(false)
	req := httptest.NewRequest("POST", "/api/v1/voice/transcribe", nil)
	rr := httptest.NewRecorder()

	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "voice_unavailable" {
		t.Errorf("code = %q, want %q", resp.Code, "voice_unavailable")
	}
}

func TestVoiceTranscribe_MissingAudioFile(t *testing.T) {
	srv := serverWithCapability(true)

	// Send a multipart form without the audio_file field
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("other_field", "value")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/voice/transcribe", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "invalid_body" {
		t.Errorf("code = %q, want %q", resp.Code, "invalid_body")
	}
}

func TestVoiceTranscribe_Success(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it received multipart form with audio_file
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
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake audio data"))
	rr := httptest.NewRecorder()

	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["text"] != "hello world" {
		t.Errorf("text = %q, want %q", resp["text"], "hello world")
	}
}

func TestVoiceTranscribe_WhisperError(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake audio data"))
	rr := httptest.NewRecorder()

	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != "voice_transcribe_failed" {
		t.Errorf("code = %q, want %q", resp.Code, "voice_transcribe_failed")
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
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake audio"))
	req.URL.RawQuery = "language=es"
	rr := httptest.NewRecorder()

	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
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
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake audio"))
	// No language query param — should trigger auto-detect (no language= in Whisper URL)
	rr := httptest.NewRecorder()

	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if strings.Contains(receivedURL, "language=") {
		t.Errorf("Whisper URL %q should NOT contain language= for auto-detect", receivedURL)
	}
}

// TestVoiceTranscribe_SkipSpeakerVerificationBypassesFilter verifies that a
// request with the explicit `skip_speaker_verification=true` query parameter
// skips the verification gate entirely and proceeds to Whisper, returning
// the transcript that would otherwise have been suppressed. The metric
// counter must increment exactly once per bypass.
// DOC: docs/plans/stt-voice-filter-retry-implementation-plan.md §9.4
func TestVoiceTranscribe_SkipSpeakerVerificationBypassesFilter(t *testing.T) {
	var whisperCalls int
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		whisperCalls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "retried transcript"})
	}))
	defer whisper.Close()

	// Speaker-verification server that would reject if consulted. The test
	// asserts it is NEVER consulted when bypass is active — so we fail hard
	// if the handler calls it.
	speaker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("speaker verification endpoint called with bypass active: %s", r.URL.Path)
	}))
	defer speaker.Close()

	srv := serverWithCapability(true)
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	srv.speakerVerificationConfig = SpeakerVerificationConfig{
		Enabled:        true,
		ProfileIDs:     []string{"default"},
		Threshold:      0.85,
		Mode:           "filter",
		RejectBehavior: "drop",
	}
	srv.speakerVerification = &SpeakerVerificationResourceClient{
		BaseURL: speaker.URL,
		Client:  speaker.Client(),
	}

	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake audio data"))
	req.URL.RawQuery = "skip_speaker_verification=true"
	rr := httptest.NewRecorder()
	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["text"] != "retried transcript" {
		t.Fatalf("text = %q, want %q", resp["text"], "retried transcript")
	}
	if whisperCalls != 1 {
		t.Fatalf("whisper calls = %d, want 1", whisperCalls)
	}
	if got := srv.metrics.VoiceSkipVerificationTotal.Load(); got != 1 {
		t.Fatalf("VoiceSkipVerificationTotal = %d, want 1", got)
	}
}

// TestVoiceTranscribe_SkipSpeakerVerificationStrictTrueOnly confirms that
// only the literal string `"true"` activates the bypass — not `"1"`,
// `"yes"`, `"TRUE"`, or trailing whitespace. The default (filter-active)
// path must run for every other value so the parameter is typo-safe.
func TestVoiceTranscribe_SkipSpeakerVerificationStrictTrueOnly(t *testing.T) {
	cases := []string{"1", "yes", "TRUE", "true ", " true", "True", ""}
	for _, raw := range cases {
		t.Run("value="+raw, func(t *testing.T) {
			whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{"text": "should not be used"})
			}))
			defer whisper.Close()

			speaker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseMultipartForm(10 << 20); err != nil {
					t.Fatalf("parse verify form: %v", err)
				}
				writeJSON(w, http.StatusOK, SpeakerVerificationResult{
					ProfileID: "default",
					Matched:   false,
					Score:     0.12,
					Threshold: 0.85,
				})
			}))
			defer speaker.Close()

			srv := serverWithCapability(true)
			srv.whisperURL = whisper.URL + "/asr?output=json"
			srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
			srv.speakerVerificationConfig = SpeakerVerificationConfig{
				Enabled: true, ProfileIDs: []string{"default"}, Threshold: 0.85,
				Mode: "filter", RejectBehavior: "drop",
			}
			srv.speakerVerification = &SpeakerVerificationResourceClient{
				BaseURL: speaker.URL,
				Client:  speaker.Client(),
			}

			req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake"))
			vals := url.Values{}
			vals.Set("skip_speaker_verification", raw)
			req.URL.RawQuery = vals.Encode()
			rr := httptest.NewRecorder()
			srv.handleVoiceTranscribe(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (raw=%q)", rr.Code, raw)
			}
			var resp map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			// Verification should have run (rejected), so text is empty.
			if resp["text"] != "" {
				t.Fatalf("text = %q, want \"\" (raw=%q); filter should have rejected", resp["text"], raw)
			}
			if got := srv.metrics.VoiceSkipVerificationTotal.Load(); got != 0 {
				t.Fatalf("VoiceSkipVerificationTotal = %d, want 0 (raw=%q); bypass should not have activated", got, raw)
			}
		})
	}
}

// TestVoiceTranscribe_SkipVerificationCounterOnlyWhenBypassActive confirms
// the metric does not drift in the default path, even when the filter
// actually rejects audio. Only explicit bypass increments the counter.
func TestVoiceTranscribe_SkipVerificationCounterOnlyWhenBypassActive(t *testing.T) {
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hello"})
	}))
	defer whisper.Close()

	srv := serverWithCapability(true)
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	// Speaker verification disabled → no filter, no bypass, no counter.
	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake"))
	rr := httptest.NewRecorder()
	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := srv.metrics.VoiceSkipVerificationTotal.Load(); got != 0 {
		t.Fatalf("VoiceSkipVerificationTotal = %d, want 0", got)
	}
}

func TestVoiceTranscribe_SpeakerVerificationRejectsAudio(t *testing.T) {
	var whisperCalls int
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusOK, SpeakerVerificationResult{
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
	srv.whisperURL = whisper.URL + "/asr?output=json"
	srv.transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	srv.speakerVerificationConfig = SpeakerVerificationConfig{
		Enabled:                     true,
		ProfileIDs:                  []string{"default"},
		Threshold:                   0.85,
		Mode:                        "filter",
		RejectBehavior:              "drop",
		FallbackWithoutVerification: false,
	}
	srv.speakerVerification = &SpeakerVerificationResourceClient{
		BaseURL: speaker.URL,
		Client:  speaker.Client(),
	}

	req := buildMultipartRequest(t, "audio_file", "test.wav", []byte("fake audio data"))
	rr := httptest.NewRecorder()
	srv.handleVoiceTranscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["text"] != "" {
		t.Fatalf("text = %q, want empty string", resp["text"])
	}
	if whisperCalls != 0 {
		t.Fatalf("whisper calls = %d, want 0", whisperCalls)
	}
}
