package capabilities

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	diagnosticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	healthstatusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
)

func TestAudioToolsChecker_ProjectsFeatureSpecificHealth(t *testing.T) {
	checker := &AudioToolsChecker{
		Scenario: ScenarioChecker{Slug: "audio-tools", Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"scenario":{"name":"audio-tools","status":"running"}}`), nil
		}},
		Features: []string{"voice-input", "voice-output"},
		ProviderHealth: func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error) {
			return &healthstatusv1.GetProviderHealthResponse{Capabilities: []*healthstatusv1.CapabilityHealth{
				{Capability: diagnosticsv1.Capability_CAPABILITY_STT, EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE},
				{Capability: diagnosticsv1.Capability_CAPABILITY_TTS, EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE},
			}}, nil
		},
	}
	result := checker.CheckResult(context.Background())
	if result.Status != StatusAvailable || result.FeatureStatus["voice-input"] != string(StatusAvailable) || result.FeatureStatus["voice-output"] != string(StatusUnavailable) {
		t.Fatalf("result = %+v", result)
	}
}

func TestAudioToolsChecker_HealthFailureIsUnknown(t *testing.T) {
	checker := &AudioToolsChecker{
		Scenario: ScenarioChecker{Slug: "audio-tools", Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"scenario":{"name":"audio-tools","status":"running"}}`), nil
		}},
		Features: []string{"voice-input"}, Timeout: time.Millisecond,
		ProviderHealth: func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error) {
			return nil, errors.New("unreachable")
		},
	}
	result := checker.CheckResult(context.Background())
	if result.Status != StatusAvailable || result.FeatureStatus["voice-input"] != string(StatusUnknown) {
		t.Fatalf("result = %+v", result)
	}
}

