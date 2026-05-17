package capabilities

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResourceChecker_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := &ResourceChecker{URL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy")
	}
}

func TestResourceChecker_Redirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	checker := &ResourceChecker{
		URL:  srv.URL,
		Doer: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy")
	}
}

func TestResourceChecker_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := &ResourceChecker{URL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource returned unexpected status" {
		t.Errorf("message = %q, want %q", msg, "resource returned unexpected status")
	}
}

func TestResourceChecker_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &ResourceChecker{URL: url, Doer: &http.Client{Timeout: time.Second}}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
	}
}

func fakeWhisperServer(t *testing.T, asrStatus int, asrBody any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		if r.URL.Path == "/asr" {
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(asrStatus)
			if asrBody != nil {
				_ = json.NewEncoder(w).Encode(asrBody)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestWhisperChecker_HealthyAndTranscribes(t *testing.T) {
	srv := fakeWhisperServer(t, http.StatusOK, map[string]string{"text": ""})
	defer srv.Close()

	checker := &WhisperChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy and transcription verified" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy and transcription verified")
	}
}

func TestWhisperChecker_LiveButTranscriptionFails(t *testing.T) {
	srv := fakeWhisperServer(t, http.StatusInternalServerError, nil)
	defer srv.Close()

	checker := &WhisperChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "transcription endpoint returned non-200 status" {
		t.Errorf("message = %q, want %q", msg, "transcription endpoint returned non-200 status")
	}
}

func TestWhisperChecker_LiveButInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	checker := &WhisperChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "transcription response is not valid JSON" {
		t.Errorf("message = %q, want %q", msg, "transcription response is not valid JSON")
	}
}

func TestWhisperChecker_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &WhisperChecker{BaseURL: url, Doer: &http.Client{Timeout: time.Second}}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
	}
}

func fakeKokoroServer(t *testing.T, voicesStatus, speechStatus int, speechBody []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/audio/voices":
			w.WriteHeader(voicesStatus)
		case "/v1/audio/speech":
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(speechStatus)
			if speechBody != nil {
				_, _ = w.Write(speechBody)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestKokoroChecker_Available(t *testing.T) {
	srv := fakeKokoroServer(t, http.StatusOK, http.StatusOK, []byte("fake-audio-bytes"))
	defer srv.Close()

	checker := &KokoroChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "resource is healthy and synthesis verified" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy and synthesis verified")
	}
}

func TestKokoroChecker_VoicesDown(t *testing.T) {
	srv := fakeKokoroServer(t, http.StatusInternalServerError, http.StatusOK, nil)
	defer srv.Close()

	checker := &KokoroChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource returned unexpected status" {
		t.Errorf("message = %q, want %q", msg, "resource returned unexpected status")
	}
}

func TestKokoroChecker_SynthesisFails(t *testing.T) {
	srv := fakeKokoroServer(t, http.StatusOK, http.StatusInternalServerError, nil)
	defer srv.Close()

	checker := &KokoroChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "synthesis endpoint returned non-200 status" {
		t.Errorf("message = %q, want %q", msg, "synthesis endpoint returned non-200 status")
	}
}

func TestKokoroChecker_EmptyAudio(t *testing.T) {
	srv := fakeKokoroServer(t, http.StatusOK, http.StatusOK, []byte("ab"))
	defer srv.Close()

	checker := &KokoroChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "synthesis returned empty audio" {
		t.Errorf("message = %q, want %q", msg, "synthesis returned empty audio")
	}
}

func TestKokoroChecker_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &KokoroChecker{
		BaseURL: url, Doer: &http.Client{Timeout: time.Second},
		InspectState: func(context.Context, string) (bool, bool, error) {
			return true, true, nil
		},
	}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
	}
}

func TestKokoroChecker_NotInstalled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &KokoroChecker{
		BaseURL: url, Doer: &http.Client{Timeout: time.Second},
		InspectState: func(context.Context, string) (bool, bool, error) {
			return false, false, nil
		},
	}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not installed" {
		t.Errorf("message = %q, want %q", msg, "resource is not installed")
	}
}

func TestKokoroChecker_Stopped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &KokoroChecker{
		BaseURL: url, Doer: &http.Client{Timeout: time.Second},
		InspectState: func(context.Context, string) (bool, bool, error) {
			return true, false, nil
		},
	}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is stopped" {
		t.Errorf("message = %q, want %q", msg, "resource is stopped")
	}
}

func TestOllamaChecker_Available(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	checker := &OllamaChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "Ollama is running" {
		t.Errorf("message = %q, want %q", msg, "Ollama is running")
	}
}

func TestOllamaChecker_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := &OllamaChecker{BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "Ollama returned unexpected status" {
		t.Errorf("message = %q, want %q", msg, "Ollama returned unexpected status")
	}
}

func TestOllamaChecker_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &OllamaChecker{BaseURL: url, Doer: &http.Client{Timeout: time.Second}}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "Ollama is not responding" {
		t.Errorf("message = %q, want %q", msg, "Ollama is not responding")
	}
}

func TestOpenRouterChecker_NoAPIKey(t *testing.T) {
	checker := &OpenRouterChecker{APIKey: ""}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "OPENROUTER_API_KEY not configured" {
		t.Errorf("message = %q, want %q", msg, "OPENROUTER_API_KEY not configured")
	}
}

func TestOpenRouterChecker_ValidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/models" && r.Header.Get("Authorization") == "Bearer test-key" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	checker := &OpenRouterChecker{APIKey: "test-key", BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != "OpenRouter is configured and reachable" {
		t.Errorf("message = %q, want %q", msg, "OpenRouter is configured and reachable")
	}
}

func TestOpenRouterChecker_InvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	checker := &OpenRouterChecker{APIKey: "bad-key", BaseURL: srv.URL, Doer: srv.Client()}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "OpenRouter API key is invalid" {
		t.Errorf("message = %q, want %q", msg, "OpenRouter API key is invalid")
	}
}

func TestOpenRouterChecker_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	checker := &OpenRouterChecker{APIKey: "some-key", BaseURL: url, Doer: &http.Client{Timeout: time.Second}}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "OpenRouter is not reachable" {
		t.Errorf("message = %q, want %q", msg, "OpenRouter is not reachable")
	}
}

func TestGenerateSilentWAV(t *testing.T) {
	wav := generateSilentWAV()

	if len(wav) < 44 {
		t.Fatalf("WAV too short: %d bytes", len(wav))
	}
	if string(wav[:4]) != "RIFF" {
		t.Errorf("missing RIFF header")
	}
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("missing WAVE marker")
	}
	if string(wav[12:16]) != "fmt " {
		t.Errorf("missing fmt chunk")
	}
}
