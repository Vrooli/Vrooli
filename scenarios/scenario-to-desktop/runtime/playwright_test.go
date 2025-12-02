package bundleruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// playwrightTestPortAllocator is a test double for PortAllocator in playwright tests.
type playwrightTestPortAllocator struct {
	ports map[string]map[string]int
}

func (m *playwrightTestPortAllocator) Allocate() error { return nil }
func (m *playwrightTestPortAllocator) Resolve(serviceID, portName string) (int, error) {
	if ports, ok := m.ports[serviceID]; ok {
		if port, ok := ports[portName]; ok {
			return port, nil
		}
	}
	return 0, fmt.Errorf("port %s not found for %s", portName, serviceID)
}
func (m *playwrightTestPortAllocator) Map() map[string]map[string]int { return m.ports }

// testPlaywrightSupervisor creates a Supervisor configured for playwright testing.
func testPlaywrightSupervisor(t *testing.T, opts Options, ports map[string]map[string]int) *Supervisor {
	t.Helper()

	appData := t.TempDir()
	if opts.AppDataDir != "" {
		appData = opts.AppDataDir
	}

	if opts.Manifest == nil {
		opts.Manifest = &manifest.Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           manifest.App{Name: "test", Version: "1.0.0"},
			IPC:           manifest.IPC{Host: "127.0.0.1", Port: 47710, AuthTokenRel: "runtime/auth-token"},
			Telemetry:     manifest.Telemetry{File: "telemetry.jsonl"},
			Services:      []manifest.Service{{ID: "api", Type: "api", Binaries: map[string]manifest.Binary{"linux-x64": {Path: "bin"}}, Health: manifest.HealthCheck{Type: "http"}, Readiness: manifest.ReadinessCheck{Type: "port_open"}}},
		}
	}

	opts.AppDataDir = appData
	if opts.BundlePath == "" {
		opts.BundlePath = t.TempDir()
	}
	opts.DryRun = true
	opts.Clock = &fakeClock{now: time.Now()}
	opts.FileSystem = RealFileSystem{}
	opts.PortAllocator = &playwrightTestPortAllocator{ports: ports}
	opts.EnvReader = RealEnvReader{}

	s, err := NewSupervisor(opts)
	if err != nil {
		t.Fatalf("NewSupervisor() error = %v", err)
	}

	s.telemetryPath = filepath.Join(appData, "telemetry.jsonl")
	return s
}

func TestServiceUsesPlaywright(t *testing.T) {
	t.Run("detects playwright in service ID", func(t *testing.T) {
		svc := manifest.Service{ID: "playwright-driver"}
		if !serviceUsesPlaywright(svc) {
			t.Error("serviceUsesPlaywright() should detect 'playwright' in ID")
		}
	})

	t.Run("detects playwright case-insensitive", func(t *testing.T) {
		svc := manifest.Service{ID: "PLAYWRIGHT-AUTOMATION"}
		if !serviceUsesPlaywright(svc) {
			t.Error("serviceUsesPlaywright() should detect 'PLAYWRIGHT' case-insensitively")
		}
	})

	t.Run("detects PLAYWRIGHT_ env vars", func(t *testing.T) {
		svc := manifest.Service{
			ID:  "browser-service",
			Env: map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "/browsers"},
		}
		if !serviceUsesPlaywright(svc) {
			t.Error("serviceUsesPlaywright() should detect PLAYWRIGHT_ env vars")
		}
	})

	t.Run("returns false for non-playwright service", func(t *testing.T) {
		svc := manifest.Service{
			ID:  "api-server",
			Env: map[string]string{"PORT": "8080"},
		}
		if serviceUsesPlaywright(svc) {
			t.Error("serviceUsesPlaywright() should return false for non-playwright service")
		}
	})

	t.Run("returns false for empty service", func(t *testing.T) {
		svc := manifest.Service{ID: "empty"}
		if serviceUsesPlaywright(svc) {
			t.Error("serviceUsesPlaywright() should return false for empty service")
		}
	})
}

