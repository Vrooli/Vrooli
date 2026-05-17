package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audio-tools/internal/ai/sttchain"
)

func TestDeepgramContract(t *testing.T) {
	a := NewDeepgramSTT()
	if a.ID() != "deepgram" {
		t.Fatalf("id: %s", a.ID())
	}
	if a.Model() != "nova-2" {
		t.Fatalf("model: %s", a.Model())
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatalf("empty key unavailable")
	}
}

func TestDeepgramTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Token ") {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":{"channels":[{"alternatives":[{"transcript":"hello"}]}]}}`))
	}))
	defer srv.Close()

	a := NewDeepgramSTT()
	a.Endpoint = srv.URL
	res, err := a.Transcribe(context.Background(), "k", sttchain.Request{Audio: []byte("x"), Format: "wav"})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if res.Text != "hello" {
		t.Fatalf("text: %q", res.Text)
	}
}

func TestDeepgramTranscribeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "auth", http.StatusUnauthorized)
	}))
	defer srv.Close()
	a := NewDeepgramSTT()
	a.Endpoint = srv.URL
	_, err := a.Transcribe(context.Background(), "k", sttchain.Request{Audio: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401: %v", err)
	}
}

func TestDeepgramTranscribeMissingKey(t *testing.T) {
	a := NewDeepgramSTT()
	if _, err := a.Transcribe(context.Background(), "", sttchain.Request{}); err == nil {
		t.Fatalf("expected missing-key rejection")
	}
}