func TestAudioToolsChecker_EnrichesReachableDegradedScenario(t *testing.T) {
	checker := &AudioToolsChecker{
		Scenario: ScenarioChecker{Slug: "audio-tools", Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"scenario":{"name":"audio-tools","status":"running","health_status":"degraded","health_error":"tts provider down"}}`), nil
		}},
		Features: []string{"voice-input", "voice-output"},
		ProviderHealth: func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error) {
			return &healthstatusv1.GetProviderHealthResponse{Capabilities: []*healthstatusv1.CapabilityHealth{
				{Capability: diagnosticsv1.Capability_CAPABILITY_STT, EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE},
				{Capability: diagnosticsv1.Capability_CAPABILITY_TTS, EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE},
			}}, nil
		},
	}
	result := checker.CheckResult(context.Background())
	if result.Status != StatusUnavailable || result.ReasonCode != "scenario_degraded" || result.FeatureStatus["voice-input"] != string(StatusAvailable) || result.FeatureStatus["voice-output"] != string(StatusUnavailable) {
		t.Fatalf("result = %+v", result)
	}
}

func TestAudioToolsCheckerKeepsProviderFeaturesIndependent(t *testing.T) {
	checker := &AudioToolsChecker{
		Scenario: ScenarioChecker{Slug: "audio-tools", Run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`{"scenario":{"name":"audio-tools","status":"running"}}`), nil
		}},
		Features: []string{"voice-input", "voice-speaker-verification", "voice-enrollment"},
		ProviderHealth: func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error) {
			return &healthstatusv1.GetProviderHealthResponse{Capabilities: []*healthstatusv1.CapabilityHealth{
				{Capability: diagnosticsv1.Capability_CAPABILITY_STT, EffectiveState: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE, Providers: []*sharedv1.ProviderHealth{
					{ProviderId: "whisper-stt", State: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE},
					{ProviderId: "speaker-verification", State: sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE},
					{ProviderId: "browser-stt", State: sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE},
				}},
			}}, nil
		},
	}
	result := checker.CheckResult(context.Background())
	if result.FeatureStatus["voice-input"] != string(StatusAvailable) {
		t.Fatalf("voice-input = %q, want available", result.FeatureStatus["voice-input"])
	}
	if result.ProviderStatus["whisper-stt"] != string(StatusAvailable) || result.ProviderStatus["speaker-verification"] != string(StatusUnavailable) {
		t.Fatalf("provider statuses = %#v, want individual provider verdicts", result.ProviderStatus)
	}
	if _, ok := result.ProviderStatus["browser-stt"]; ok {
		t.Fatalf("client-owned browser provider leaked into server provider statuses: %#v", result.ProviderStatus)
	}
	if result.FeatureStatus["voice-speaker-verification"] != string(StatusUnavailable) || result.FeatureStatus["voice-enrollment"] != string(StatusUnavailable) {
		t.Fatalf("speaker features = %#v, want unavailable", result.FeatureStatus)
	}
	if result.FeatureStatus["voice-input"] != string(StatusAvailable) {
		t.Fatalf("browser provider should remain part of feature serviceability: %#v", result.FeatureStatus)
	}
	if result.FeatureReason["voice-speaker-verification"] == "" || result.FeatureOperatorCommand["voice-speaker-verification"] != "vrooli resource status sherpa-onnx --json" {
		t.Fatalf("speaker remediation = reason %q command %q", result.FeatureReason["voice-speaker-verification"], result.FeatureOperatorCommand["voice-speaker-verification"])
	}
}

func TestStaticChecker_ConfigurationAndProbeResults(t *testing.T) {
	for _, tc := range []struct {
		name    string
		checker *StaticChecker
		status  Status
		msg     string
	}{
		{name: "nil", checker: nil, status: StatusUnavailable, msg: "host capability probe is not configured"},
		{name: "missing probe", checker: &StaticChecker{}, status: StatusUnavailable, msg: "host capability probe is not configured"},
		{name: "available", checker: &StaticChecker{Available: func() (bool, string) { return true, "ignored" }}, status: StatusAvailable, msg: "available"},
		{name: "unavailable", checker: &StaticChecker{Available: func() (bool, string) { return false, "not installed" }}, status: StatusUnavailable, msg: "not installed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := tc.checker.Check(context.Background())
			if status != tc.status || msg != tc.msg {
				t.Fatalf("Check() = (%q, %q), want (%q, %q)", status, msg, tc.status, tc.msg)
			}
		})
	}
}

func TestBridgeChecker_ProbeOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantStatus Status
		wantMsg    string
	}{
		{name: "healthy", statusCode: http.StatusOK, wantStatus: StatusAvailable, wantMsg: "Bridge is reachable and ready"},
		{name: "unhealthy", statusCode: http.StatusBadGateway, wantStatus: StatusUnavailable, wantMsg: "Bridge registry is unreachable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" || r.Header.Get("Authorization") != "owner" || r.Header.Get("X-Bridge-Owner-Reauth") != "reauth" {
					t.Fatalf("probe request = %s %s auth=%q reauth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("X-Bridge-Owner-Reauth"))
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			result := (&BridgeChecker{BaseURL: srv.URL, OwnerToken: "owner", ReauthToken: "reauth", Probe: true}).CheckResult(context.Background())
			if result.Status != tc.wantStatus || result.Message != tc.wantMsg {
				t.Fatalf("result = %+v, want status=%q message=%q", result, tc.wantStatus, tc.wantMsg)
			}
		})
	}

	configured := (&BridgeChecker{BaseURL: " https://bridge.test/ ", OwnerToken: "LocalSession owner", Probe: false}).CheckResult(context.Background())
	if configured.Status != StatusAvailable || configured.Message != "Bridge is configured" {
		t.Fatalf("configured result = %+v", configured)
	}
	refused := (&BridgeChecker{BaseURL: "http://127.0.0.1:1", OwnerToken: "owner", ReauthToken: "reauth", Probe: true}).CheckResult(context.Background())
	if refused.ReasonCode != "bridge_unreachable" {
		t.Fatalf("refused reason = %q", refused.ReasonCode)
	}
}

func TestResourceChecker_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := &ResourceChecker{URL: srv.URL}
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
		URL:    srv.URL,
		Client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
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

	checker := &ResourceChecker{URL: srv.URL}
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

	checker := &ResourceChecker{URL: url}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "resource is not responding" {
		t.Errorf("message = %q, want %q", msg, "resource is not responding")
	}
}

func TestResourceChecker_InvalidURL(t *testing.T) {
	status, msg := (&ResourceChecker{URL: "://bad"}).Check(context.Background())
	if status != StatusUnavailable || msg == "" {
		t.Fatalf("invalid URL result = (%q, %q)", status, msg)
	}
}

// fakeWhisperServer / WhisperChecker / KokoroChecker tests removed in the
// audio-tools adoption — the corresponding checker types were deleted
// (audio-tools owns Whisper + Kokoro end-to-end now). Resource liveness
// is exercised via the audio-tools scenario's own checker tests.

func TestOllamaChecker_Available(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	checker := &OllamaChecker{BaseURL: srv.URL}
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

	checker := &OllamaChecker{BaseURL: srv.URL}
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

	checker := &OllamaChecker{BaseURL: url}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "Ollama is not responding" {
		t.Errorf("message = %q, want %q", msg, "Ollama is not responding")
	}
}

func TestOllamaChecker_InvalidURL(t *testing.T) {
	status, msg := (&OllamaChecker{BaseURL: "://bad"}).Check(context.Background())
	if status != StatusUnavailable || msg == "" {
		t.Fatalf("invalid URL result = (%q, %q)", status, msg)
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

	checker := &OpenRouterChecker{APIKey: "test-key", BaseURL: srv.URL}
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

	checker := &OpenRouterChecker{APIKey: "bad-key", BaseURL: srv.URL}
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

	checker := &OpenRouterChecker{APIKey: "some-key", BaseURL: url}
	status, msg := checker.Check(context.Background())

	if status != StatusUnavailable {
		t.Errorf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg != "OpenRouter is not reachable" {
		t.Errorf("message = %q, want %q", msg, "OpenRouter is not reachable")
	}
}

func TestOpenRouterChecker_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	status, msg := (&OpenRouterChecker{APIKey: "key", BaseURL: srv.URL}).Check(context.Background())
	if status != StatusUnavailable || msg != "OpenRouter returned unexpected status" {
		t.Fatalf("result = (%q, %q)", status, msg)
	}
}

// TestGenerateSilentWAV removed — generateSilentWAV was a helper for
// WhisperChecker which has been deleted in the audio-tools adoption.
