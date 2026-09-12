package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audio-tools/internal/ai/ttschain"
)

func TestElevenLabsContract(t *testing.T) {
	a := NewElevenLabsTTS()
	if a.ID() != "elevenlabs" {
		t.Fatalf("id: %s", a.ID())
	}
	if a.Model() != "eleven_multilingual_v2" {
		t.Fatalf("model: %s", a.Model())
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatalf("empty key unavailable")
	}
}

func TestCanonicalToElevenVoiceIDOverride(t *testing.T) {
	if got := canonicalToElevenVoiceID("voice.feminine.warm", map[string]string{"byok:elevenlabs": "custom"}); got != "custom" {
		t.Fatalf("override wrong: %s", got)
	}
	if got := canonicalToElevenVoiceID("unknown", nil); got == "" {
		t.Fatalf("expected fallback voice id")
	}
}

func TestElevenLabsSynthesizeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "k" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("MP3"))
	}))
	defer srv.Close()

	a := NewElevenLabsTTS()
	a.BaseURL = srv.URL
	res, err := a.Synthesize(context.Background(), "k", ttschain.Request{Text: "hi", Voice: "voice.neutral.default"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if string(res.Audio) != "MP3" {
		t.Fatalf("audio: %q", res.Audio)
	}
	if res.ContentType != "audio/mpeg" {
		t.Fatalf("ct: %s", res.ContentType)
	}
}

func TestElevenLabsSynthesizeMissingKey(t *testing.T) {
	a := NewElevenLabsTTS()
	if _, err := a.Synthesize(context.Background(), "", ttschain.Request{Text: "x"}); err == nil {
		t.Fatalf("expected missing-key rejection")
	}
}

func TestElevenLabsSynthesizeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "too many", http.StatusTooManyRequests)
	}))
	defer srv.Close()
	a := NewElevenLabsTTS()
	a.BaseURL = srv.URL
	_, err := a.Synthesize(context.Background(), "k", ttschain.Request{Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected 429: %v", err)
	}
}
