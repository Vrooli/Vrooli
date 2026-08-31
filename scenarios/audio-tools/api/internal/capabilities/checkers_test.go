package capabilities

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

	"audio-tools/internal/byokstore"
)

func TestBYOKCredentialCheckerProbesConfiguredEndpointWithoutExposingSecret(t *testing.T) {
	var probed string
	checker := &BYOKCredentialChecker{
		ProviderID: "openai-tts", Capability: "tts",
		List: func(context.Context) ([]byokstore.Credential, error) {
			return []byokstore.Credential{{ProviderID: "openai-tts", Capability: "tts"}}, nil
		},
		Get:   func(context.Context, string, string) (string, bool, error) { return "secret-key", true, nil },
		Probe: func(_ context.Context, key string) error { probed = key; return nil },
	}
	status, message := checker.Check(context.Background())
	if status != StatusAvailable || message != "BYOK credential configured" || probed != "secret-key" {
		t.Fatalf("status=%q message=%q probed=%q", status, message, probed)
	}
}

func TestBYOKCredentialCheckerReportsEndpointFailureWithoutSecret(t *testing.T) {
	checker := &BYOKCredentialChecker{
		ProviderID: "openai-whisper", Capability: "stt",
		List: func(context.Context) ([]byokstore.Credential, error) {
			return []byokstore.Credential{{ProviderID: "openai-whisper", Capability: "stt"}}, nil
		},
		Get:   func(context.Context, string, string) (string, bool, error) { return "secret-key", true, nil },
		Probe: func(context.Context, string) error { return errors.New("network failure") },
	}
	status, message := checker.Check(context.Background())
	if status != StatusUnavailable || strings.Contains(message, "secret-key") || !strings.Contains(message, "endpoint") {
		t.Fatalf("status=%q message=%q", status, message)
	}
}

func TestBYOKCredentialCheckerHandlesStoreAndCredentialFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		list func(context.Context) ([]byokstore.Credential, error)
		get  func(context.Context, string, string) (string, bool, error)
		want Status
	}{
		{name: "missing store", want: StatusUnknown},
		{name: "store error", list: func(context.Context) ([]byokstore.Credential, error) { return nil, errors.New("store down") }, want: StatusUnknown},
		{name: "missing credential", list: func(context.Context) ([]byokstore.Credential, error) { return []byokstore.Credential{}, nil }, want: StatusUnavailable},
		{name: "secret unavailable", list: func(context.Context) ([]byokstore.Credential, error) {
			return []byokstore.Credential{{ProviderID: "p", Capability: "c"}}, nil
		}, get: func(context.Context, string, string) (string, bool, error) { return "", false, nil }, want: StatusUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &BYOKCredentialChecker{ProviderID: "p", Capability: "c", List: tt.list, Get: tt.get, Probe: func(context.Context, string) error { return nil }}
			status, _ := checker.Check(context.Background())
			if status != tt.want {
				t.Fatalf("status = %q, want %q", status, tt.want)
			}
		})
	}
}

type recordingDoer struct {
	method string
}

func (d *recordingDoer) Do(req *http.Request) (*http.Response, error) {
	d.method = req.Method
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestResourceChecker_UsesGETForLivenessProbe(t *testing.T) {
	doer := &recordingDoer{}
	status, _ := (&ResourceChecker{URL: "http://provider.invalid/health", Doer: doer}).Check(context.Background())
	if status != StatusAvailable {
		t.Fatalf("status = %q, want %q", status, StatusAvailable)
	}
	if doer.method != http.MethodGet {
		t.Fatalf("method = %q, want %q", doer.method, http.MethodGet)
	}
}

func TestResourceChecker_HTTPStatuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		redirect   bool
		wantStatus Status
		wantMsg    string
	}{
		{name: "healthy", statusCode: http.StatusOK, wantStatus: StatusAvailable, wantMsg: "resource is healthy"},
		{name: "redirect", statusCode: http.StatusTemporaryRedirect, redirect: true, wantStatus: StatusAvailable, wantMsg: "resource is healthy"},
		{name: "unavailable", statusCode: http.StatusInternalServerError, wantStatus: StatusUnavailable, wantMsg: "resource returned unexpected status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tt.statusCode) }))
			defer srv.Close()
			doer := srv.Client()
			if tt.redirect {
				doer = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			}
			status, message := (&ResourceChecker{URL: srv.URL, Doer: doer}).Check(context.Background())
			if status != tt.wantStatus || message != tt.wantMsg {
				t.Fatalf("Check() = (%q, %q), want (%q, %q)", status, message, tt.wantStatus, tt.wantMsg)
			}
		})
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

