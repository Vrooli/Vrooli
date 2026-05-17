package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

type proxyResolverStub struct {
	url string
	err error
}

func (s proxyResolverStub) Resolve() (string, error) {
	return s.url, s.err
}

func TestAudioToolsProxy_ForwardsConnectProcedureSameOrigin(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ttsconnect.TTSServiceListVoicesProcedure {
			t.Fatalf("path = %q, want %q", r.URL.Path, ttsconnect.TTSServiceListVoicesProcedure)
		}
		if r.Host != r.URL.Host && r.Host == "" {
			t.Fatalf("expected upstream host to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodPost, ttsconnect.TTSServiceListVoicesProcedure, nil)
	rec := httptest.NewRecorder()

	newAudioToolsProxy(proxyResolverStub{url: upstream.URL}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestAudioToolsProxy_ReturnsUnavailableWhenResolverFails(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, ttsconnect.TTSServiceListVoicesProcedure, nil)
	rec := httptest.NewRecorder()

	newAudioToolsProxy(proxyResolverStub{err: errors.New("scenario not running")}).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestBuildUpstreamWS_PreservesQuery(t *testing.T) {
	got, err := buildUpstreamWS("http://localhost:15000/base/", "language=en")
	if err != nil {
		t.Fatalf("buildUpstreamWS returned error: %v", err)
	}
	want := "ws://localhost:15000/base/api/v1/voice/stream?language=en"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}
