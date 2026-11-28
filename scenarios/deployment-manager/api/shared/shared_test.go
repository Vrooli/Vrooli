package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Validation tests
// ---------------------------------------------------------------------------

func TestIsValidSecretClass(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string (legacy)", "", true},
		{"infrastructure", SecretClassInfrastructure, true},
		{"per_install_generated", SecretClassPerInstallGenerated, true},
		{"user_prompt", SecretClassUserPrompt, true},
		{"remote_fetch", SecretClassRemoteFetch, true},
		{"unknown class", "unknown", false},
		{"whitespace", " ", false},
		{"similar but wrong", "Infrastructure", false},
		{"numeric string", "123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSecretClass(tt.input); got != tt.want {
				t.Errorf("IsValidSecretClass(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsBundleSafeSecretClass(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"infrastructure is NOT safe", SecretClassInfrastructure, false},
		{"per_install_generated is safe", SecretClassPerInstallGenerated, true},
		{"user_prompt is safe", SecretClassUserPrompt, true},
		{"remote_fetch is safe", SecretClassRemoteFetch, true},
		{"empty string is safe", "", true},
		{"unknown class is safe (not infrastructure)", "unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBundleSafeSecretClass(tt.input); got != tt.want {
				t.Errorf("IsBundleSafeSecretClass(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidServiceType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"ui-bundle", ServiceTypeUIBundle, true},
		{"api-binary", ServiceTypeAPIBinary, true},
		{"worker", ServiceTypeWorker, true},
		{"resource", ServiceTypeResource, true},
		{"empty string", "", false},
		{"unknown", "microservice", false},
		{"case sensitive", "Worker", false},
		{"partial match", "ui", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidServiceType(tt.input); got != tt.want {
				t.Errorf("IsValidServiceType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetServiceTypeError(t *testing.T) {
	t.Run("valid type returns nil", func(t *testing.T) {
		for _, st := range []string{ServiceTypeUIBundle, ServiceTypeAPIBinary, ServiceTypeWorker, ServiceTypeResource} {
			if err := GetServiceTypeError(st); err != nil {
				t.Errorf("GetServiceTypeError(%q) = %v, want nil", st, err)
			}
		}
	})
	t.Run("invalid type returns error", func(t *testing.T) {
		err := GetServiceTypeError("bogus")
		if err == nil {
			t.Fatal("expected error for invalid service type, got nil")
		}
		// Error should mention the invalid type
		if got := err.Error(); got == "" {
			t.Error("expected non-empty error message")
		}
	})
	t.Run("error message includes the invalid value", func(t *testing.T) {
		err := GetServiceTypeError("bad-type")
		if err == nil {
			t.Fatal("expected error")
		}
		want := `"bad-type"`
		if got := err.Error(); !containsSubstring(got, want) {
			t.Errorf("error %q should contain %q", got, want)
		}
	})
}

func TestIsValidSecretTargetType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"env", SecretTargetEnv, true},
		{"file", SecretTargetFile, true},
		{"empty string", "", false},
		{"unknown", "volume", false},
		{"case sensitive", "Env", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSecretTargetType(tt.input); got != tt.want {
				t.Errorf("IsValidSecretTargetType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidIPCMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"loopback-http", IPCModeLoopbackHTTP, true},
		{"empty string", "", false},
		{"unix-socket", "unix-socket", false},
		{"case sensitive", "Loopback-HTTP", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidIPCMode(tt.input); got != tt.want {
				t.Errorf("IsValidIPCMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidSchemaVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"v0.1", BundleSchemaVersionV01, true},
		{"empty string", "", false},
		{"v1.0", "v1.0", false},
		{"no prefix", "0.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSchemaVersion(tt.input); got != tt.want {
				t.Errorf("IsValidSchemaVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidBundleTarget(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"desktop", BundleTargetDesktop, true},
		{"empty string", "", false},
		{"mobile", "mobile", false},
		{"case sensitive", "Desktop", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidBundleTarget(tt.input); got != tt.want {
				t.Errorf("IsValidBundleTarget(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HTTP response helper tests
// ---------------------------------------------------------------------------

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	JSONError(w, "something went wrong", http.StatusBadRequest)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "something went wrong" {
		t.Errorf("error field = %q, want %q", body["error"], "something went wrong")
	}
}

func TestJSONSuccess(t *testing.T) {
	type payload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	w := httptest.NewRecorder()
	JSONSuccess(w, payload{ID: 42, Name: "test"}, http.StatusAccepted)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusAccepted)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body payload
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ID != 42 || body.Name != "test" {
		t.Errorf("body = %+v, want {ID:42 Name:test}", body)
	}
}

func TestJSONOK(t *testing.T) {
	w := httptest.NewRecorder()
	JSONOK(w, map[string]string{"status": "ok"})

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("body[status] = %q, want %q", body["status"], "ok")
	}
}

func TestJSONCreated(t *testing.T) {
	w := httptest.NewRecorder()
	JSONCreated(w, map[string]int{"id": 1})

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusCreated)
	}

	var body map[string]int
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["id"] != 1 {
		t.Errorf("body[id] = %d, want 1", body["id"])
	}
}

func TestJSONError_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	JSONError(w, "", http.StatusInternalServerError)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusInternalServerError)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "" {
		t.Errorf("error field = %q, want empty string", body["error"])
	}
}

func TestJSONOK_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	JSONOK(w, nil)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	// nil encodes as JSON "null"
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// ---------------------------------------------------------------------------
// TimeProvider tests
// ---------------------------------------------------------------------------

type fakeTimeProvider struct {
	fixedTime time.Time
}

func (f *fakeTimeProvider) Now() time.Time {
	return f.fixedTime
}

func TestRealTimeProvider_ReturnsApproximatelyNow(t *testing.T) {
	rtp := RealTimeProvider{}
	before := time.Now()
	got := rtp.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("RealTimeProvider.Now() = %v, not between %v and %v", got, before, after)
	}
}

func TestSetAndGetTimeProvider(t *testing.T) {
	original := GetTimeProvider()
	t.Cleanup(func() { SetTimeProvider(original) })

	fixed := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	fake := &fakeTimeProvider{fixedTime: fixed}
	SetTimeProvider(fake)

	got := GetTimeProvider()
	if got != fake {
		t.Error("GetTimeProvider did not return the provider set via SetTimeProvider")
	}
	if now := got.Now(); !now.Equal(fixed) {
		t.Errorf("Now() = %v, want %v", now, fixed)
	}
}

func TestDefaultTimeProviderIsReal(t *testing.T) {
	original := GetTimeProvider()
	t.Cleanup(func() { SetTimeProvider(original) })

	// Reset to default
	SetTimeProvider(RealTimeProvider{})
	tp := GetTimeProvider()

	if _, ok := tp.(RealTimeProvider); !ok {
		t.Errorf("default TimeProvider type = %T, want RealTimeProvider", tp)
	}
}

// ---------------------------------------------------------------------------
// HTTPClient context injection tests
// ---------------------------------------------------------------------------

type fakeHTTPClient struct {
	response *http.Response
	err      error
}

func (f *fakeHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return f.response, f.err
}

func TestWithHTTPClient_GetHTTPClient_Roundtrip(t *testing.T) {
	fake := &fakeHTTPClient{}
	ctx := WithHTTPClient(context.Background(), fake)

	got := GetHTTPClient(ctx)
	if got != fake {
		t.Error("GetHTTPClient did not return the client injected via WithHTTPClient")
	}
}

func TestGetHTTPClient_FallsBackToDefault(t *testing.T) {
	got := GetHTTPClient(context.Background())
	if got != http.DefaultClient {
		t.Errorf("GetHTTPClient(empty ctx) = %v, want http.DefaultClient", got)
	}
}

func TestGetHTTPClient_NilValueFallsBack(t *testing.T) {
	// Inject nil explicitly
	ctx := WithHTTPClient(context.Background(), nil)
	got := GetHTTPClient(ctx)
	if got != http.DefaultClient {
		t.Errorf("GetHTTPClient(nil client ctx) = %v, want http.DefaultClient", got)
	}
}

// ---------------------------------------------------------------------------
// GetScenarioDependencies tests
// ---------------------------------------------------------------------------

func TestGetScenarioDependencies_Success(t *testing.T) {
	origResolver := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(origResolver) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/analyze/my-scenario" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"resources": []map[string]string{
				{"dependency_name": "postgres"},
				{"dependency_name": "redis"},
			},
		})
	}))
	defer srv.Close()

	SetConfigResolver(&staticConfigResolver{analyzerURL: srv.URL})

	deps, err := GetScenarioDependencies(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Fatalf("len(deps) = %d, want 2", len(deps))
	}
	if deps[0] != "postgres" || deps[1] != "redis" {
		t.Errorf("deps = %v, want [postgres redis]", deps)
	}
}

func TestGetScenarioDependencies_EmptyResources(t *testing.T) {
	origResolver := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(origResolver) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"resources": []map[string]string{},
		})
	}))
	defer srv.Close()

	SetConfigResolver(&staticConfigResolver{analyzerURL: srv.URL})

	deps, err := GetScenarioDependencies(context.Background(), "empty-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("len(deps) = %d, want 0", len(deps))
	}
}

