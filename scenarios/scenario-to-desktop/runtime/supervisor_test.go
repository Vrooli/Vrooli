package bundleruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
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

// ipcPortPublishFailureFS fails exactly the ipc_port write so the supervisor's
// publish path can be exercised without disturbing any other file operation.
type ipcPortPublishFailureFS struct {
	*testutil.MockFileSystem
	failSuffix string
}

func (f *ipcPortPublishFailureFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if strings.HasSuffix(path, f.failSuffix) {
		return errors.New("disk full")
	}
	return f.MockFileSystem.WriteFile(path, data, perm)
}

// The desktop shell and vrooli-shim learn the control API address only from
// runtime/ipc_port. A runtime that binds but cannot publish is unreachable, so
// Start must fail rather than report success.
func TestStart_FailsWhenIpcPortCannotBePublished(t *testing.T) {
	tmp := t.TempDir()
	appDataDir := filepath.Join(tmp, "appdata")
	mockFS := &ipcPortPublishFailureFS{
		MockFileSystem: testutil.NewMockFileSystem(),
		failSuffix:     filepath.Join("runtime", "ipc_port"),
	}

	m := &manifest.Manifest{
		App:       manifest.App{Name: "test-app", Version: "1.0.0"},
		IPC:       manifest.IPC{Host: "127.0.0.1", Port: 0, AuthTokenRel: "runtime/auth-token"},
		Telemetry: manifest.Telemetry{File: "telemetry.jsonl"},
	}

	s, err := NewSupervisor(Options{
		Manifest:      m,
		BundlePath:    tmp,
		AppDataDir:    appDataDir,
		Clock:         testutil.NewMockClock(time.Now()),
		FileSystem:    mockFS,
		SecretStore:   testutil.NewMockSecretStore(nil),
		PortAllocator: testutil.NewMockPortAllocator(),
		HealthChecker: testutil.NewMockHealthChecker(),
	})
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = s.Start(ctx)
	if err == nil {
		_ = s.Shutdown(context.Background())
		t.Fatal("Start() should fail when the IPC port cannot be published")
	}
	if !strings.Contains(err.Error(), "publish IPC port") {
		t.Fatalf("Start() error = %v, want it to name the IPC port publish failure", err)
	}
	if s.started {
		t.Fatal("Start() must not mark the supervisor started when the port was never published")
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

func TestPublishPeerRecordUsesAllocatedPorts(t *testing.T) {
	home := t.TempDir()
	allocator := testutil.NewMockPortAllocator()
	allocator.SetPort("api", "api", 23117)
	allocator.SetPort("landing-page-business-suite--api", "api", 23119)
	m := &manifest.Manifest{
		App:      manifest.App{Scenario: "browser-automation-studio", Version: "1.2.3"},
		IPC:      manifest.IPC{Port: 23118, AuthTokenRel: "runtime/auth-token"},
		Services: []manifest.Service{{ID: "api"}, {ID: "landing-page-business-suite--api"}},
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
	if record.Tier != 2 || record.Version != "1.2.3" || record.Ports["api"] != 23117 || record.Ports["ipc"] != 23118 || len(record.Ports) != 2 || len(record.Provides) != 1 || record.Provides[0] != "api" {
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

func TestPublishPeerRecordDeclaresOnlyRootOwnerFeatures(t *testing.T) {
	home := t.TempDir()
	allocator := testutil.NewMockPortAllocator()
	allocator.SetPort("api", "api", 23117)
	allocator.SetPort("component-api", "component", 23119)
	allocator.SetPort("unlisted", "unknown", 23120)
	m := &manifest.Manifest{
		App:      manifest.App{Scenario: "audio-tools", Version: "9.9.9"},
		Features: map[string]manifest.FeatureExport{"audio.processing": {OwnerVersion: "0.0.1", Service: "api", Port: "api"}},
		Services: []manifest.Service{
			{ID: "api", Health: manifest.HealthCheck{Type: "http"}, Readiness: manifest.ReadinessCheck{Type: "port"}, Ports: &manifest.ServicePorts{Requested: []manifest.PortRequest{{Name: "api"}}}},
			{ID: "component-api", ComponentSlice: "usage", Health: manifest.HealthCheck{Type: "http"}, Readiness: manifest.ReadinessCheck{Type: "port"}, Ports: &manifest.ServicePorts{Requested: []manifest.PortRequest{{Name: "component"}}}},
		},
	}
	s := &Supervisor{opts: Options{Manifest: m}, appData: t.TempDir(), peerHome: home, portAllocator: allocator, startedAt: time.Now()}
	if err := s.publishPeerRecord(); err != nil {
		t.Fatal(err)
	}
	record, err := peerrecord.Read(home, "audio-tools")
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Features) != 1 || record.Features["audio.processing"].OwnerVersion != "0.0.1" || record.Features["audio.processing"].Port != "api" {
		t.Fatalf("wrong owner feature: %#v", record.Features)
	}
	if record.Ports["component"] != 0 || record.Ports["unknown"] != 0 {
		t.Fatalf("non-root ports leaked: %#v", record.Ports)
	}
	m.Features["usage.live"] = manifest.FeatureExport{OwnerVersion: "0.0.1", Service: "component-api", Port: "component"}
	if err := s.publishPeerRecord(); err == nil {
		t.Fatal("component feature published as a full peer")
	}
}

func TestSM11FullFeatureRejectsChangedProbeIdentity(t *testing.T) {
	for _, change := range []string{"owner_version", "feature_withdrawn", "instance_replaced", "route_moved", "record_removed"} {
		t.Run(change, func(t *testing.T) {
			home := t.TempDir()
			record := peerrecord.Record{Scenario: "audio-tools", Version: "1.2.0", OwnerPID: os.Getpid(), StartedAt: time.Now(), Ports: map[string]int{}, Features: map[string]peerrecord.Feature{"audio.processing": {OwnerVersion: "1.2.0", Port: "api"}}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" {
					http.NotFound(w, r)
					return
				}
				switch change {
				case "owner_version":
					record.Features["audio.processing"] = peerrecord.Feature{OwnerVersion: "2.0.0", Port: "api"}
				case "feature_withdrawn":
					delete(record.Features, "audio.processing")
				case "instance_replaced":
					record.StartedAt = record.StartedAt.Add(time.Second)
				case "route_moved":
					record.Ports["api"]++
				case "record_removed":
					if err := peerrecord.Remove(home, "audio-tools", os.Getpid()); err != nil {
						t.Error(err)
					}
					w.WriteHeader(http.StatusOK)
					return
				}
				if err := peerrecord.Write(home, record); err != nil {
					t.Error(err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			port, err := strconv.Atoi(strings.TrimPrefix(server.URL, "http://127.0.0.1:"))
			if err != nil {
				t.Fatal(err)
			}
			record.Ports["api"] = port
			if err := peerrecord.Write(home, record); err != nil {
				t.Fatal(err)
			}
			s := &Supervisor{opts: Options{Manifest: &manifest.Manifest{App: manifest.App{Scenario: "web-console"}, Peers: []manifest.Peer{{Scenario: "audio-tools", VersionRange: ">=1 <2"}}}}, peerHome: home}
			selected, err := s.resolveFullFeature(context.Background(), nil, "audio-tools", "audio.processing", ">=1 <2")
			if err == nil || selected.Source != "" {
				t.Fatalf("changed provider returned after probe: selected=%+v error=%v", selected, err)
			}
		})
	}
}

// These states describe observations at a call boundary. Registry publication
// precedes readiness; only the feature resolver's successful probe permits
// stand-down. Bare discovery deliberately does not promise health.
func TestSM11SupervisorTopologyEventMatrix(t *testing.T) {
	states := []string{"closed", "unready", "full", "out_of_range"}
	events := []string{"lookup_again", "publish_ready", "publish_incompatible", "lose_health", "close"}
	next := [][]int{{0, 2, 3, 0, 0}, {1, 2, 3, 1, 0}, {2, 2, 3, 1, 0}, {3, 2, 3, 3, 0}}
	for _, local := range []bool{false, true} {
		for from := range states {
			for event := range events {
				t.Run(fmt.Sprintf("embedded_%t/%s/%s", local, states[from], events[event]), func(t *testing.T) {
					s, publish, healthCode := newSM11SelectionFixture(t, local)
					current := from
					applyState := func(state int) {
						t.Helper()
						healthCode.Store(200)
						switch state {
						case 0:
							publish("")
						case 1:
							healthCode.Store(503)
							publish("1.2.0")
						case 2:
							publish("1.2.0")
						case 3:
							publish("2.0.0")
						}
					}
					assert := func(state int) {
						t.Helper()
						selected, err := s.ResolveFeature(context.Background(), "audio-tools", "audio.processing", ">=1 <2")
						want := ""
						if state == 2 {
							want = "full_app"
						} else if local {
							want = "embedded"
						}
						if selected.Source != want || (want != "" && err != nil) || (want == "" && err == nil) {
							t.Fatalf("state%s selected=%+v error=%v want%s", states[state], selected, err, want)
						}
						if state == 2 && local {
							status, _ := s.getStatus("audio-tools--api")
							if !status.Skipped || s.getProc("audio-tools--api") != nil {
								t.Fatal("full selection did not stand down embedded owner")
							}
						}
					}
					applyState(current)
					assert(current)
					switch event {
					case 0:
					case 1:
						publish("1.2.0")
						healthCode.Store(200)
					case 2:
						publish("2.0.0")
					case 3:
						healthCode.Store(503)
					case 4:
						publish("")
					}
					current = next[from][event]
					assert(current)
				})
			}
		}
	}
}

func TestSM11SupervisorPublicationFallbackReplay(t *testing.T) {
	s, publish, healthCode := newSM11SelectionFixture(t, true)
	assert := func(label, want string) {
		t.Helper()
		selected, err := s.ResolveFeature(context.Background(), "audio-tools", "audio.processing", ">=1 <2")
		if err != nil || selected.Source != want {
			t.Fatalf("%s selected=%+v error=%v want%s", label, selected, err, want)
		}
		t.Logf("%s -> %s", label, selected.Source)
	}
	assert("embedded available", "embedded")
	healthCode.Store(503)
	publish("1.2.0")
	assert("published before ready", "embedded")
	healthCode.Store(200)
	assert("full ready and local stand-down", "full_app")
	assert("repeat lookup is stable", "full_app")
	healthCode.Store(503)
	assert("health lost, local fallback", "embedded")
	healthCode.Store(200)
	assert("full recovers", "full_app")
	publish("")
	assert("full closed, local fallback", "embedded")
	checker := s.healthChecker.(*testutil.MockHealthChecker)
	checker.SetWaitReadinessErr(errors.New("embedded not ready"))
	if selected, err := s.ResolveFeature(context.Background(), "audio-tools", "audio.processing", ">=1 <2"); err == nil || selected.Source != "" {
		t.Fatalf("unready fallback selected: %+v %v", selected, err)
	}
	checker.SetWaitReadinessErr(nil)
	assert("local readiness recovers", "embedded")
}

func newSM11SelectionFixture(t *testing.T, embedded bool) (*Supervisor, func(string), *atomic.Int32) {
	t.Helper()
	home := t.TempDir()
	healthCode := &atomic.Int32{}
	healthCode.Store(200)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(int(healthCode.Load()))
	}))
	t.Cleanup(endpoint.Close)
	port, err := strconv.Atoi(strings.TrimPrefix(endpoint.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatal(err)
	}
	ports := testutil.NewMockPortAllocator()
	ports.SetPort("audio-tools--api", "api", 24569)
	m := &manifest.Manifest{App: manifest.App{Scenario: "web-console"}, Peers: []manifest.Peer{{Scenario: "audio-tools", VersionRange: ">=1 <2"}}}
	if embedded {
		m.Services = []manifest.Service{{ID: "audio-tools--api", Type: "api-binary"}}
		m.Peers[0].BundlePolicy = "embed"
		m.Peers[0].Features = map[string]manifest.FeatureExport{"audio.processing": {OwnerVersion: "1.2.0", Service: "audio-tools--api", Port: "api"}}
	}
	s, err := NewSupervisor(Options{Manifest: m, BundlePath: t.TempDir(), AppDataDir: t.TempDir(), PeerHomeDir: home, DryRun: true, PortAllocator: ports, SecretStore: testutil.NewMockSecretStore(nil), HealthChecker: testutil.NewMockHealthChecker()})
	if err != nil {
		t.Fatal(err)
	}
	s.runtimeCtx = context.Background()
	if embedded {
		s.setProc("audio-tools--api", &serviceProcess{started: time.Now()})
		s.setStatus("audio-tools--api", ServiceStatus{Ready: true})
	}
	publisherPorts := testutil.NewMockPortAllocator()
	publisherPorts.SetPort("api", "api", port)
	publisher := &Supervisor{opts: Options{Manifest: &manifest.Manifest{App: manifest.App{Scenario: "audio-tools", Version: "1.2.0"}, Services: []manifest.Service{{ID: "api", Health: manifest.HealthCheck{Type: "http"}, Readiness: manifest.ReadinessCheck{Type: "port"}, Ports: &manifest.ServicePorts{Requested: []manifest.PortRequest{{Name: "api"}}}}}, Features: map[string]manifest.FeatureExport{"audio.processing": {OwnerVersion: "1.2.0", Service: "api", Port: "api"}}}}, appData: t.TempDir(), peerHome: home, portAllocator: publisherPorts, startedAt: time.Now()}
	publish := func(version string) {
		t.Helper()
		if version == "" {
			if err := peerrecord.Remove(home, "audio-tools", os.Getpid()); err != nil {
				t.Fatal(err)
			}
			return
		}
		publisher.opts.Manifest.App.Version = version
		feature := publisher.opts.Manifest.Features["audio.processing"]
		feature.OwnerVersion = version
		publisher.opts.Manifest.Features["audio.processing"] = feature
		if err := publisher.publishPeerRecord(); err != nil {
			t.Fatal(err)
		}
	}
	return s, publish, healthCode
}

func TestSM11EmbeddedFeatureUsesProgramBindingRange(t *testing.T) {
	for _, tc := range []struct {
		name, ordinary, binding string
		want                    bool
	}{{"binding_restricts", ">=1 <2", ">=2 <3", false}, {"binding_overrides", ">=2 <3", ">=1 <2", true}} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newSM11SelectionFixture(t, true)
			s.opts.Manifest.Peers[0].VersionRange = tc.ordinary
			s.opts.Manifest.ProgramBindingPeers = []manifest.Peer{{Scenario: "audio-tools", VersionRange: tc.binding}}
			selected, err := s.ResolveFeature(context.Background(), "audio-tools", "audio.processing", "")
			if tc.want && (err != nil || selected.Source != "embedded") {
				t.Fatalf("eligible binding not used: %+v %v", selected, err)
			}
			if !tc.want && (err == nil || selected.Source != "") {
				t.Fatalf("ineligible binding bypassed: %+v %v", selected, err)
			}
		})
	}
}
