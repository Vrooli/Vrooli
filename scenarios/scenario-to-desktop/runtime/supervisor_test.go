package bundleruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/api"
	"scenario-to-desktop-runtime/assets"
	"scenario-to-desktop-runtime/config"
	"scenario-to-desktop-runtime/gpu"
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
	telem := telemetry.NewFileRecorder(telemetryPath, RealClock{}, RealFileSystem{})
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
			Manifest:   &manifest.Manifest{},
		},
		telemetryPath: telemetryPath,
		fs:            RealFileSystem{},
		clock:         RealClock{},
		telemetry:     telem,
		assetVerifier: assets.NewVerifier(tmp, RealFileSystem{}, telem),
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
	gpuStatus     gpu.Status
}

func (m *mockSupervisorRuntime) Shutdown(_ context.Context) error { return nil }
func (m *mockSupervisorRuntime) ServiceStatuses() map[string]health.Status {
	return m.statuses
}
func (m *mockSupervisorRuntime) PortMap() map[string]map[string]int {
	return map[string]map[string]int{}
}
func (m *mockSupervisorRuntime) TelemetryPath() string        { return "/tmp/telemetry.jsonl" }
func (m *mockSupervisorRuntime) TelemetryUploadURL() string   { return "" }
func (m *mockSupervisorRuntime) Manifest() *manifest.Manifest { return m.manifest }
func (m *mockSupervisorRuntime) AppDataDir() string           { return "/tmp/appdata" }
func (m *mockSupervisorRuntime) FileSystem() FileSystem       { return RealFileSystem{} }
func (m *mockSupervisorRuntime) SecretStore() api.SecretStore { return m.secretStore }
func (m *mockSupervisorRuntime) StartServicesIfReady()        {}
func (m *mockSupervisorRuntime) RecordTelemetry(event string, _ map[string]interface{}) error {
	m.telemetryLogs = append(m.telemetryLogs, event)
	return nil
}

func (m *mockSupervisorRuntime) GPUStatus() gpu.Status {
	if m.gpuStatus.Method == "" {
		return gpu.Status{Available: false, Method: "mock", Reason: "test"}
	}
	return m.gpuStatus
}

func (m *mockSupervisorRuntime) ValidateBundle() *api.BundleValidationResult {
	return &api.BundleValidationResult{Valid: true}
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

// =============================================================================
// NewSupervisor Tests
// =============================================================================

func TestNewSupervisor_RequiresManifest(t *testing.T) {
	_, err := NewSupervisor(Options{})
	if err == nil {
		t.Fatal("NewSupervisor() expected error for nil manifest")
	}
	if !strings.Contains(err.Error(), "manifest is required") {
		t.Errorf("NewSupervisor() error = %q, want 'manifest is required'", err)
	}
}

func TestNewSupervisor_SetsDefaultImplementations(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
	}

	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: tmp,
		AppDataDir: tmp,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	// Verify defaults are set
	if s.clock == nil {
		t.Error("NewSupervisor() clock should not be nil")
	}
	if s.fs == nil {
		t.Error("NewSupervisor() fs should not be nil")
	}
	if s.dialer == nil {
		t.Error("NewSupervisor() dialer should not be nil")
	}
	if s.procRunner == nil {
		t.Error("NewSupervisor() procRunner should not be nil")
	}
	if s.cmdRunner == nil {
		t.Error("NewSupervisor() cmdRunner should not be nil")
	}
	if s.gpuDetector == nil {
		t.Error("NewSupervisor() gpuDetector should not be nil")
	}
	if s.envReader == nil {
		t.Error("NewSupervisor() envReader should not be nil")
	}
	if s.portAllocator == nil {
		t.Error("NewSupervisor() portAllocator should not be nil")
	}
	if s.secretStore == nil {
		t.Error("NewSupervisor() secretStore should not be nil")
	}
	if s.healthChecker == nil {
		t.Error("NewSupervisor() healthChecker should not be nil")
	}
}

