package bundleruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-desktop-runtime/api"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/telemetry"
	"scenario-to-desktop-runtime/testutil"
)

func TestEnsureAssetsSizeBudget(t *testing.T) {
	tmp := t.TempDir()
	assetPath := filepath.Join(tmp, "resources", "playwright", "chromium", "chrome")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	expected := int64(5 * 1024 * 1024) // 5MB
	if err := os.WriteFile(assetPath, make([]byte, expected), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
			Manifest:   &manifest.Manifest{},
		},
		telemetryPath: telemetryPath,
		fs:            RealFileSystem{},
		clock:         RealClock{},
		telemetry:     telemetry.NewFileRecorder(telemetryPath, RealClock{}, RealFileSystem{}),
	}
	svc := manifest.Service{
		ID: "playwright-driver",
		Assets: []manifest.Asset{
			{Path: "resources/playwright/chromium/chrome", SizeBytes: expected},
		},
	}

	if err := s.ensureAssets(svc); err != nil {
		t.Fatalf("expected asset budget to pass: %v", err)
	}

	// Grow the asset beyond the budget + slack to trigger a failure.
	if err := os.WriteFile(assetPath, make([]byte, expected+7*1024*1024), 0o644); err != nil { // +7MB
		t.Fatalf("expand asset: %v", err)
	}
	err := s.ensureAssets(svc)
	if err == nil {
		t.Fatalf("expected asset size budget violation")
	}
	if !strings.Contains(err.Error(), "size budget") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyPlaywrightConventionsFallback(t *testing.T) {
	tmp := t.TempDir()
	fallbackChrome := filepath.Join(tmp, "electron-chrome")
	if err := os.WriteFile(fallbackChrome, []byte("chrome"), 0o644); err != nil {
		t.Fatalf("write fallback chrome: %v", err)
	}
	t.Setenv("ELECTRON_CHROMIUM_PATH", fallbackChrome)

	telemetryPath := filepath.Join(tmp, "telemetry.jsonl")
	s := &Supervisor{
		opts: Options{
			BundlePath: filepath.Join(tmp, "bundle"),
			Manifest:   &manifest.Manifest{},
		},
		portAllocator: &testutil.MockPortAllocator{Ports: map[string]map[string]int{
			"playwright-driver": {"http": 48000},
		}},
		telemetryPath: telemetryPath,
		fs:            RealFileSystem{},
		clock:         RealClock{},
		envReader:     RealEnvReader{},
		telemetry:     telemetry.NewFileRecorder(telemetryPath, RealClock{}, RealFileSystem{}),
	}
	svc := manifest.Service{
		ID: "playwright-driver",
		Env: map[string]string{
			"PLAYWRIGHT_CHROMIUM_PATH": "resources/playwright/chromium/chrome",
		},
	}
	env := map[string]string{
		"PLAYWRIGHT_CHROMIUM_PATH": "resources/playwright/chromium/chrome",
	}

	if err := s.applyPlaywrightConventions(svc, env); err != nil {
		t.Fatalf("applyPlaywrightConventions: %v", err)
	}

	if got := env["PLAYWRIGHT_DRIVER_PORT"]; got != "48000" {
		t.Fatalf("expected driver port set from allocated port, got %q", got)
	}
	if got := env["PLAYWRIGHT_DRIVER_URL"]; got != "http://127.0.0.1:48000" {
		t.Fatalf("expected driver url set, got %q", got)
	}
	if got := env["PLAYWRIGHT_CHROMIUM_PATH"]; got != fallbackChrome {
		t.Fatalf("expected chromium fallback path, got %q", got)
	}
	if got := env["ENGINE"]; got != "playwright" {
		t.Fatalf("expected ENGINE=playwright, got %q", got)
	}
}

// mockSupervisorRuntime wraps a Supervisor to implement api.Runtime for testing.
type mockSupervisorRuntime struct {
	manifest      *manifest.Manifest
	secretStore   *secrets.Manager
	statuses      map[string]health.Status
	telemetryLogs []string
}

func (m *mockSupervisorRuntime) Shutdown(_ context.Context) error { return nil }
func (m *mockSupervisorRuntime) ServiceStatuses() map[string]health.Status {
	return m.statuses
}
func (m *mockSupervisorRuntime) PortMap() map[string]map[string]int {
	return map[string]map[string]int{}
}
func (m *mockSupervisorRuntime) TelemetryPath() string         { return "/tmp/telemetry.jsonl" }
func (m *mockSupervisorRuntime) TelemetryUploadURL() string    { return "" }
func (m *mockSupervisorRuntime) Manifest() *manifest.Manifest  { return m.manifest }
func (m *mockSupervisorRuntime) AppDataDir() string            { return "/tmp/appdata" }
func (m *mockSupervisorRuntime) FileSystem() FileSystem        { return RealFileSystem{} }
func (m *mockSupervisorRuntime) SecretStore() api.SecretStore  { return m.secretStore }
func (m *mockSupervisorRuntime) StartServicesIfReady()         {}
func (m *mockSupervisorRuntime) RecordTelemetry(event string, _ map[string]interface{}) error {
	m.telemetryLogs = append(m.telemetryLogs, event)
	return nil
}

func TestHandleSecretsGetReturnsStatus(t *testing.T) {
	manifestData := &manifest.Manifest{
		App: manifest.App{Name: "demo", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
		Services: []manifest.Service{
			{ID: "api", Health: manifest.HealthCheck{Type: "tcp"}, Readiness: manifest.ReadinessCheck{Type: "tcp"}},
		},
		Secrets: []manifest.Secret{
			{ID: "API_KEY", Class: "user_prompt", Description: "API key", Required: ptrBool(true), Prompt: map[string]string{"label": "API key"}},
			{ID: "OPTIONAL_HINT", Class: "per_install_generated", Required: ptrBool(false)},
		},
	}
	sm := secrets.NewManager(manifestData, testutil.NewMockFileSystem(), "/tmp/secrets.json")
	sm.Set(map[string]string{"OPTIONAL_HINT": "seed"})

	rt := &mockSupervisorRuntime{
		manifest:    manifestData,
		secretStore: sm,
		statuses:    map[string]health.Status{"api": {Ready: false}},
	}
	server := api.NewServer(rt, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/secrets", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Secrets []struct {
			ID       string `json:"id"`
			HasValue bool   `json:"has_value"`
			Required bool   `json:"required"`
		} `json:"secrets"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(resp.Secrets))
	}
	foundMissing := false
	for _, sec := range resp.Secrets {
		if sec.ID == "API_KEY" && (!sec.HasValue && sec.Required) {
			foundMissing = true
		}
		if sec.ID == "OPTIONAL_HINT" && !sec.HasValue {
			t.Fatalf("expected OPTIONAL_HINT to be marked present")
		}
	}
	if !foundMissing {
		t.Fatalf("expected to flag missing required API_KEY")
	}
}

func ptrBool(v bool) *bool {
	return &v
}
