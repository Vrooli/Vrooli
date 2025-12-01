package bundleruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-desktop-runtime/manifest"
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

	s := &Supervisor{
		opts: Options{
			BundlePath: tmp,
			Manifest:   &manifest.Manifest{},
		},
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
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

	s := &Supervisor{
		opts: Options{
			BundlePath: filepath.Join(tmp, "bundle"),
			Manifest:   &manifest.Manifest{},
		},
		portMap: map[string]map[string]int{
			"playwright-driver": {"http": 48000},
		},
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
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