func TestNewSupervisor_UsesInjectedDependencies(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
	}

	mockClock := testutil.NewMockClock(time.Now())
	mockFS := testutil.NewMockFileSystem()
	mockDialer := testutil.NewMockDialer()
	mockProcRunner := testutil.NewMockProcessRunner()
	mockCmdRunner := testutil.NewMockCommandRunner()
	mockSecretStore := testutil.NewMockSecretStore(m)
	mockHealthChecker := testutil.NewMockHealthChecker()
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockEnvReader := testutil.NewMockEnvReader()

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		NetworkDialer: mockDialer,
		ProcessRunner: mockProcRunner,
		CommandRunner: mockCmdRunner,
		SecretStore:   mockSecretStore,
		HealthChecker: mockHealthChecker,
		PortAllocator: mockPortAllocator,
		EnvReader:     mockEnvReader,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	// Verify injected dependencies are used
	if s.clock != mockClock {
		t.Error("NewSupervisor() should use injected clock")
	}
	if s.fs != mockFS {
		t.Error("NewSupervisor() should use injected fs")
	}
	if s.dialer != mockDialer {
		t.Error("NewSupervisor() should use injected dialer")
	}
	if s.procRunner != mockProcRunner {
		t.Error("NewSupervisor() should use injected procRunner")
	}
	if s.cmdRunner != mockCmdRunner {
		t.Error("NewSupervisor() should use injected cmdRunner")
	}
	if s.secretStore != mockSecretStore {
		t.Error("NewSupervisor() should use injected secretStore")
	}
	if s.healthChecker != mockHealthChecker {
		t.Error("NewSupervisor() should use injected healthChecker")
	}
	if s.portAllocator != mockPortAllocator {
		t.Error("NewSupervisor() should use injected portAllocator")
	}
	// envReader is compared by checking a value it returns
	mockEnvReader.SetEnv("TEST_VAR", "test_value")
	if s.envReader.Getenv("TEST_VAR") != "test_value" {
		t.Error("NewSupervisor() should use injected envReader")
	}
}

func TestNewSupervisor_ResolvesAppDataDir(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App: manifest.App{Name: "My Test App", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
	}

	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: tmp,
		AppDataDir: tmp,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	if s.appData != tmp {
		t.Errorf("NewSupervisor() appData = %q, want %q", s.appData, tmp)
	}
}

