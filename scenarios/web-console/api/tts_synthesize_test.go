package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockSynthesizer implements TTSSynthesizer for testing.
type mockSynthesizer struct {
	body        string
	contentType string
	err         error
}

func (m *mockSynthesizer) Synthesize(_ context.Context, _ SynthesizeRequest) (io.ReadCloser, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return io.NopCloser(strings.NewReader(m.body)), m.contentType, nil
}

func newSynthesizeTestServer(synth TTSSynthesizer, capAvailable bool) *Server {
	srv := newFakeTestServer()
	srv.ttsSynthesizer = synth
	srv.ttsConfig = DefaultTTSConfig()

	checker := &fakeChecker{status: StatusUnavailable, message: "down"}
	if capAvailable {
		checker.status = StatusAvailable
		checker.message = "ok"
	}
	srv.capabilities = NewCapabilityRegistry(
		[]CapabilityDef{{ID: "kokoro-tts", Name: "Kokoro TTS"}},
		map[string]StatusChecker{"kokoro-tts": checker},
		time.Minute,
	)
	return srv
}

func TestHandleTTSSynthesize_HappyPath(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{
		body:        "fake-audio-bytes",
		contentType: "audio/mpeg",
	}, true)

	body := strings.NewReader(`{"input":"Hello world","voice":"af_heart","response_format":"mp3"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "audio/mpeg" {
		t.Errorf("expected Content-Type audio/mpeg, got %s", ct)
	}
	if rec.Body.String() != "fake-audio-bytes" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandleTTSSynthesize_503WhenCapabilityUnavailable(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, false)

	body := strings.NewReader(`{"input":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSSynthesize_400OnEmptyInput(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, true)

	body := strings.NewReader(`{"input":"  ","voice":"af_heart"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSSynthesize_400OnInputTooLong(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, true)

	longInput := strings.Repeat("a", maxSynthesizeInputLength+1)
	body := strings.NewReader(`{"input":"` + longInput + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSSynthesize_502OnSynthesizerError(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{err: errors.New("backend down")}, true)

	body := strings.NewReader(`{"input":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSSynthesize_DefaultVoiceFromConfig(t *testing.T) {
	var capturedReq SynthesizeRequest
	synth := &capturingSynthesizer{
		body:        "audio",
		contentType: "audio/mpeg",
		captured:    &capturedReq,
	}
	srv := newSynthesizeTestServer(synth, true)
	srv.ttsConfig.KokoroVoice = "bf_emma"

	body := strings.NewReader(`{"input":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedReq.Voice != "bf_emma" {
		t.Errorf("expected voice bf_emma from config, got %s", capturedReq.Voice)
	}
}

func TestHandleTTSSynthesize_400OnInvalidFormat(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, true)

	body := strings.NewReader(`{"input":"Hello","response_format":"aac"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSSynthesize_NilSynthesizer(t *testing.T) {
	srv := newSynthesizeTestServer(nil, true)
	srv.ttsSynthesizer = nil

	body := strings.NewReader(`{"input":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	// Should return an error when synthesizer is nil
	if rec.Code == http.StatusOK {
		t.Fatal("expected error when synthesizer is nil")
	}
}

func TestHandleTTSSynthesize_SpeedClamped(t *testing.T) {
	var capturedReq SynthesizeRequest
	synth := &capturingSynthesizer{
		body:        "audio",
		contentType: "audio/mpeg",
		captured:    &capturedReq,
	}
	srv := newSynthesizeTestServer(synth, true)

	body := strings.NewReader(`{"input":"Hello","speed":100}`)
	req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleTTSSynthesize(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if capturedReq.Speed != 4.0 {
		t.Errorf("expected speed clamped to 4.0, got %f", capturedReq.Speed)
	}
}

func TestHandleTTSSynthesize_StructuredErrors(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		capAvailable bool
		expectedCode string
		expectedHTTP int
	}{
		{
			name:         "unavailable returns structured error",
			body:         `{"input":"Hello"}`,
			capAvailable: false,
			expectedCode: "tts_unavailable",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "empty input returns structured error",
			body:         `{"input":"  "}`,
			capAvailable: true,
			expectedCode: "tts_input_required",
			expectedHTTP: http.StatusBadRequest,
		},
		{
			name:         "invalid format returns structured error",
			body:         `{"input":"Hello","response_format":"aac"}`,
			capAvailable: true,
			expectedCode: "tts_invalid_format",
			expectedHTTP: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, tt.capAvailable)
			req := httptest.NewRequest("POST", "/api/v1/tts/synthesize", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.handleTTSSynthesize(rec, req)

			if rec.Code != tt.expectedHTTP {
				t.Fatalf("expected %d, got %d: %s", tt.expectedHTTP, rec.Code, rec.Body.String())
			}
			var errResp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("expected structured JSON error, got parse error: %v\nbody: %s", err, rec.Body.String())
			}
			if errResp.Code != tt.expectedCode {
				t.Errorf("expected error code %q, got %q", tt.expectedCode, errResp.Code)
			}
			if errResp.Category == "" {
				t.Error("expected non-empty category in structured error")
			}
			if errResp.Recovery == "" {
				t.Error("expected non-empty recovery in structured error")
			}
		})
	}
}

// capturingSynthesizer captures the request for inspection.
type capturingSynthesizer struct {
	body        string
	contentType string
	captured    *SynthesizeRequest
}

func (c *capturingSynthesizer) Synthesize(_ context.Context, req SynthesizeRequest) (io.ReadCloser, string, error) {
	*c.captured = req
	return io.NopCloser(strings.NewReader(c.body)), c.contentType, nil
}
