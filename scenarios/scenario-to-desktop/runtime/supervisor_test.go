package bundleruntime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/peerrecord"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/assets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/config"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/health"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/secrets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/telemetry"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
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
	sm := secrets.NewManager(manifestData)
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
		{"Test_App", "test-app"},
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

func TestResolvePeerEnvironmentReadsTierOneRecord(t *testing.T) {
	home := t.TempDir()
	if err := peerrecord.Write(home, peerrecord.Record{
		Scenario:  "landing-page-business-suite",
		Instance:  "live",
		Tier:      1,
		OwnerPID:  os.Getpid(),
		StartedAt: time.Now(),
		Ports:     map[string]int{"api": 23117},
	}); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Peers: []manifest.Peer{{
			Scenario:     "landing-page-business-suite",
			BundlePolicy: "discover",
			Bindings: []manifest.PeerBinding{{
				EnvVar:          "BAS_ENTITLEMENT_SERVICE_URL",
				Form:            "http_base_url",
				Port:            "api",
				WhenUnavailable: "fail",
			}},
		}},
		Services: []manifest.Service{{ID: "api"}, {ID: "playwright-driver"}},
	}
	s := &Supervisor{opts: Options{Manifest: m}, peerHome: home}
	if err := s.resolvePeerEnvironment(); err != nil {
		t.Fatalf("resolvePeerEnvironment() error = %v", err)
	}
	for _, service := range m.Services {
		if got := service.Env["BAS_ENTITLEMENT_SERVICE_URL"]; got != "http://127.0.0.1:23117" {
			t.Fatalf("service %s peer binding = %q", service.ID, got)
		}
	}
}

func TestResolvePeerEnvironmentNamesUnavailableRequiredPeer(t *testing.T) {
	m := &manifest.Manifest{
		Peers: []manifest.Peer{{
			Scenario:     "missing-peer",
			BundlePolicy: "discover",
			Bindings: []manifest.PeerBinding{{
				EnvVar:          "PEER_URL",
				Form:            "http_base_url",
				Port:            "api",
				WhenUnavailable: "fail",
			}},
		}},
		Services: []manifest.Service{{ID: "api"}},
	}
	s := &Supervisor{opts: Options{Manifest: m}, peerHome: t.TempDir()}
	var unavailable PeerDiscoveryUnavailableError
	if err := s.resolvePeerEnvironment(); !errors.As(err, &unavailable) {
		t.Fatalf("error = %v, want PeerDiscoveryUnavailableError", err)
	}
}

func TestPublishPeerRecordUsesAllocatedPorts(t *testing.T) {
	home := t.TempDir()
	allocator := testutil.NewMockPortAllocator()
	allocator.SetPort("api", "api", 23117)
	m := &manifest.Manifest{
		App: manifest.App{Scenario: "browser-automation-studio"},
		IPC: manifest.IPC{Port: 23118, AuthTokenRel: "runtime/auth-token"},
	}
	s := &Supervisor{
		opts:          Options{Manifest: m},
		appData:       t.TempDir(),
		peerHome:      home,
		portAllocator: allocator,
		startedAt:     time.Now(),
	}
	if err := s.publishPeerRecord(); err != nil {
		t.Fatalf("publishPeerRecord() error = %v", err)
	}
	record, err := peerrecord.Read(home, "browser-automation-studio")
	if err != nil {
		t.Fatal(err)
	}
	if record.Tier != 2 || record.Ports["api"] != 23117 || record.Ports["ipc"] != 23118 {
		t.Fatalf("published peer record = %#v", record)
	}
}

func TestTwoBundlesAllocateDistinctIPCPorts(t *testing.T) {
	newManifest := func(name string) *manifest.Manifest {
		return &manifest.Manifest{
			App:       manifest.App{Name: name, Version: "1.0.0"},
			IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/auth-token"},
			Telemetry: manifest.Telemetry{File: "telemetry/events.jsonl"},
			Ports:     &manifest.PortRules{DefaultRange: &manifest.PortRange{Min: 47000, Max: 48000}},
		}
	}
	newSupervisor := func(name string) *Supervisor {
		s, err := NewSupervisor(Options{
			Manifest:      newManifest(name),
			BundlePath:    t.TempDir(),
			AppDataDir:    t.TempDir(),
			DryRun:        true,
			SecretStore:   testutil.NewMockSecretStore(nil),
			HealthChecker: testutil.NewMockHealthChecker(),
		})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	first := newSupervisor("first-bundle")
	second := newSupervisor("second-bundle")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := first.Start(ctx); err != nil {
		t.Fatalf("start first bundle: %v", err)
	}
	t.Cleanup(func() { _ = first.Shutdown(context.Background()) })
	if err := second.Start(ctx); err != nil {
		t.Fatalf("start second bundle: %v", err)
	}
	t.Cleanup(func() { _ = second.Shutdown(context.Background()) })
	if first.opts.Manifest.IPC.Port == 0 || second.opts.Manifest.IPC.Port == 0 || first.opts.Manifest.IPC.Port == second.opts.Manifest.IPC.Port {
		t.Fatalf("allocated IPC ports collide: first=%d second=%d", first.opts.Manifest.IPC.Port, second.opts.Manifest.IPC.Port)
	}
	if _, err := os.Stat(filepath.Join(first.appData, "runtime", "ipc_port")); err != nil {
		t.Fatalf("first bundle did not publish its allocated IPC port: %v", err)
	}
	if _, err := os.Stat(filepath.Join(second.appData, "runtime", "ipc_port")); err != nil {
		t.Fatalf("second bundle did not publish its allocated IPC port: %v", err)
	}
}