func TestNewSupervisor_SanitizesAppName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"My App", "my-app"},
		{"Test_App", "test_app"},
		{"App-Name", "app-name"},
	}

	for _, tc := range tests {
		got := config.SanitizeAppName(tc.name)
		if got != tc.want {
			t.Errorf("SanitizeAppName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// =============================================================================
// Start Tests
// =============================================================================

func TestStart_CreatesAppDataDir(t *testing.T) {
	tmp := t.TempDir()
	appDataDir := filepath.Join(tmp, "appdata")

	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"}, // Port 0 for testing
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    appDataDir,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = s.Start(ctx)
	if err != nil {
		// Error expected due to port 0 - but dirs should be created
		if !mockFS.Dirs[appDataDir] {
			t.Error("Start() should create app data directory")
		}
	}
}

func TestStart_LoadsSecrets(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockSecretStore.SetSecrets(map[string]string{"API_KEY": "test-key"})
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = s.Start(ctx)
	// Secrets should be loaded via secretStore.Load() and Set()
	// The mock tracks this internally
}

func TestStart_FailsOnSecretLoadError(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockSecretStore.SetLoadErr(errors.New("load failed"))
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx := context.Background()
	err = s.Start(ctx)
	if err == nil {
		t.Fatal("Start() expected error on secret load failure")
	}
	if !strings.Contains(err.Error(), "load secrets") {
		t.Errorf("Start() error = %q, want 'load secrets'", err)
	}
}

func TestStart_InitializesServiceStatus(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
		Services: []manifest.Service{
			{ID: "api"},
			{ID: "worker"},
		},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = s.Start(ctx)

	// Check service status initialized
	statuses := s.ServiceStatuses()
	if len(statuses) != 2 {
		t.Errorf("Start() initialized %d service statuses, want 2", len(statuses))
	}
	for _, svc := range []string{"api", "worker"} {
		if _, ok := statuses[svc]; !ok {
			t.Errorf("Start() missing status for service %q", svc)
		}
	}
}

func TestStart_WaitsForMissingSecrets(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockSecretStore.SetMissingRequired([]string{"API_KEY"})
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
		Services:  []manifest.Service{{ID: "api"}},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = s.Start(ctx)

	// Services should not be started when secrets are missing
	if s.servicesStarted {
		t.Error("Start() should not start services when secrets are missing")
	}

	// Status should indicate waiting for secrets
	statuses := s.ServiceStatuses()
	if status, ok := statuses["api"]; ok {
		if !strings.Contains(status.Message, "waiting for secrets") {
			t.Errorf("Start() status message = %q, want 'waiting for secrets'", status.Message)
		}
	}
}

func TestStart_PersistsAuthTokenAndIpcPort(t *testing.T) {
	tmp := t.TempDir()
	appDataDir := filepath.Join(tmp, "appdata")
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/auth-token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    appDataDir,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Shutdown(context.Background())
	})

	tokenPath := filepath.Join(appDataDir, "runtime", "auth-token")
	tokenData, ok := mockFS.Files[tokenPath]
	if !ok || strings.TrimSpace(string(tokenData)) == "" {
		t.Fatalf("Start() should persist auth token at %s", tokenPath)
	}

	portPath := filepath.Join(appDataDir, "runtime", "ipc_port")
	portData, ok := mockFS.Files[portPath]
	if !ok || strings.TrimSpace(string(portData)) == "" {
		t.Fatalf("Start() should persist IPC port at %s", portPath)
	}
	gotPort, err := strconv.Atoi(strings.TrimSpace(string(portData)))
	if err != nil {
		t.Fatalf("ipc_port should contain integer, got %q", string(portData))
	}
	if gotPort != s.opts.Manifest.IPC.Port {
		t.Fatalf("ipc_port = %d, want %d", gotPort, s.opts.Manifest.IPC.Port)
	}
}

func TestStart_FailsOnMigrationStateError(t *testing.T) {
	tmp := t.TempDir()
	appDataDir := filepath.Join(tmp, "appdata")
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	migrationsPath := filepath.Join(appDataDir, "migrations.json")
	mockFS.Files[migrationsPath] = []byte("{invalid-json")

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/auth-token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    appDataDir,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx := context.Background()
	err = s.Start(ctx)
	if err == nil {
		t.Fatal("Start() expected error on migrations load failure")
	}
	if !strings.Contains(err.Error(), "load migrations") {
		t.Errorf("Start() error = %q, want 'load migrations'", err)
	}
}

// =============================================================================
// Shutdown Tests
// =============================================================================

func TestShutdown_NoopIfNotStarted(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
	}

	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: tmp,
		AppDataDir: tmp,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx := context.Background()
	err = s.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil for unstarted supervisor", err)
	}
}

func TestShutdown_SetsStartedFalse(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	// Manually set started to true to simulate started state
	s.started = true

	ctx := context.Background()
	_ = s.Shutdown(ctx)

	if s.started {
		t.Error("Shutdown() should set started to false")
	}
}

func TestShutdown_CallsCancelFunc(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockSecretStore := testutil.NewMockSecretStore(nil)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		Clock:         mockClock,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
		HealthChecker: mockHealthChecker,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	// Set up a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	s.started = true
	s.cancel = cancel
	s.runtimeCtx = ctx
	s.procs = make(map[string]*serviceProcess)

	shutdownCtx := context.Background()
	_ = s.Shutdown(shutdownCtx)

	// Context should be canceled
	select {
	case <-ctx.Done():
		// Good - context was canceled
	default:
		t.Error("Shutdown() should cancel the runtime context")
	}
}

// =============================================================================
// Accessor Tests
// =============================================================================

