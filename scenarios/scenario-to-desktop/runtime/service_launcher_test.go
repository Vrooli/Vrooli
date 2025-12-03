package bundleruntime

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"scenario-to-desktop-runtime/assets"
	"scenario-to-desktop-runtime/env"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
	"scenario-to-desktop-runtime/testutil"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want *int
	}{
		{"nil error", nil, intPtr(0)},
		{"generic error", errors.New("failed"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exitCode(tt.err)
			if tt.want == nil {
				if got != nil {
					t.Errorf("exitCode() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("exitCode() = nil, want %v", *tt.want)
				} else if *got != *tt.want {
					t.Errorf("exitCode() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestStartService_DryRun(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			DryRun: true,
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{
						ID: "api",
						Binaries: map[string]manifest.Binary{
							"linux_amd64": {Path: "bin/api"},
						},
					},
				},
			},
		},
		serviceStatus: make(map[string]ServiceStatus),
	}

	svc := manifest.Service{
		ID: "api",
		Binaries: map[string]manifest.Binary{
			"linux_amd64": {Path: "bin/api"},
		},
	}
	ctx := context.Background()

	if err := s.startService(ctx, svc); err != nil {
		t.Fatalf("startService() error = %v", err)
	}

	status := s.serviceStatus["api"]
	if !status.Ready {
		t.Error("startService() dry-run should set Ready=true")
	}
	if status.Message != "dry-run" {
		t.Errorf("startService() dry-run message = %q, want %q", status.Message, "dry-run")
	}
}

func TestStartService_NoBinary(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{},
			},
		},
		serviceStatus: make(map[string]ServiceStatus),
	}

	svc := manifest.Service{ID: "api", Binaries: map[string]manifest.Binary{}} // No matching binary
	ctx := context.Background()

	err := s.startService(ctx, svc)
	if err == nil {
		t.Error("startService() expected error for missing binary")
	}
}

