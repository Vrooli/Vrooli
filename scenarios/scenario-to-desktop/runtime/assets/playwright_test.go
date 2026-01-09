package assets

import (
	"os"
	"path/filepath"
	"testing"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/testutil"
)

func TestServiceUsesPlaywright(t *testing.T) {
	t.Run("detects playwright in service ID", func(t *testing.T) {
		svc := manifest.Service{ID: "playwright-driver"}
		if !ServiceUsesPlaywright(svc) {
			t.Error("ServiceUsesPlaywright() should detect 'playwright' in ID")
		}
	})

	t.Run("detects playwright case-insensitive", func(t *testing.T) {
		svc := manifest.Service{ID: "PLAYWRIGHT-AUTOMATION"}
		if !ServiceUsesPlaywright(svc) {
			t.Error("ServiceUsesPlaywright() should detect 'PLAYWRIGHT' case-insensitively")
		}
	})

	t.Run("detects PLAYWRIGHT_ env vars", func(t *testing.T) {
		svc := manifest.Service{
			ID:  "browser-service",
			Env: map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "/browsers"},
		}
		if !ServiceUsesPlaywright(svc) {
			t.Error("ServiceUsesPlaywright() should detect PLAYWRIGHT_ env vars")
		}
	})

	t.Run("returns false for non-playwright service", func(t *testing.T) {
		svc := manifest.Service{
			ID:  "api-server",
			Env: map[string]string{"PORT": "8080"},
		}
		if ServiceUsesPlaywright(svc) {
			t.Error("ServiceUsesPlaywright() should return false for non-playwright service")
		}
	})

	t.Run("returns false for empty service", func(t *testing.T) {
		svc := manifest.Service{ID: "empty"}
		if ServiceUsesPlaywright(svc) {
			t.Error("ServiceUsesPlaywright() should return false for empty service")
		}
	})
}

func TestPlaywrightDetectionWithEnvVars(t *testing.T) {
	tests := []struct {
		name   string
		svc    manifest.Service
		expect bool
	}{
		{
			name:   "PLAYWRIGHT_BROWSERS_PATH",
			svc:    manifest.Service{ID: "svc", Env: map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "/browsers"}},
			expect: true,
		},
		{
			name:   "PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD",
			svc:    manifest.Service{ID: "svc", Env: map[string]string{"PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD": "1"}},
			expect: true,
		},
		{
			name:   "PLAYWRIGHT_CHROMIUM_PATH",
			svc:    manifest.Service{ID: "svc", Env: map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "/chrome"}},
			expect: true,
		},
		{
			name:   "Multiple env vars including playwright",
			svc:    manifest.Service{ID: "svc", Env: map[string]string{"PORT": "8080", "PLAYWRIGHT_DEBUG": "1", "NODE_ENV": "prod"}},
			expect: true,
		},
		{
			name:   "Non-playwright env vars",
			svc:    manifest.Service{ID: "svc", Env: map[string]string{"PORT": "8080", "DATABASE_URL": "postgres://"}},
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ServiceUsesPlaywright(tc.svc)
			if got != tc.expect {
				t.Errorf("ServiceUsesPlaywright() = %v, want %v", got, tc.expect)
			}
		})
	}
}