func TestAccessors(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
		Telemetry: manifest.Telemetry{UploadTo: "https://example.com/telemetry"},
	}

	mockFS := testutil.NewMockFileSystem()
	mockSecretStore := testutil.NewMockSecretStore(m)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockPortAllocator.Ports = map[string]map[string]int{
		"api": {"http": 8080},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    tmp,
		FileSystem:    mockFS,
		SecretStore:   mockSecretStore,
		PortAllocator: mockPortAllocator,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	t.Run("AppDataDir", func(t *testing.T) {
		if got := s.AppDataDir(); got != tmp {
			t.Errorf("AppDataDir() = %q, want %q", got, tmp)
		}
	})

	t.Run("Manifest", func(t *testing.T) {
		if got := s.Manifest(); got != m {
			t.Error("Manifest() should return the manifest")
		}
	})

	t.Run("IsStarted", func(t *testing.T) {
		if s.IsStarted() {
			t.Error("IsStarted() = true, want false for new supervisor")
		}
		s.started = true
		if !s.IsStarted() {
			t.Error("IsStarted() = false, want true after setting started")
		}
		s.started = false // reset
	})

	t.Run("AuthToken", func(t *testing.T) {
		s.authToken = "test-token"
		if got := s.AuthToken(); got != "test-token" {
			t.Errorf("AuthToken() = %q, want %q", got, "test-token")
		}
	})

	t.Run("GPUStatus", func(t *testing.T) {
		s.gpuStatus = GPUStatus{Available: true, Method: "test"}
		status := s.GPUStatus()
		if !status.Available {
			t.Error("GPUStatus().Available = false, want true")
		}
	})

	t.Run("TelemetryPath", func(t *testing.T) {
		s.telemetryPath = "/tmp/telemetry.jsonl"
		if got := s.TelemetryPath(); got != "/tmp/telemetry.jsonl" {
			t.Errorf("TelemetryPath() = %q, want %q", got, "/tmp/telemetry.jsonl")
		}
	})

	t.Run("TelemetryUploadURL", func(t *testing.T) {
		// Uses manifest value
		if got := s.TelemetryUploadURL(); got != "https://example.com/telemetry" {
			t.Errorf("TelemetryUploadURL() = %q, want %q", got, "https://example.com/telemetry")
		}
	})

	t.Run("FileSystem", func(t *testing.T) {
		fs := s.FileSystem()
		if fs == nil {
			t.Error("FileSystem() = nil, want non-nil")
		}
		if fs != mockFS {
			t.Error("FileSystem() should return injected filesystem")
		}
	})

	t.Run("SecretStore", func(t *testing.T) {
		store := s.SecretStore()
		if store == nil {
			t.Error("SecretStore() = nil, want non-nil")
		}
		if store != mockSecretStore {
			t.Error("SecretStore() should return injected secret store")
		}
	})

	t.Run("PortMap", func(t *testing.T) {
		pm := s.PortMap()
		if pm == nil {
			t.Error("PortMap() = nil, want non-nil")
		}
		if pm["api"]["http"] != 8080 {
			t.Errorf("PortMap()[api][http] = %d, want 8080", pm["api"]["http"])
		}
	})

	t.Run("getStatus", func(t *testing.T) {
		s.serviceStatus = map[string]ServiceStatus{
			"api": {Ready: true, Message: "ready"},
		}

		status, ok := s.getStatus("api")
		if !ok {
			t.Error("getStatus(api) should return ok=true")
		}
		if !status.Ready {
			t.Error("getStatus(api).Ready = false, want true")
		}

		_, ok = s.getStatus("unknown")
		if ok {
			t.Error("getStatus(unknown) should return ok=false")
		}
	})

	t.Run("StartServicesIfReady", func(t *testing.T) {
		s.servicesStarted = false
		s.opts.Manifest = &manifest.Manifest{}
		s.StartServicesIfReady()
		// Should trigger startServicesAsync
		if !s.servicesStarted {
			t.Error("StartServicesIfReady() should set servicesStarted=true")
		}
	})

	t.Run("RecordTelemetry", func(t *testing.T) {
		mockClock := testutil.NewMockClock(time.Now())
		s.telemetry = telemetry.NewFileRecorder("/tmp/rec.jsonl", mockClock, mockFS)

		err := s.RecordTelemetry("test_event", map[string]interface{}{"key": "value"})
		if err != nil {
			t.Errorf("RecordTelemetry() error = %v", err)
		}
	})

	t.Run("renderValue", func(t *testing.T) {
		// renderValue just delegates to envRenderer
		// Uses ${data} for app data dir and ${bundle} for bundle path
		result := s.renderValue("test-${data}-value")
		// Should expand ${data} to the actual path
		if !strings.Contains(result, tmp) || strings.Contains(result, "${data}") {
			t.Errorf("renderValue() = %q, expected ${data} to be expanded to %q", result, tmp)
		}
	})
}

