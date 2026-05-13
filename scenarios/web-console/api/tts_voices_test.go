package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
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

	voices, err := callTTSVoices(t, srv)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(voices) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(voices))
	}
}

func TestHandleTTSVoices_503WhenCapabilityUnavailable(t *testing.T) {
	srv := newVoicesTestServer(&mockVoiceLister{
		voices: []TTSVoice{{ID: "af_heart", Name: "af_heart"}},
	}, false)

	_, err := callTTSVoices(t, srv)
	if connectCode(err) != connect.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSVoices_OnListerError(t *testing.T) {
	srv := newVoicesTestServer(&mockVoiceLister{
		err: errors.New("kokoro down"),
	}, true)

	_, err := callTTSVoices(t, srv)
	if connectCode(err) != connect.CodeInternal {
		t.Fatalf("expected CodeInternal, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSVoices_NilLister(t *testing.T) {
	srv := newVoicesTestServer(nil, true)
	srv.ttsVoiceLister = nil

	_, err := callTTSVoices(t, srv)
	if connectCode(err) == connect.CodeUnknown {
		t.Fatalf("expected a Connect error, got %v", err)
	}
}
