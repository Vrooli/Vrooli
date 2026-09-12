package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audio-tools/internal/ai/ttschain"
)

func TestOpenAITTSContract(t *testing.T) {
	a := NewOpenAITTS()
	if a.ID() != "openai-tts" || a.Model() != "tts-1" {
		t.Fatalf("identity wrong: %s/%s", a.ID(), a.Model())
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatalf("empty key must be unavailable")
	}
	if a.StreamingCapability() {
		t.Fatalf("unary only")
	}
}

func TestCanonicalToOpenAIVoice(t *testing.T) {
	cases := map[string]string{
		"voice.feminine.warm":     "shimmer",
		"voice.feminine.neutral":  "nova",
		"voice.masculine.warm":    "onyx",
		"voice.masculine.neutral": "echo",
		"voice.neutral.default":   "alloy",
		"unknown":                 "alloy",
	}
	for in, want := range cases {
		if got := canonicalToOpenAIVoice(in, nil); got != want {
			t.Errorf("canonicalToOpenAIVoice(%q) = %q, want %q", in, got, want)
		}
	}
	if got := canonicalToOpenAIVoice("voice.feminine.warm", map[string]string{"byok:openai-tts": "custom"}); got != "custom" {
		t.Fatalf("override should win, got %q", got)
	}
}

func TestClampSpeed(t *testing.T) {
	cases := map[float64]float64{
		0:   1.0,
		-1:  1.0,
		0.1: 0.25,
		1.0: 1.0,
		2.0: 2.0,
		5.0: 4.0,
	}
	for in, want := range cases {
		if got := clampSpeed(in); got != want {
			t.Errorf("clampSpeed(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestTTSContentType(t *testing.T) {
	if ttsContentType("mp3") != "audio/mpeg" {
		t.Fatal("mp3")
	}
	if ttsContentType("wav") != "audio/wav" {
		t.Fatal("wav")
	}
	if ttsContentType("unknown") != "application/octet-stream" {
		t.Fatal("fallback")
	}
}

func TestOpenAITTSSynthesizeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("AUDIO"))
	}))
	defer srv.Close()

	a := NewOpenAITTS()
	a.Endpoint = srv.URL
	res, err := a.Synthesize(context.Background(), "sk-test", ttschain.Request{
		Text: "hi", Voice: "voice.neutral.default", ResponseFormat: "mp3",
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if string(res.Audio) != "AUDIO" {
		t.Fatalf("audio: %q", res.Audio)
	}
	if res.ContentType != "audio/mpeg" {
		t.Fatalf("ct: %q", res.ContentType)
	}
}

func TestOpenAITTSSynthesizeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "paid", http.StatusPaymentRequired)
	}))
	defer srv.Close()
	a := NewOpenAITTS()
	a.Endpoint = srv.URL
	_, err := a.Synthesize(context.Background(), "k", ttschain.Request{Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "402") {
		t.Fatalf("expected 402, got %v", err)
	}
}