func TestProbeBYOKEndpointUsesProviderAuthentication(t *testing.T) {
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	var got *http.Request
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req.Clone(req.Context())
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	for _, provider := range []string{"openai-whisper", "openai-tts", "deepgram", "elevenlabs"} {
		got = nil
		if err := ProbeBYOKEndpoint(provider)(context.Background(), "secret"); err != nil {
			t.Fatalf("%s probe: %v", provider, err)
		}
		if got == nil || got.Method != http.MethodGet || got.Header.Get("Authorization") == "" && got.Header.Get("xi-api-key") == "" {
			t.Fatalf("%s request did not carry authentication: %+v", provider, got)
		}
	}
	if err := ProbeBYOKEndpoint("unknown")(context.Background(), "secret"); err == nil {
		t.Fatal("unknown provider unexpectedly probed")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestOllamaChecker_VerifiesConfiguredSummarizeModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "fixture-summarize-model"}},
		})
	}))
	defer srv.Close()

	checker := &OllamaChecker{BaseURL: srv.URL, Doer: srv.Client(), Model: "fixture-summarize-model"}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Errorf("status = %q, want %q", status, StatusAvailable)
	}
	if msg != `Ollama is running and summarize model "fixture-summarize-model" is available` {
		t.Errorf("message = %q", msg)
	}
}

func TestOllamaChecker_ModelMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "fixture-other-model"}},
		})
	}))
	defer srv.Close()

	// A concrete tag (carries a ":") that is absent stays on the physical
	// path and reports "not installed" without consulting role policy.
	checker := &OllamaChecker{BaseURL: srv.URL, Doer: srv.Client(), Model: "fixture-summarize-model:latest"}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != `Ollama is running but summarize model "fixture-summarize-model:latest" is not installed` {
		t.Errorf("message = %q", msg)
	}
}

// TestOllamaChecker_RoleResolvesToInstalledModel is the regression guard for the
// degraded-health bug: the configured selector is a logical role
// ("summarize.default"), which never appears verbatim in /api/tags. The checker
// must resolve it through the policy SSOT to the physical model and verify THAT
// is installed, rather than literally matching the role name and falsely
// reporting it uninstalled.
func TestOllamaChecker_RoleResolvesToInstalledModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "qwen3.5:9b"}},
		})
	}))
	defer srv.Close()

	checker := &OllamaChecker{
		BaseURL: srv.URL, Doer: srv.Client(),
		Model:       "summarize.default",
		ResolveRole: func(_ context.Context, role string) (string, error) { return "qwen3.5:9b", nil },
	}
	status, msg := checker.Check(context.Background())

	if status != StatusAvailable {
		t.Fatalf("status = %q, want %q (msg=%q)", status, StatusAvailable, msg)
	}
	if msg != `Ollama is running and summarize model "qwen3.5:9b" (role "summarize.default") is available` {
		t.Errorf("message = %q", msg)
	}
}

// TestOllamaChecker_RoleResolvesToMissingModel covers a genuinely degraded
// state: the role resolves to a model that is not pulled. The message names the
// resolved physical model so an operator knows exactly what to pull.
func TestOllamaChecker_RoleResolvesToMissingModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "qwen3.5:4b"}},
		})
	}))
	defer srv.Close()

	checker := &OllamaChecker{
		BaseURL: srv.URL, Doer: srv.Client(),
		Model:       "summarize.default",
		ResolveRole: func(_ context.Context, role string) (string, error) { return "qwen3.5:9b", nil },
	}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != `Ollama is running but summarize model "qwen3.5:9b" (role "summarize.default") is not installed` {
		t.Errorf("message = %q", msg)
	}
}

// TestOllamaChecker_RoleResolutionFails covers the policy SSOT being
// unreachable: the summarize capability genuinely cannot be planned, so the
// checker reports unavailable with a clear, role-named message.
func TestOllamaChecker_RoleResolutionFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "qwen3.5:9b"}},
		})
	}))
	defer srv.Close()

	checker := &OllamaChecker{
		BaseURL: srv.URL, Doer: srv.Client(),
		Model:       "summarize.default",
		ResolveRole: func(_ context.Context, role string) (string, error) { return "", errors.New("policy unavailable") },
	}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != `Ollama is running but summarize role "summarize.default" could not be resolved: policy unavailable` {
		t.Errorf("message = %q", msg)
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
	if msg != "resource is healthy and ASR readiness verified; transcript quality not assessed" {
		t.Errorf("message = %q, want %q", msg, "resource is healthy and ASR readiness verified; transcript quality not assessed")
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

	checker := &KokoroChecker{BaseURL: url, Doer: &http.Client{Timeout: time.Second}}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
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