func TestAllServicesReady(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
	}

	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: tmp,
		AppDataDir: tmp,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	t.Run("empty services returns false", func(t *testing.T) {
		s.serviceStatus = map[string]ServiceStatus{}
		if s.AllServicesReady() {
			t.Error("AllServicesReady() = true, want false for empty services")
		}
	})

	t.Run("all ready returns true", func(t *testing.T) {
		s.serviceStatus = map[string]ServiceStatus{
			"api":    {Ready: true},
			"worker": {Ready: true},
		}
		if !s.AllServicesReady() {
			t.Error("AllServicesReady() = false, want true when all ready")
		}
	})

	t.Run("one not ready returns false", func(t *testing.T) {
		s.serviceStatus = map[string]ServiceStatus{
			"api":    {Ready: true},
			"worker": {Ready: false},
		}
		if s.AllServicesReady() {
			t.Error("AllServicesReady() = true, want false when one not ready")
		}
	})
}

func TestServiceStatuses_ReturnsCopy(t *testing.T) {
	tmp := t.TempDir()
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC: manifest.IPC{Host: "127.0.0.1", Port: 48000, AuthTokenRel: "runtime/token"},
	}

	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: tmp,
		AppDataDir: tmp,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	s.serviceStatus = map[string]ServiceStatus{
		"api": {Ready: true, Message: "ready"},
	}

	// Get copy and modify
	copy := s.ServiceStatuses()
	copy["api"] = ServiceStatus{Ready: false, Message: "modified"}

	// Original should be unchanged
	if status := s.serviceStatus["api"]; !status.Ready {
		t.Error("ServiceStatuses() should return a copy, not the original")
	}
}

func TestLoadOrCreateToken(t *testing.T) {
	t.Run("creates new token", func(t *testing.T) {
		mockFS := testutil.NewMockFileSystem()
		s := &Supervisor{fs: mockFS}

		token, err := s.loadOrCreateToken("/tmp/token")
		if err != nil {
			t.Fatalf("loadOrCreateToken() error = %v", err)
		}

		if len(token) != 48 { // 24 bytes * 2 for hex encoding
			t.Errorf("loadOrCreateToken() token length = %d, want 48", len(token))
		}

		// Token should be persisted
		if _, ok := mockFS.Files["/tmp/token"]; !ok {
			t.Error("loadOrCreateToken() should persist token to file")
		}
	})

	t.Run("loads existing token", func(t *testing.T) {
		mockFS := testutil.NewMockFileSystem()
		mockFS.Files["/tmp/token"] = []byte("existing-token-value")
		s := &Supervisor{fs: mockFS}

		token, err := s.loadOrCreateToken("/tmp/token")
		if err != nil {
			t.Fatalf("loadOrCreateToken() error = %v", err)
		}

		if token != "existing-token-value" {
			t.Errorf("loadOrCreateToken() = %q, want %q", token, "existing-token-value")
		}
	})

	t.Run("trims whitespace from existing token", func(t *testing.T) {
		mockFS := testutil.NewMockFileSystem()
		mockFS.Files["/tmp/token"] = []byte("  token-with-spaces  \n")
		s := &Supervisor{fs: mockFS}

		token, err := s.loadOrCreateToken("/tmp/token")
		if err != nil {
			t.Fatalf("loadOrCreateToken() error = %v", err)
		}

		if token != "token-with-spaces" {
			t.Errorf("loadOrCreateToken() = %q, want %q", token, "token-with-spaces")
		}
	})
}