func TestApplyPlaywrightConventions(t *testing.T) {
	t.Run("skips non-playwright services", func(t *testing.T) {
		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: t.TempDir(),
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"api": {"http": 8080}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "api", Env: map[string]string{}}
		env := map[string]string{}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if _, ok := env["PLAYWRIGHT_DRIVER_PORT"]; ok {
			t.Error("ApplyPlaywrightConventions() should not set PLAYWRIGHT vars for non-playwright service")
		}
	})

	t.Run("sets driver port from allocated ports", func(t *testing.T) {
		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: t.TempDir(),
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_DRIVER_PORT"] != "9222" {
			t.Errorf("PLAYWRIGHT_DRIVER_PORT = %q, want '9222'", env["PLAYWRIGHT_DRIVER_PORT"])
		}
	})

	t.Run("sets driver URL from allocated ports", func(t *testing.T) {
		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: t.TempDir(),
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_DRIVER_URL"] != "http://127.0.0.1:9222" {
			t.Errorf("PLAYWRIGHT_DRIVER_URL = %q, want 'http://127.0.0.1:9222'", env["PLAYWRIGHT_DRIVER_URL"])
		}
	})

	t.Run("sets default ENGINE to playwright", func(t *testing.T) {
		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: t.TempDir(),
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if env["ENGINE"] != "playwright" {
			t.Errorf("ENGINE = %q, want 'playwright'", env["ENGINE"])
		}
	})

	t.Run("preserves existing ENGINE value", func(t *testing.T) {
		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: t.TempDir(),
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"ENGINE": "custom-engine"}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if env["ENGINE"] != "custom-engine" {
			t.Errorf("ENGINE = %q, want 'custom-engine' (should preserve existing)", env["ENGINE"])
		}
	})

	t.Run("preserves existing PLAYWRIGHT_DRIVER_PORT", func(t *testing.T) {
		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: t.TempDir(),
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_DRIVER_PORT": "5000"}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_DRIVER_PORT"] != "5000" {
			t.Errorf("PLAYWRIGHT_DRIVER_PORT = %q, want '5000' (should preserve existing)", env["PLAYWRIGHT_DRIVER_PORT"])
		}
	})

	t.Run("resolves chromium path relative to bundle", func(t *testing.T) {
		bundleDir := t.TempDir()

		chromiumPath := filepath.Join(bundleDir, "browsers", "chromium")
		if err := os.MkdirAll(filepath.Dir(chromiumPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(chromiumPath, []byte("fake chromium"), 0755); err != nil {
			t.Fatal(err)
		}

		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: bundleDir,
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "browsers/chromium"}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		expectedPath := filepath.Join(bundleDir, "browsers", "chromium")
		if env["PLAYWRIGHT_CHROMIUM_PATH"] != expectedPath {
			t.Errorf("PLAYWRIGHT_CHROMIUM_PATH = %q, want %q", env["PLAYWRIGHT_CHROMIUM_PATH"], expectedPath)
		}
	})

	t.Run("uses electron chromium fallback when path missing", func(t *testing.T) {
		bundleDir := t.TempDir()

		electronPath := "/path/to/electron/chromium"
		t.Setenv("ELECTRON_CHROMIUM_PATH", electronPath)

		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: bundleDir,
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "nonexistent/chromium"}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err != nil {
			t.Errorf("ApplyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_CHROMIUM_PATH"] != electronPath {
			t.Errorf("PLAYWRIGHT_CHROMIUM_PATH = %q, want %q (electron fallback)", env["PLAYWRIGHT_CHROMIUM_PATH"], electronPath)
		}
	})

	t.Run("returns error when chromium missing and no fallback", func(t *testing.T) {
		bundleDir := t.TempDir()

		t.Setenv("ELECTRON_CHROMIUM_PATH", "")

		telem := &mockTelemetryRecorder{}
		cfg := PlaywrightConfig{
			BundlePath: bundleDir,
			FS:         infra.RealFileSystem{},
			EnvReader:  infra.RealEnvReader{},
			Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
			Telemetry:  telem,
		}

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "nonexistent/chromium"}

		err := ApplyPlaywrightConventions(cfg, svc, env)
		if err == nil {
			t.Error("ApplyPlaywrightConventions() should return error when chromium missing and no fallback")
		}
	})
}

func TestApplyPlaywrightConventionsElectronFallbackEmpty(t *testing.T) {
	bundleDir := t.TempDir()

	electronPath := "/electron/browsers/chromium"
	t.Setenv("ELECTRON_CHROMIUM_PATH", electronPath)

	telem := &mockTelemetryRecorder{}
	cfg := PlaywrightConfig{
		BundlePath: bundleDir,
		FS:         infra.RealFileSystem{},
		EnvReader:  infra.RealEnvReader{},
		Ports:      &testutil.MockPortAllocator{Ports: map[string]map[string]int{"playwright-driver": {"http": 9222}}},
		Telemetry:  telem,
	}

	svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
	env := map[string]string{}

	err := ApplyPlaywrightConventions(cfg, svc, env)
	if err != nil {
		t.Errorf("ApplyPlaywrightConventions() error = %v", err)
	}

	if env["PLAYWRIGHT_CHROMIUM_PATH"] != electronPath {
		t.Errorf("PLAYWRIGHT_CHROMIUM_PATH = %q, want %q (electron fallback when empty)", env["PLAYWRIGHT_CHROMIUM_PATH"], electronPath)
	}
}
