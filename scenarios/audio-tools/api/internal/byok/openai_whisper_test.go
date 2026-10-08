package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audio-tools/internal/ai/sttchain"
)

func TestOpenAIWhisperContract(t *testing.T) {
	a := NewOpenAIWhisperSTT()
	if a.ID() != "openai-whisper" {
		t.Fatalf("ID: %s", a.ID())
	}
	if a.Model() != "whisper-1" {
		t.Fatalf("Model: %s", a.Model())
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatalf("empty key must be unavailable")
	}
	if !a.IsAvailable(context.Background(), "sk-test") {
		t.Fatalf("non-empty key must be available")
	}
	if a.StreamingCapability() {
		t.Fatalf("whisper-1 has no streaming")
	}
}

func TestOpenAIWhisperTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"hello world"}`))
	}))
	defer srv.Close()

	a := NewOpenAIWhisperSTT()
	a.Endpoint = srv.URL

	res, err := a.Transcribe(context.Background(), "sk-test", sttchain.Request{
		Audio: []byte("audio-bytes"), Format: "wav", Language: "en",
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if res.Text != "hello world" {
		t.Fatalf("text: %q", res.Text)
	}
}

func TestOpenAIWhisperTranscribeMissingKey(t *testing.T) {
	a := NewOpenAIWhisperSTT()
	if _, err := a.Transcribe(context.Background(), "", sttchain.Request{}); err == nil {
		t.Fatalf("expected missing-key rejection")
	}
}

func TestOpenAIWhisperTranscribeUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	a := NewOpenAIWhisperSTT()
	a.Endpoint = srv.URL
	_, err := a.Transcribe(context.Background(), "sk-test", sttchain.Request{Audio: []byte("x"), Format: "wav"})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected 429 error, got %v", err)
	}
}
