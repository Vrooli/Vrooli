package codesigning

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type proxyScenarioLookup struct {
	scenario string
	err      error
}

func (l proxyScenarioLookup) GetScenarioAndTier(context.Context, string) (string, int, error) {
	return l.scenario, 1, l.err
}

func TestProxyRepositoryRoundTripAndDelegatedOperations(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/signing/demo"):
			_ = json.NewEncoder(w).Encode(signingConfigResponse{Scenario: "demo", Config: &SigningConfig{Enabled: true}})
		case strings.HasSuffix(r.URL.Path, "/validate"):
			_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/ready"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ready": true, "platform": "linux"})
		case r.URL.Path == "/api/v1/signing/prerequisites":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": []ToolDetectionResult{{Tool: "signtool", Platform: PlatformWindows}}})
		case strings.Contains(r.URL.Path, "/discover/"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"certificates": []DiscoveredCertificate{{Platform: PlatformLinux}}})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	repo := NewProxyRepository(proxyScenarioLookup{scenario: "demo"}, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	ctx := context.Background()
	config, err := repo.Get(ctx, "profile-1")
	if err != nil || config == nil || !config.Enabled {
		t.Fatalf("get: %+v %v", config, err)
	}
	config.Windows = &WindowsSigningConfig{CertificateSource: CertSourceFile, CertificatePasswordEnv: "CERT_PASSWORD"}
	for name, call := range map[string]func() error{
		"save":            func() error { return repo.Save(ctx, "profile-1", config) },
		"delete":          func() error { return repo.Delete(ctx, "profile-1") },
		"save platform":   func() error { return repo.SaveForPlatform(ctx, "profile-1", PlatformLinux, &LinuxSigningConfig{}) },
		"delete platform": func() error { return repo.DeleteForPlatform(ctx, "profile-1", PlatformLinux) },
	} {
		if err := call(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	platform, err := repo.GetForPlatform(ctx, "profile-1", PlatformWindows)
	if err != nil || platform == nil {
		t.Fatalf("platform get: %+v %v", platform, err)
	}
	if _, err := repo.GetForPlatform(ctx, "profile-1", "other"); err == nil {
		t.Fatal("unknown platform should fail")
	}
	validation, err := repo.Validate(ctx, "profile-1")
	if err != nil || validation == nil {
		t.Fatalf("validate: %+v %v", validation, err)
	}
	ready, details, err := repo.CheckReady(ctx, "profile-1")
	if err != nil || !ready || details["platform"] != "linux" {
		t.Fatalf("ready: %v %+v %v", ready, details, err)
	}
	tools, err := repo.DetectPrerequisites(ctx)
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools: %+v %v", tools, err)
	}
	certs, err := repo.DiscoverCertificates(ctx, PlatformLinux)
	if err != nil || len(certs) != 1 {
		t.Fatalf("certificates: %+v %v", certs, err)
	}
	if len(methods) < 8 {
		t.Fatalf("expected delegated calls, got %v", methods)
	}
}

func TestProxyRepositoryErrorsAndURLResolution(t *testing.T) {
	lookupErr := NewProxyRepository(proxyScenarioLookup{err: context.Canceled}, WithBaseURL("http://127.0.0.1:1"))
	if _, err := lookupErr.Get(context.Background(), "p"); err != ErrProfileNotFound {
		t.Fatalf("lookup error should map to profile not found: %v", err)
	}
	empty := NewProxyRepository(proxyScenarioLookup{scenario: ""}, WithBaseURL("http://127.0.0.1:1"))
	if _, err := empty.Get(context.Background(), "p"); err != ErrProfileNotFound {
		t.Fatalf("empty scenario should fail: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream failure"))
	}))
	defer server.Close()
	repo := NewProxyRepository(proxyScenarioLookup{scenario: "demo"}, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if _, err := repo.Get(context.Background(), "p"); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatal("upstream failure should be surfaced")
	}
	t.Setenv("SCENARIO_TO_DESKTOP_URL", " http://example.test/ ")
	if got := getDefaultSigningAPIURL(); got != "http://example.test" {
		t.Fatalf("unexpected configured URL %q", got)
	}
}
