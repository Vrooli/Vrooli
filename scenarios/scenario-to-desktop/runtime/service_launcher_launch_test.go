package bundleruntime

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
	"scenario-to-desktop-runtime/testutil"
)

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

// =============================================================================
// resolveUIDistRoot Tests
// =============================================================================

func TestResolveUIDistRoot_ExplicitDistRoot(t *testing.T) {
	tmp := t.TempDir()
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
		},
	}

	svc := manifest.Service{
		ID:       "ui",
		Type:     "ui-bundle",
		DistRoot: "custom/dist/path",
		Assets: []manifest.Asset{
			{Path: "ui/dist/assets/app.js"},
			{Path: "ui/dist/index.html"},
		},
	}

	got, err := s.resolveUIDistRoot(svc)
	if err != nil {
		t.Fatalf("resolveUIDistRoot() error = %v", err)
	}

	want := filepath.Join(tmp, "custom/dist/path")
	if got != want {
		t.Errorf("resolveUIDistRoot() = %q, want %q", got, want)
	}
}

func TestResolveUIDistRoot_FindIndexHtml(t *testing.T) {
	tmp := t.TempDir()
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
		},
	}

	// This simulates the bug scenario: first asset is in assets/ subdirectory,
	// but index.html is at the dist root. The function should find index.html
	// and use its parent directory.
	svc := manifest.Service{
		ID:   "ui",
		Type: "ui-bundle",
		Assets: []manifest.Asset{
			{Path: "ui/dist/assets/abap-BdImnpbu.js"},  // First asset in subdirectory
			{Path: "ui/dist/assets/app-xyz123.js"},    // Another asset
			{Path: "ui/dist/assets/styles-abc456.css"}, // Another asset
			{Path: "ui/dist/index.html"},               // Entry point at dist root
		},
	}

	got, err := s.resolveUIDistRoot(svc)
	if err != nil {
		t.Fatalf("resolveUIDistRoot() error = %v", err)
	}

	want := filepath.Join(tmp, "ui/dist")
	if got != want {
		t.Errorf("resolveUIDistRoot() = %q, want %q", got, want)
	}
}

func TestResolveUIDistRoot_IndexHtmlAtRoot(t *testing.T) {
	tmp := t.TempDir()
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
		},
	}

	// Simple case: index.html is the first asset
	svc := manifest.Service{
		ID:   "ui",
		Type: "ui-bundle",
		Assets: []manifest.Asset{
			{Path: "dist/index.html"},
			{Path: "dist/assets/main.js"},
		},
	}

	got, err := s.resolveUIDistRoot(svc)
	if err != nil {
		t.Fatalf("resolveUIDistRoot() error = %v", err)
	}

	want := filepath.Join(tmp, "dist")
	if got != want {
		t.Errorf("resolveUIDistRoot() = %q, want %q", got, want)
	}
}

func TestResolveUIDistRoot_NoIndexHtmlNoDistRoot(t *testing.T) {
	tmp := t.TempDir()
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
		},
	}

	// No index.html and no dist_root - should error
	svc := manifest.Service{
		ID:   "ui",
		Type: "ui-bundle",
		Assets: []manifest.Asset{
			{Path: "ui/dist/assets/app.js"},
			{Path: "ui/dist/assets/styles.css"},
		},
	}

	_, err := s.resolveUIDistRoot(svc)
	if err == nil {
		t.Fatal("resolveUIDistRoot() expected error when no index.html and no dist_root")
	}

	// Error should mention both options
	errMsg := err.Error()
	if !strings.Contains(errMsg, "dist_root") {
		t.Errorf("error should mention dist_root, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "index.html") {
		t.Errorf("error should mention index.html, got: %s", errMsg)
	}
}

func TestResolveUIDistRoot_EmptyAssets(t *testing.T) {
	tmp := t.TempDir()
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
		},
	}

	// No assets at all
	svc := manifest.Service{
		ID:     "ui",
		Type:   "ui-bundle",
		Assets: []manifest.Asset{},
	}

	_, err := s.resolveUIDistRoot(svc)
	if err == nil {
		t.Fatal("resolveUIDistRoot() expected error when assets is empty")
	}
}

func TestResolveUIDistRoot_ExplicitDistRootOverridesIndexHtml(t *testing.T) {
	tmp := t.TempDir()
	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
		},
	}

	// Both dist_root and index.html present - dist_root takes precedence
	svc := manifest.Service{
		ID:       "ui",
		Type:     "ui-bundle",
		DistRoot: "explicit/path",
		Assets: []manifest.Asset{
			{Path: "different/path/index.html"},
		},
	}

	got, err := s.resolveUIDistRoot(svc)
	if err != nil {
		t.Fatalf("resolveUIDistRoot() error = %v", err)
	}

	want := filepath.Join(tmp, "explicit/path")
	if got != want {
		t.Errorf("resolveUIDistRoot() = %q, want %q (explicit dist_root should take precedence)", got, want)
	}
}