func TestPrepareServiceDirs(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()

	s := &Supervisor{
		fs:      mockFS,
		appData: tmp,
	}

	svc := manifest.Service{
		ID:       "api",
		DataDirs: []string{"data", "cache", "logs"},
	}

	if err := s.prepareServiceDirs(svc); err != nil {
		t.Fatalf("prepareServiceDirs() error = %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		tmp + "/data",
		tmp + "/cache",
		tmp + "/logs",
	}
	for _, dir := range expectedDirs {
		if !mockFS.Dirs[dir] {
			t.Errorf("prepareServiceDirs() didn't create %q", dir)
		}
	}
}

func TestPrepareServiceDirs_WithLogDir(t *testing.T) {
	tmp := t.TempDir()

	s := &Supervisor{
		fs:      RealFileSystem{},
		appData: tmp,
	}

	svc := manifest.Service{
		ID:     "api",
		LogDir: "logs/api.log",
	}

	if err := s.prepareServiceDirs(svc); err != nil {
		t.Fatalf("prepareServiceDirs() error = %v", err)
	}

	// Verify log file was touched
	logPath := tmp + "/logs/api.log"
	if _, err := s.fs.Stat(logPath); err != nil {
		t.Errorf("prepareServiceDirs() didn't create log file: %v", err)
	}
}

func TestLogWriter(t *testing.T) {
	tmp := t.TempDir()

	s := &Supervisor{
		fs:      RealFileSystem{},
		appData: tmp,
	}

	tests := []struct {
		name      string
		logDir    string
		wantNil   bool
		wantPath  string
		wantError bool
		setupDir  bool
	}{
		{"empty log dir", "", true, "", false, false},
		{"valid log dir", "logs/api.log", false, tmp + "/logs/api.log", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parent directory if needed
			if tt.setupDir && tt.logDir != "" {
				logPath := tmp + "/" + tt.logDir
				if err := s.fs.MkdirAll(logPath[:len(logPath)-len("/api.log")], 0o755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			}

			svc := manifest.Service{ID: "api", LogDir: tt.logDir}
			writer, path, err := s.logWriter(svc)

			if (err != nil) != tt.wantError {
				t.Errorf("logWriter() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantNil && writer != nil {
				t.Error("logWriter() returned non-nil writer, expected nil")
			}

			if !tt.wantNil && writer == nil {
				t.Error("logWriter() returned nil writer")
			}

			if path != tt.wantPath {
				t.Errorf("logWriter() path = %q, want %q", path, tt.wantPath)
			}

			if writer != nil {
				writer.Close()
			}
		})
	}
}

func TestGracefulStop(t *testing.T) {
	mockClock := testutil.NewMockClock(time.Now())
	s := &Supervisor{clock: mockClock}

	proc := testutil.NewMockProcess(123)
	svcProc := &serviceProcess{proc: proc}

	ctx := context.Background()

	// Simulate process exiting after signal - exit before calling gracefulStop
	// to simulate a process that exits immediately
	proc.Exit(nil)

	s.gracefulStop(ctx, svcProc)

	if !proc.Signaled() {
		t.Error("gracefulStop() didn't signal process")
	}
}

func TestGracefulStop_Timeout(t *testing.T) {
	mockClock := testutil.NewMockClock(time.Now())
	s := &Supervisor{clock: mockClock}

	proc := testutil.NewMockProcess(123)
	svcProc := &serviceProcess{proc: proc}

	ctx := context.Background()

	// Don't exit process, so it doesn't exit gracefully
	s.gracefulStop(ctx, svcProc)

	if !proc.Signaled() {
		t.Error("gracefulStop() didn't signal process")
	}
	if !proc.Killed() {
		t.Error("gracefulStop() didn't kill process after timeout")
	}
}

func TestServiceProcess(t *testing.T) {
	proc := testutil.NewMockProcess(12345)
	svcProc := &serviceProcess{
		proc:    proc,
		logPath: "/var/log/api.log",
		service: manifest.Service{ID: "api"},
		started: time.Now(),
	}

	if svcProc.proc.Pid() != 12345 {
		t.Errorf("serviceProcess.proc.Pid() = %d, want 12345", svcProc.proc.Pid())
	}
	if svcProc.logPath != "/var/log/api.log" {
		t.Errorf("serviceProcess.logPath = %q, want %q", svcProc.logPath, "/var/log/api.log")
	}
}

func TestLaunchServices_EmptyManifest(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{},
			},
		},
	}

	ctx := context.Background()
	err := s.launchServices(ctx)
	if err != nil {
		t.Errorf("launchServices() error = %v for empty services", err)
	}
}

func TestLaunchServices_ContextCanceled(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{ID: "api"},
				},
			},
		},
		serviceStatus: make(map[string]ServiceStatus),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.launchServices(ctx)
	if err == nil {
		t.Error("launchServices() expected error for canceled context")
	}
}

func TestStopServices_NoProcesses(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{ID: "api"},
				},
			},
		},
		procs: make(map[string]*serviceProcess),
		clock: testutil.NewMockClock(time.Now()),
	}

	ctx := context.Background()

	// Should not panic
	s.stopServices(ctx)
}

// =============================================================================
// startService Comprehensive Tests
// =============================================================================

// platformBinaryKey returns the correct binary key for the current platform.
func platformBinaryKey() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return runtime.GOOS + "-" + arch
}

// testSupervisorConfig holds configuration for creating a test supervisor.
type testSupervisorConfig struct {
	manifest      *manifest.Manifest
	bundlePath    string
	appData       string
	procShouldErr bool
}

