package bundleruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/assets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/gpu"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
)

// =============================================================================
// startUIBundleService Readiness Tests
// =============================================================================

func TestStartUIBundleService_ReadinessSuccess(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle")
	appData := filepath.Join(tmp, "appdata")

	if err := os.MkdirAll(filepath.Join(bundlePath, "ui", "dist"), 0o755); err != nil {
		t.Fatalf("failed to create ui dist dir: %v", err)
	}
	indexPath := filepath.Join(bundlePath, "ui", "dist", "index.html")
	if err := os.WriteFile(indexPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write ui index: %v", err)
	}

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID:   "ui",
				Type: "ui-bundle",
				Assets: []manifest.Asset{
					{Path: filepath.ToSlash(filepath.Join("ui", "dist", "index.html"))},
				},
				LogDir: filepath.ToSlash(filepath.Join("logs", "ui.log")),
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "ui", Range: manifest.PortRange{Min: 24100, Max: 24200}},
					},
				},
				Health: manifest.HealthCheck{
					Type:     "http",
					Path:     "/health",
					PortName: "ui",
				},
				Readiness: manifest.ReadinessCheck{
					Type:     "health_success",
					PortName: "ui",
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: bundlePath,
		appData:    appData,
	})

	realFS := RealFileSystem{}
	s.fs = realFS
	s.assetVerifier = assets.NewVerifier(bundlePath, realFS, s.telemetry)

	if ports, ok := s.portAllocator.(*testutil.MockPortAllocator); ok {
		ports.SetPort("ui", "ui", 0)
	}

	if err := s.startUIBundleService(context.Background(), m.Services[0]); err != nil {
		t.Fatalf("startUIBundleService failed: %v", err)
	}
	s.wg.Wait()
	defer s.stopServices(context.Background())

	status := s.ServiceStatuses()["ui"]
	if !status.Ready {
		t.Errorf("ui service ready = false, want true")
	}
	if status.Message != "ready" {
		t.Errorf("ui service message = %q, want %q", status.Message, "ready")
	}
}

func TestStartUIBundleService_ReadinessFailure(t *testing.T) {
	tmp := t.TempDir()
	bundlePath := filepath.Join(tmp, "bundle")
	appData := filepath.Join(tmp, "appdata")

	if err := os.MkdirAll(filepath.Join(bundlePath, "ui", "dist"), 0o755); err != nil {
		t.Fatalf("failed to create ui dist dir: %v", err)
	}
	indexPath := filepath.Join(bundlePath, "ui", "dist", "index.html")
	if err := os.WriteFile(indexPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write ui index: %v", err)
	}

	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID:   "ui",
				Type: "ui-bundle",
				Assets: []manifest.Asset{
					{Path: filepath.ToSlash(filepath.Join("ui", "dist", "index.html"))},
				},
				LogDir: filepath.ToSlash(filepath.Join("logs", "ui.log")),
				Ports: &manifest.ServicePorts{
					Requested: []manifest.PortRequest{
						{Name: "ui", Range: manifest.PortRange{Min: 24100, Max: 24200}},
					},
				},
				Health: manifest.HealthCheck{
					Type:     "http",
					Path:     "/health",
					PortName: "ui",
				},
				Readiness: manifest.ReadinessCheck{
					Type:     "health_success",
					PortName: "ui",
				},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: bundlePath,
		appData:    appData,
	})

	realFS := RealFileSystem{}
	s.fs = realFS
	s.assetVerifier = assets.NewVerifier(bundlePath, realFS, s.telemetry)

	if ports, ok := s.portAllocator.(*testutil.MockPortAllocator); ok {
		ports.SetPort("ui", "ui", 0)
	}
	if hc, ok := s.healthChecker.(*testutil.MockHealthChecker); ok {
		hc.SetWaitReadinessErr(errors.New("health check failed"))
	}

	if err := s.startUIBundleService(context.Background(), m.Services[0]); err != nil {
		t.Fatalf("startUIBundleService failed: %v", err)
	}
	s.wg.Wait()
	defer s.stopServices(context.Background())

	status := s.ServiceStatuses()["ui"]
	if status.Ready {
		t.Errorf("ui service ready = true, want false")
	}
	if !strings.Contains(status.Message, "health check failed") {
		t.Errorf("ui service message = %q, want readiness error", status.Message)
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

func TestStartService_GPURequiredMissing(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()
	rec := &recordingTelemetry{}

	status := gpu.Status{Available: false, Method: "mock", Reason: "no gpu detected"}
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "gpu-api",
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/api"},
				},
				GPU: &manifest.GPURequirements{Requirement: "required"},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
		telemetry:  rec,
		gpuStatus:  status,
	})
	s.gpuApplier = gpu.NewApplier(status, rec)

	err := s.startService(context.Background(), m.Services[0])
	if err == nil {
		t.Fatal("startService() expected error when GPU is required but unavailable")
	}
	if !strings.Contains(err.Error(), "gpu required") {
		t.Fatalf("startService() error = %v, want gpu required error", err)
	}
	if !rec.Has("gpu_required_missing") {
		t.Errorf("gpu_required_missing telemetry not recorded")
	}
	pr := s.procRunner.(*testutil.MockProcessRunner)
	if len(pr.StartedCmds()) != 0 {
		t.Errorf("startService() should not launch process when GPU is missing, got %d starts", len(pr.StartedCmds()))
	}
}