func TestGetScenarioDependencies_NonOKStatus(t *testing.T) {
	origResolver := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(origResolver) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	SetConfigResolver(&staticConfigResolver{analyzerURL: srv.URL})

	_, err := GetScenarioDependencies(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
	if !containsSubstring(err.Error(), "404") {
		t.Errorf("error %q should mention status code 404", err.Error())
	}
}

func TestGetScenarioDependencies_ResolverError(t *testing.T) {
	origResolver := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(origResolver) })

	SetConfigResolver(&staticConfigResolver{analyzerErr: fmt.Errorf("resolver broken")})

	_, err := GetScenarioDependencies(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error when resolver fails, got nil")
	}
	if !containsSubstring(err.Error(), "resolver broken") {
		t.Errorf("error %q should contain 'resolver broken'", err.Error())
	}
}

func TestGetScenarioDependencies_UsesContextHTTPClient(t *testing.T) {
	origResolver := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(origResolver) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"resources": []map[string]string{
				{"dependency_name": "qdrant"},
			},
		})
	}))
	defer srv.Close()

	SetConfigResolver(&staticConfigResolver{analyzerURL: srv.URL})

	// Inject a custom HTTP client via context to confirm it's used
	ctx := WithHTTPClient(context.Background(), srv.Client())
	deps, err := GetScenarioDependencies(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 || deps[0] != "qdrant" {
		t.Errorf("deps = %v, want [qdrant]", deps)
	}
}