// newTestSupervisor creates a fully initialized Supervisor for testing startService.
func newTestSupervisor(t *testing.T, cfg testSupervisorConfig) *Supervisor {
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockProcRunner := testutil.NewMockProcessRunner()
	if cfg.procShouldErr {
		mockProcRunner.SetShouldFail(true)
	}
	mockHealthChecker := testutil.NewMockHealthChecker()
	mockSecretStore := testutil.NewMockSecretStore(cfg.manifest)
	mockPortAllocator := testutil.NewMockPortAllocator()
	mockEnvReader := testutil.NewMockEnvReader()

	// Set up some default ports
	for _, svc := range cfg.manifest.Services {
		mockPortAllocator.SetPort(svc.ID, "http", 8080)
	}

	telem := telemetry.NopRecorder{}
	gpuStatus := gpu.Status{Available: false, Method: "mock", Reason: "test"}

	return &Supervisor{
		opts: Options{
			Manifest:   cfg.manifest,
			BundlePath: cfg.bundlePath,
		},
		appData:       cfg.appData,
		fs:            mockFS,
		clock:         mockClock,
		procRunner:    mockProcRunner,
		healthChecker: mockHealthChecker,
		secretStore:   mockSecretStore,
		portAllocator: mockPortAllocator,
		envReader:     mockEnvReader,
		telemetry:     telem,
		envRenderer:   env.NewRenderer(cfg.appData, cfg.bundlePath, mockPortAllocator, mockEnvReader),
		gpuApplier:    gpu.NewApplier(gpuStatus, telem),
		assetVerifier: assets.NewVerifier(cfg.bundlePath, mockFS, telem),
		gpuStatus:     gpuStatus,
		serviceStatus: make(map[string]ServiceStatus),
		procs:         make(map[string]*serviceProcess),
	}
}

func TestStartService_HappyPath(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "api",
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/api"},
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
	})

	svc := m.Services[0]

	ctx := context.Background()
	err := s.startService(ctx, svc)
	if err != nil {
		t.Fatalf("startService() error = %v", err)
	}

	// Verify process was started
	pr := s.procRunner.(*testutil.MockProcessRunner)
	startedCmds := pr.StartedCmds()
	if len(startedCmds) == 0 {
		t.Error("startService() should start a process")
	}

	// Verify status was set
	status := s.serviceStatus["api"]
	if status.Message != "starting" {
		t.Errorf("startService() status message = %q, want %q", status.Message, "starting")
	}

	// Verify proc was tracked
	if s.procs["api"] == nil {
		t.Error("startService() should track the process")
	}
}

func TestStartService_WithEnvironment(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "api",
				Binaries: map[string]manifest.Binary{
					binKey: {
						Path: "bin/api",
						Env: map[string]string{
							"LOG_LEVEL": "debug",
						},
					},
				},
				Env: map[string]string{
					"APP_NAME": "test",
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
	})

	svc := m.Services[0]

	ctx := context.Background()
	err := s.startService(ctx, svc)
	if err != nil {
		t.Fatalf("startService() error = %v", err)
	}

	// Process should have been started with env
	pr := s.procRunner.(*testutil.MockProcessRunner)
	startedCmds := pr.StartedCmds()
	if len(startedCmds) == 0 {
		t.Error("startService() should start process with environment")
	}
}

func TestStartService_WithWorkingDirectory(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "api",
				Binaries: map[string]manifest.Binary{
					binKey: {
						Path: "bin/api",
						CWD:  "services/api",
					},
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
	})

	svc := m.Services[0]

	ctx := context.Background()
	err := s.startService(ctx, svc)
	if err != nil {
		t.Fatalf("startService() error = %v", err)
	}
}

func TestStartService_ProcessStartFails(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "api",
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/api"},
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:      m,
		bundlePath:    tmp,
		appData:       tmp,
		procShouldErr: true,
	})

	svc := m.Services[0]

	ctx := context.Background()
	err := s.startService(ctx, svc)
	if err == nil {
		t.Fatal("startService() expected error when process start fails")
	}
}