func TestApplyPlaywrightConventions(t *testing.T) {
	t.Run("skips non-playwright services", func(t *testing.T) {
		s := testPlaywrightSupervisor(t, Options{}, map[string]map[string]int{"api": {"http": 8080}})

		svc := manifest.Service{ID: "api", Env: map[string]string{}}
		env := map[string]string{}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if _, ok := env["PLAYWRIGHT_DRIVER_PORT"]; ok {
			t.Error("applyPlaywrightConventions() should not set PLAYWRIGHT vars for non-playwright service")
		}
	})

	t.Run("sets driver port from allocated ports", func(t *testing.T) {
		s := testPlaywrightSupervisor(t, Options{}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_DRIVER_PORT"] != "9222" {
			t.Errorf("PLAYWRIGHT_DRIVER_PORT = %q, want '9222'", env["PLAYWRIGHT_DRIVER_PORT"])
		}
	})

	t.Run("sets driver URL from allocated ports", func(t *testing.T) {
		s := testPlaywrightSupervisor(t, Options{}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_DRIVER_URL"] != "http://127.0.0.1:9222" {
			t.Errorf("PLAYWRIGHT_DRIVER_URL = %q, want 'http://127.0.0.1:9222'", env["PLAYWRIGHT_DRIVER_URL"])
		}
	})

	t.Run("sets default ENGINE to playwright", func(t *testing.T) {
		s := testPlaywrightSupervisor(t, Options{}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if env["ENGINE"] != "playwright" {
			t.Errorf("ENGINE = %q, want 'playwright'", env["ENGINE"])
		}
	})

	t.Run("preserves existing ENGINE value", func(t *testing.T) {
		s := testPlaywrightSupervisor(t, Options{}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"ENGINE": "custom-engine"}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if env["ENGINE"] != "custom-engine" {
			t.Errorf("ENGINE = %q, want 'custom-engine' (should preserve existing)", env["ENGINE"])
		}
	})

	t.Run("preserves existing PLAYWRIGHT_DRIVER_PORT", func(t *testing.T) {
		s := testPlaywrightSupervisor(t, Options{}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_DRIVER_PORT": "5000"}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_DRIVER_PORT"] != "5000" {
			t.Errorf("PLAYWRIGHT_DRIVER_PORT = %q, want '5000' (should preserve existing)", env["PLAYWRIGHT_DRIVER_PORT"])
		}
	})

	t.Run("resolves chromium path relative to bundle", func(t *testing.T) {
		bundleDir := t.TempDir()

		// Create the chromium executable
		chromiumPath := filepath.Join(bundleDir, "browsers", "chromium")
		if err := os.MkdirAll(filepath.Dir(chromiumPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(chromiumPath, []byte("fake chromium"), 0755); err != nil {
			t.Fatal(err)
		}

		s := testPlaywrightSupervisor(t, Options{BundlePath: bundleDir}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "browsers/chromium"}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		expectedPath := filepath.Join(bundleDir, "browsers", "chromium")
		if env["PLAYWRIGHT_CHROMIUM_PATH"] != expectedPath {
			t.Errorf("PLAYWRIGHT_CHROMIUM_PATH = %q, want %q", env["PLAYWRIGHT_CHROMIUM_PATH"], expectedPath)
		}
	})

	t.Run("uses electron chromium fallback when path missing", func(t *testing.T) {
		bundleDir := t.TempDir()

		// Set up electron chromium path
		electronPath := "/path/to/electron/chromium"
		os.Setenv("ELECTRON_CHROMIUM_PATH", electronPath)
		defer os.Unsetenv("ELECTRON_CHROMIUM_PATH")

		s := testPlaywrightSupervisor(t, Options{BundlePath: bundleDir}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "nonexistent/chromium"}

		err := s.applyPlaywrightConventions(svc, env)
		if err != nil {
			t.Errorf("applyPlaywrightConventions() error = %v", err)
		}

		if env["PLAYWRIGHT_CHROMIUM_PATH"] != electronPath {
			t.Errorf("PLAYWRIGHT_CHROMIUM_PATH = %q, want %q (electron fallback)", env["PLAYWRIGHT_CHROMIUM_PATH"], electronPath)
		}
	})

	t.Run("returns error when chromium missing and no fallback", func(t *testing.T) {
		bundleDir := t.TempDir()

		// Ensure no electron fallback
		os.Unsetenv("ELECTRON_CHROMIUM_PATH")

		s := testPlaywrightSupervisor(t, Options{BundlePath: bundleDir}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

		svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
		env := map[string]string{"PLAYWRIGHT_CHROMIUM_PATH": "nonexistent/chromium"}

		err := s.applyPlaywrightConventions(svc, env)
		if err == nil {
			t.Error("applyPlaywrightConventions() should return error when chromium missing and no fallback")
		}
	})
}

func TestApplyPlaywrightConventionsElectronFallbackEmpty(t *testing.T) {
	bundleDir := t.TempDir()

	// Test with empty PLAYWRIGHT_CHROMIUM_PATH but ELECTRON_CHROMIUM_PATH available
	electronPath := "/electron/browsers/chromium"
	os.Setenv("ELECTRON_CHROMIUM_PATH", electronPath)
	defer os.Unsetenv("ELECTRON_CHROMIUM_PATH")

	s := testPlaywrightSupervisor(t, Options{BundlePath: bundleDir}, map[string]map[string]int{"playwright-driver": {"http": 9222}})

	svc := manifest.Service{ID: "playwright-driver", Env: map[string]string{}}
	env := map[string]string{} // No PLAYWRIGHT_CHROMIUM_PATH set

	err := s.applyPlaywrightConventions(svc, env)
	if err != nil {
		t.Errorf("applyPlaywrightConventions() error = %v", err)
	}

	if env["PLAYWRIGHT_CHROMIUM_PATH"] != electronPath {
		t.Errorf("PLAYWRIGHT_CHROMIUM_PATH = %q, want %q (electron fallback when empty)", env["PLAYWRIGHT_CHROMIUM_PATH"], electronPath)
	}
}

func TestOsStatForPlaywright(t *testing.T) {
	appData := t.TempDir()
	s := &Supervisor{
		fs: RealFileSystem{},
	}

	t.Run("returns nil for existing file", func(t *testing.T) {
		path := filepath.Join(appData, "exists.txt")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		err := s.osStatForPlaywright(path)
		if err != nil {
			t.Errorf("osStatForPlaywright() error = %v for existing file", err)
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		err := s.osStatForPlaywright("/nonexistent/path")
		if err == nil {
			t.Error("osStatForPlaywright() should return error for missing file")
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
			got := serviceUsesPlaywright(tc.svc)
			if got != tc.expect {
				t.Errorf("serviceUsesPlaywright() = %v, want %v", got, tc.expect)
			}
		})
	}
}