func TestGetScenarioDependencies_EscapesScenarioName(t *testing.T) {
	origResolver := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(origResolver) })

	var gotRequestURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"resources": []map[string]string{},
		})
	}))
	defer srv.Close()

	SetConfigResolver(&staticConfigResolver{analyzerURL: srv.URL})

	_, err := GetScenarioDependencies(context.Background(), "has spaces/slashes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// RequestURI preserves percent-encoding; spaces and slashes should be escaped
	if strings.Contains(gotRequestURI, " ") {
		t.Errorf("scenario name was not URL-escaped: got %q", gotRequestURI)
	}
}

// ---------------------------------------------------------------------------
// ConfigResolver tests
// ---------------------------------------------------------------------------

func TestEnvConfigResolver_ResolveAnalyzerURL_FromEnv(t *testing.T) {
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", "  http://localhost:9090/  ")
	r := NewEnvConfigResolver()
	got, err := r.ResolveAnalyzerURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should trim whitespace and trailing slash
	if got != "http://localhost:9090" {
		t.Errorf("ResolveAnalyzerURL() = %q, want %q", got, "http://localhost:9090")
	}
}

func TestEnvConfigResolver_ResolveSecretsManagerURL_FromEnv(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_URL", "http://secrets:8080/")
	r := NewEnvConfigResolver()
	got, err := r.ResolveSecretsManagerURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://secrets:8080" {
		t.Errorf("ResolveSecretsManagerURL() = %q, want %q", got, "http://secrets:8080")
	}
}

func TestEnvConfigResolver_ResolveDesktopPackagerURL_FromEnv(t *testing.T) {
	t.Setenv("SCENARIO_TO_DESKTOP_URL", "http://packager:5000")
	r := NewEnvConfigResolver()
	got, err := r.ResolveDesktopPackagerURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://packager:5000" {
		t.Errorf("ResolveDesktopPackagerURL() = %q, want %q", got, "http://packager:5000")
	}
}

func TestEnvConfigResolver_ResolveTelemetryDir_FromEnv(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR", "/custom/telemetry")
	r := NewEnvConfigResolver()
	got, err := r.ResolveTelemetryDir()
	if err != nil {
		t.Fatalf("ResolveTelemetryDir() unexpected error: %v", err)
	}
	if got != "/custom/telemetry" {
		t.Errorf("ResolveTelemetryDir() = %q, want %q", got, "/custom/telemetry")
	}
}

func TestEnvConfigResolver_ResolveTelemetryDir_Default(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR", "")
	r := NewEnvConfigResolver()
	got, err := r.ResolveTelemetryDir()
	if err != nil {
		t.Fatalf("ResolveTelemetryDir() unexpected error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Fatalf("UserHomeDir() unexpected error: %v", err)
	}
	want := filepath.Join(home, ".vrooli", "logs", "vrooli", "deployment-manager", "telemetry")
	if got != want {
		t.Errorf("ResolveTelemetryDir() = %q, want %q", got, want)
	}
}

func TestSetAndGetConfigResolver(t *testing.T) {
	original := GetConfigResolver()
	t.Cleanup(func() { SetConfigResolver(original) })

	mock := &staticConfigResolver{analyzerURL: "http://mock"}
	SetConfigResolver(mock)

	got := GetConfigResolver()
	if got != mock {
		t.Error("GetConfigResolver did not return the resolver set via SetConfigResolver")
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// staticConfigResolver is a test double for ConfigResolver.
type staticConfigResolver struct {
	analyzerURL  string
	analyzerErr  error
	secretsURL   string
	secretsErr   error
	desktopURL   string
	desktopErr   error
	telemetryDir string
}

func (s *staticConfigResolver) ResolveAnalyzerURL() (string, error) {
	return s.analyzerURL, s.analyzerErr
}

func (s *staticConfigResolver) ResolveSecretsManagerURL() (string, error) {
	return s.secretsURL, s.secretsErr
}

func (s *staticConfigResolver) ResolveDesktopPackagerURL() (string, error) {
	return s.desktopURL, s.desktopErr
}

func (s *staticConfigResolver) ResolveTelemetryDir() (string, error) {
	if s.telemetryDir != "" {
		return s.telemetryDir, nil
	}
	return "/tmp/test-telemetry", nil
}

// containsSubstring reports whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