func TestStartService_WithDataDirs(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID:       "api",
				DataDirs: []string{"data", "cache"},
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/api"},
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
	})

	svc := m.Services[0]

	ctx := context.Background()
	err := s.startService(ctx, svc)
	if err != nil {
		t.Fatalf("startService() error = %v", err)
	}

	// Verify data directories were created
	fs := s.fs.(*testutil.MockFileSystem)
	if !fs.Dirs[filepath.Join(tmp, "data")] {
		t.Error("startService() should create data directory")
	}
	if !fs.Dirs[filepath.Join(tmp, "cache")] {
		t.Error("startService() should create cache directory")
	}
}

func TestStartService_WithArgs(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "api",
				Binaries: map[string]manifest.Binary{
					binKey: {
						Path: "bin/api",
						Args: []string{"--port", "{{port:api:http}}", "--config", "config.yaml"},
					},
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
	})

	svc := m.Services[0]

	ctx := context.Background()
	err := s.startService(ctx, svc)
	if err != nil {
		t.Fatalf("startService() error = %v", err)
	}

	// Process was started
	pr := s.procRunner.(*testutil.MockProcessRunner)
	startedCmds := pr.StartedCmds()
	if len(startedCmds) == 0 {
		t.Error("startService() should start process with args")
	}
}

func TestLaunchServices_DependencyOrder(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID:           "api",
				Dependencies: []string{"db"},
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/api"},
				},
			},
			{
				ID: "db",
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/db"},
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
	})

	ctx := context.Background()
	err := s.launchServices(ctx)
	if err != nil {
		t.Fatalf("launchServices() error = %v", err)
	}

	// Both services should have been started (db first, then api)
	pr := s.procRunner.(*testutil.MockProcessRunner)
	startedCmds := pr.StartedCmds()
	if len(startedCmds) < 2 {
		t.Errorf("launchServices() started %d services, want 2", len(startedCmds))
	}

	// Verify db was started before api
	dbIdx, apiIdx := -1, -1
	for i, cmd := range startedCmds {
		if filepath.Base(cmd) == "db" {
			dbIdx = i
		}
		if filepath.Base(cmd) == "api" {
			apiIdx = i
		}
	}
	if dbIdx != -1 && apiIdx != -1 && dbIdx > apiIdx {
		t.Error("launchServices() should start db before api")
	}
}

func TestStopServices_GracefulShutdown(t *testing.T) {
	mockClock := testutil.NewMockClock(time.Now())

	proc1 := testutil.NewMockProcess(1)
	proc2 := testutil.NewMockProcess(2)

	// Make processes exit when signaled
	proc1.Exit(nil)
	proc2.Exit(nil)

	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{ID: "api", Dependencies: []string{"db"}},
					{ID: "db"},
				},
			},
		},
		procs: map[string]*serviceProcess{
			"api": {proc: proc1, service: manifest.Service{ID: "api"}},
			"db":  {proc: proc2, service: manifest.Service{ID: "db"}},
		},
		clock: mockClock,
	}

	ctx := context.Background()
	s.stopServices(ctx)

	// Both processes should have been signaled
	if !proc1.Signaled() {
		t.Error("stopServices() should signal api process")
	}
	if !proc2.Signaled() {
		t.Error("stopServices() should signal db process")
	}
}

func TestStartServicesAsync_OnlyStartsOnce(t *testing.T) {
	tmp := t.TempDir()

	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(time.Now())
	mockHealthChecker := testutil.NewMockHealthChecker()

	m := &manifest.Manifest{
		App:      manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{},
	}

	s := &Supervisor{
		opts: Options{
			Manifest:   m,
			BundlePath: tmp,
		},
		appData:         tmp,
		fs:              mockFS,
		clock:           mockClock,
		healthChecker:   mockHealthChecker,
		telemetry:       telemetry.NopRecorder{},
		servicesStarted: false,
		runtimeCtx:      context.Background(),
	}

	// First call should start services
	s.startServicesAsync()
	if !s.servicesStarted {
		t.Error("startServicesAsync() should set servicesStarted=true")
	}

	// Second call should be a no-op
	s.startServicesAsync() // Should not panic or start again
}
