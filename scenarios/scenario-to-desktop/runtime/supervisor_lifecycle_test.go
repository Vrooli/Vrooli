package bundleruntime

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
	"scenario-to-desktop-runtime/testutil"
)

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