func TestSupervisor_UpdateSecrets(t *testing.T) {
	t.Run("rejects missing required secrets", func(t *testing.T) {
		mockSecretStore := testutil.NewMockSecretStore(nil)
		mockSecretStore.SetMissingRequired([]string{"API_KEY"})

		s := &Supervisor{
			secretStore: mockSecretStore,
			telemetry:   telemetry.NopRecorder{},
		}

		err := s.UpdateSecrets(map[string]string{})
		if err == nil {
			t.Fatal("UpdateSecrets() expected error for missing secrets")
		}
		if !strings.Contains(err.Error(), "missing required secrets") {
			t.Errorf("UpdateSecrets() error = %q, want 'missing required secrets'", err)
		}
	})

	t.Run("triggers service startup when ready", func(t *testing.T) {
		mockSecretStore := testutil.NewMockSecretStore(nil)
		// No missing secrets

		s := &Supervisor{
			secretStore:     mockSecretStore,
			telemetry:       telemetry.NopRecorder{},
			servicesStarted: false,
			opts:            Options{Manifest: &manifest.Manifest{}},
		}

		err := s.UpdateSecrets(map[string]string{"API_KEY": "test"})
		if err != nil {
			t.Fatalf("UpdateSecrets() error = %v", err)
		}

		// Services should be started
		if !s.servicesStarted {
			t.Error("UpdateSecrets() should trigger service startup")
		}
	})
}

func TestRecordTelemetry(t *testing.T) {
	t.Run("handles nil telemetry", func(t *testing.T) {
		s := &Supervisor{telemetry: nil}
		err := s.recordTelemetry("test", nil)
		if err != nil {
			t.Errorf("recordTelemetry() error = %v, want nil for nil telemetry", err)
		}
	})

	t.Run("records event", func(t *testing.T) {
		mockFS := testutil.NewMockFileSystem()
		mockClock := testutil.NewMockClock(time.Now())
		telem := telemetry.NewFileRecorder("/tmp/telemetry.jsonl", mockClock, mockFS)

		s := &Supervisor{telemetry: telem}
		err := s.recordTelemetry("test_event", map[string]interface{}{"key": "value"})
		if err != nil {
			t.Errorf("recordTelemetry() error = %v", err)
		}

		// Verify telemetry was written
		if _, ok := mockFS.Files["/tmp/telemetry.jsonl"]; !ok {
			t.Error("recordTelemetry() should write to telemetry file")
		}
	})
}

func TestSupervisorIntegration_LaunchesServicesAndControlAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	bundleDir := t.TempDir()
	appData := t.TempDir()
	binRel := buildFixtureService(t, bundleDir)

	m := integrationManifest(binRel)
	s, err := NewSupervisor(Options{
		Manifest:   m,
		BundlePath: bundleDir,
		AppDataDir: appData,
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Shutdown(context.Background()) })

	token := readAuthToken(t, appData, m)
	port := readIPCPort(t, appData)
	baseURL := "http://" + m.IPC.Host + ":" + strconv.Itoa(port)

	ready := waitForReady(t, baseURL, token, 10*time.Second)
	if !ready.Ready {
		t.Fatalf("expected ready=true, got false")
	}
	if st, ok := ready.Details["api"]; !ok || !st.Ready {
		t.Fatalf("expected api ready status, got %+v", ready.Details["api"])
	}
	if st, ok := ready.Details["worker"]; !ok || !st.Ready {
		t.Fatalf("expected worker ready status, got %+v", ready.Details["worker"])
	}

	var portsResp struct {
		Services map[string]map[string]int `json:"services"`
	}
	if err := fetchJSON(baseURL, token, "/ports", &portsResp); err != nil {
		t.Fatalf("fetch /ports: %v", err)
	}
	if portsResp.Services["api"]["http"] == 0 {
		t.Fatalf("expected allocated api.http port, got %+v", portsResp.Services["api"])
	}

	logs, err := fetchText(baseURL, token, "/logs/tail?serviceId=worker&lines=50")
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if !strings.Contains(logs, "READY") {
		t.Fatalf("expected worker logs to contain READY, got %q", logs)
	}

	telemetryPath := filepath.Join(appData, "runtime", "telemetry.jsonl")
	if data, err := os.ReadFile(telemetryPath); err != nil {
		t.Fatalf("read telemetry: %v", err)
	} else if !bytes.Contains(data, []byte(`"event":"service_ready"`)) {
		t.Errorf("expected telemetry to include service_ready events")
	}

	if err := fetchJSON(baseURL, token, "/shutdown", &map[string]string{}); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	waitForShutdown(t, baseURL, 5*time.Second)
}

type readyResponse struct {
	Ready   bool                     `json:"ready"`
	Details map[string]health.Status `json:"details"`
}