func TestStartService_GPUOptionalFallback(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()
	rec := &recordingTelemetry{}

	status := gpu.Status{Available: false, Method: "mock", Reason: "no gpu detected"}
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "gpu-worker",
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/worker"},
				},
				GPU: &manifest.GPURequirements{Requirement: "optional_with_cpu_fallback"},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
		telemetry:  rec,
		gpuStatus:  status,
	})
	s.gpuApplier = gpu.NewApplier(status, rec)

	if err := s.startService(context.Background(), m.Services[0]); err != nil {
		t.Fatalf("startService() unexpected error with optional GPU fallback: %v", err)
	}

	pr := s.procRunner.(*testutil.MockProcessRunner)
	envCalls := pr.StartedEnvs()
	if len(envCalls) == 0 {
		t.Fatal("startService() should start process and record env")
	}
	envMap := envSliceToMap(envCalls[0])
	if envMap["BUNDLE_GPU_MODE"] != "cpu" {
		t.Errorf("BUNDLE_GPU_MODE = %q, want %q", envMap["BUNDLE_GPU_MODE"], "cpu")
	}
	if envMap["BUNDLE_GPU_AVAILABLE"] != "false" {
		t.Errorf("BUNDLE_GPU_AVAILABLE = %q, want %q", envMap["BUNDLE_GPU_AVAILABLE"], "false")
	}
	if !rec.Has("gpu_fallback_cpu") {
		t.Errorf("gpu_fallback_cpu telemetry not recorded")
	}
}

func TestStartService_GPUOptionalWarn(t *testing.T) {
	tmp := t.TempDir()
	binKey := platformBinaryKey()
	rec := &recordingTelemetry{}

	status := gpu.Status{Available: false, Method: "mock", Reason: "no gpu detected"}
	m := &manifest.Manifest{
		App: manifest.App{Name: "test-app", Version: "1.0.0"},
		Services: []manifest.Service{
			{
				ID: "gpu-ui",
				Binaries: map[string]manifest.Binary{
					binKey: {Path: "bin/ui"},
				},
				GPU: &manifest.GPURequirements{Requirement: "optional_but_warn"},
			},
		},
	}

	s := newTestSupervisor(t, testSupervisorConfig{
		manifest:   m,
		bundlePath: tmp,
		appData:    tmp,
		telemetry:  rec,
		gpuStatus:  status,
	})
	s.gpuApplier = gpu.NewApplier(status, rec)

	if err := s.startService(context.Background(), m.Services[0]); err != nil {
		t.Fatalf("startService() unexpected error with optional warn GPU: %v", err)
	}

	pr := s.procRunner.(*testutil.MockProcessRunner)
	envCalls := pr.StartedEnvs()
	if len(envCalls) == 0 {
		t.Fatal("startService() should start process and record env for optional warn")
	}
	envMap := envSliceToMap(envCalls[0])
	if envMap["BUNDLE_GPU_MODE"] != "cpu" {
		t.Errorf("BUNDLE_GPU_MODE = %q, want %q", envMap["BUNDLE_GPU_MODE"], "cpu")
	}
	if envMap["BUNDLE_GPU_AVAILABLE"] != "false" {
		t.Errorf("BUNDLE_GPU_AVAILABLE = %q, want %q", envMap["BUNDLE_GPU_AVAILABLE"], "false")
	}
	if !rec.Has("gpu_optional_unavailable") {
		t.Errorf("gpu_optional_unavailable telemetry not recorded")
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
