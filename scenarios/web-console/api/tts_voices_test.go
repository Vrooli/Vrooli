package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockVoiceLister implements TTSVoiceLister for testing.
type mockVoiceLister struct {
	voices []TTSVoice
	err    error
}

func (m *mockVoiceLister) ListVoices(_ context.Context) ([]TTSVoice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.voices, nil
}

func newVoicesTestServer(lister TTSVoiceLister, capAvailable bool) *Server {
	srv := newFakeTestServer()
	srv.ttsVoiceLister = lister

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

func TestHandleTTSVoices_HappyPath(t *testing.T) {
	srv := newVoicesTestServer(&mockVoiceLister{
		voices: []TTSVoice{
			{ID: "af_heart", Name: "af_heart"},
			{ID: "bf_emma", Name: "bf_emma"},
		},
	}, true)

	req := httptest.NewRequest("GET", "/api/v1/tts/voices", nil)
	rec := httptest.NewRecorder()
	srv.handleTTSVoices(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	// Verify the body contains voice data
	body := rec.Body.String()
	if body == "" || body == "null\n" {
		t.Error("expected non-empty voice list")
	}
}

func TestHandleTTSVoices_503WhenCapabilityUnavailable(t *testing.T) {
	srv := newVoicesTestServer(&mockVoiceLister{
		voices: []TTSVoice{{ID: "af_heart", Name: "af_heart"}},
	}, false)

	req := httptest.NewRequest("GET", "/api/v1/tts/voices", nil)
	rec := httptest.NewRecorder()
	srv.handleTTSVoices(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSVoices_502OnListerError(t *testing.T) {
	srv := newVoicesTestServer(&mockVoiceLister{
		err: errors.New("kokoro down"),
	}, true)

	req := httptest.NewRequest("GET", "/api/v1/tts/voices", nil)
	rec := httptest.NewRecorder()
	srv.handleTTSVoices(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSVoices_NilLister(t *testing.T) {
	srv := newVoicesTestServer(nil, true)
	srv.ttsVoiceLister = nil

	req := httptest.NewRequest("GET", "/api/v1/tts/voices", nil)
	rec := httptest.NewRecorder()
	srv.handleTTSVoices(rec, req)

	// Should return an error when lister is nil
	if rec.Code == http.StatusOK {
		t.Fatal("expected error when voice lister is nil")
	}
}