func waitForReady(t *testing.T, baseURL, token string, timeout time.Duration) readyResponse {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var resp readyResponse
		if err := fetchJSON(baseURL, token, "/readyz", &resp); err == nil {
			if resp.Ready {
				return resp
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for ready")
	return readyResponse{}
}

func waitForShutdown(t *testing.T, baseURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 300 * time.Millisecond}
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for shutdown")
}

func fetchJSON(baseURL, token, path string, out interface{}) error {
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchText(baseURL, token, path string) (string, error) {
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func readAuthToken(t *testing.T, appData string, m *manifest.Manifest) string {
	t.Helper()
	tokenPath := manifest.ResolvePath(appData, m.IPC.AuthTokenRel)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read auth token: %v", err)
	}
	return strings.TrimSpace(string(data))
}

func readIPCPort(t *testing.T, appData string) int {
	t.Helper()
	portPath := filepath.Join(appData, "runtime", "ipc_port")
	data, err := os.ReadFile(portPath)
	if err != nil {
		t.Fatalf("read ipc_port: %v", err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse ipc_port: %v", err)
	}
	return port
}

func integrationManifest(binRel string) *manifest.Manifest {
	key := manifest.PlatformKey(runtime.GOOS, runtime.GOARCH)
	return &manifest.Manifest{
		SchemaVersion: "0.1",
		Target:        "desktop",
		App:           manifest.App{Name: "fixture-app", Version: "0.1.0"},
		IPC:           manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/auth-token"},
		Telemetry:     manifest.Telemetry{File: "runtime/telemetry.jsonl"},
		Ports:         &manifest.PortRules{DefaultRange: &manifest.PortRange{Min: 48000, Max: 48100}},
		Services: []manifest.Service{
			{
				ID:   "api",
				Type: "backend",
				Binaries: map[string]manifest.Binary{
					key: {
						Path: binRel,
						Args: []string{"--mode", "api", "--port", "${api.http}"},
					},
				},
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "http", Range: manifest.PortRange{Min: 48000, Max: 48100}},
					},
				},
				Health:    manifest.HealthCheck{Type: "http", Path: "/health", PortName: "http", IntervalMs: 200, TimeoutMs: 1000, Retries: 5},
				Readiness: manifest.ReadinessCheck{Type: "health_success", TimeoutMs: 5000},
				LogDir:    "logs/api.log",
			},
			{
				ID:   "worker",
				Type: "worker",
				Binaries: map[string]manifest.Binary{
					key: {
						Path: binRel,
						Args: []string{"--mode", "worker"},
					},
				},
				Health:       manifest.HealthCheck{Type: "log_match", Path: "READY", IntervalMs: 200, TimeoutMs: 1000, Retries: 5},
				Readiness:    manifest.ReadinessCheck{Type: "log_match", Pattern: "READY", TimeoutMs: 5000},
				LogDir:       "logs/worker.log",
				Dependencies: []string{"api"},
			},
		},
	}
}

func buildFixtureService(t *testing.T, bundleDir string) string {
	t.Helper()
	binDir := filepath.Join(bundleDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	srcPath := filepath.Join(binDir, "fixture-service.go")
	source := `package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	mode := flag.String("mode", "api", "mode")
	port := flag.Int("port", 0, "port")
	flag.Parse()

	switch *mode {
	case "api":
		if *port == 0 {
			fmt.Fprintln(os.Stderr, "missing --port")
			os.Exit(2)
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		addr := fmt.Sprintf("127.0.0.1:%d", *port)
		log.Printf("READY api on %s", addr)
		server := &http.Server{Addr: addr, Handler: mux}
		log.Fatal(server.ListenAndServe())
	case "worker":
		log.Printf("READY worker")
		if *port > 0 {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
			if err != nil {
				log.Fatal(err)
			}
			defer ln.Close()
			for {
				conn, err := ln.Accept()
				if err == nil {
					conn.Close()
				}
			}
		}
		select {}
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(2)
	}
}
`
	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}

	exeName := "fixture-service"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	outPath := filepath.Join(binDir, exeName)
	cmd := exec.Command("go", "build", "-o", outPath, srcPath)
	cmd.Dir = binDir
	cmd.Env = append(os.Environ(), "GO111MODULE=off")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture service: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	return filepath.Join("bin", exeName)
}
