package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
	return &Server{capabilities: reg}
}

// bypassTranscode installs a no-op transcodeAudio for the duration of the test.
func bypassTranscode(t *testing.T) {
	t.Helper()
	orig := transcodeAudio
	transcodeAudio = func(_ context.Context, audio []byte) ([]byte, error) { return audio, nil }
	t.Cleanup(func() { transcodeAudio = orig })
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
	bypassTranscode(t)
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

	origURL := whisperURL
	whisperURL = whisper.URL + "/asr?output=json"
	defer func() { whisperURL = origURL }()

	srv := serverWithCapability(true)
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
	bypassTranscode(t)
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer whisper.Close()

	origURL := whisperURL
	whisperURL = whisper.URL + "/asr?output=json"
	defer func() { whisperURL = origURL }()

	srv := serverWithCapability(true)
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
	bypassTranscode(t)
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

	origURL := whisperURL
	whisperURL = whisper.URL + "/asr?output=json"
	defer func() { whisperURL = origURL }()

	srv := serverWithCapability(true)
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
	bypassTranscode(t)
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

	origURL := whisperURL
	whisperURL = whisper.URL + "/asr?output=json"
	defer func() { whisperURL = origURL }()

	srv := serverWithCapability(true)
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
