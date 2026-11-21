package assets

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/ports"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/telemetry"
)

// PlaywrightConfig holds configuration for Playwright conventions.
type PlaywrightConfig struct {
	BundlePath string
	FS         infra.FileSystem
	EnvReader  infra.EnvReader
	Ports      ports.Allocator
	Telemetry  telemetry.Recorder
}

// ApplyPlaywrightConventions sets up environment variables for Playwright-based services.
// This includes setting PLAYWRIGHT_DRIVER_PORT, PLAYWRIGHT_DRIVER_URL, ENGINE,
// and handling BROWSER_EXECUTABLE_PATH with fallback to ELECTRON_CHROMIUM_PATH.
func ApplyPlaywrightConventions(cfg PlaywrightConfig, svc manifest.Service, env map[string]string) error {
	if !ServiceUsesPlaywright(svc) {
		return nil
	}

	// Set driver port if not already specified.
	if _, ok := env["PLAYWRIGHT_DRIVER_PORT"]; !ok {
		if port, err := cfg.Ports.Resolve(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_PORT"] = strconv.Itoa(port)
		}
	}

	// Set driver URL if not already specified.
	if _, ok := env["PLAYWRIGHT_DRIVER_URL"]; !ok {
		if port, err := cfg.Ports.Resolve(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_URL"] = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	}

	// Default engine to playwright.
	if _, ok := env["ENGINE"]; !ok {
		env["ENGINE"] = "playwright"
	}

	// Resolve a declared package-local browser closure against the bundle root.
	// Packaged runtimes must never inherit a developer's ~/.cache/ms-playwright
	// path by accident. The driver uses this directory to select its pinned
	// Chromium revision from the package's browsers.json inventory.
	if browserDir := strings.TrimSpace(env["PLAYWRIGHT_BROWSERS_PATH"]); browserDir != "" {
		resolved := manifest.ResolvePath(cfg.BundlePath, browserDir)
		info, err := cfg.FS.Stat(resolved)
		if err != nil {
			return fmt.Errorf("Playwright browser closure directory missing for %s: %s", svc.ID, resolved)
		}
		if !info.IsDir() {
			return fmt.Errorf("Playwright browser closure path is not a directory for %s: %s", svc.ID, resolved)
		}
		env["PLAYWRIGHT_BROWSERS_PATH"] = resolved
	}

	// Handle Chromium path with fallback.
	chromePath := strings.TrimSpace(env["BROWSER_EXECUTABLE_PATH"])
	if chromePath == "" {
		// No path specified; try Electron's Chromium as fallback.
		if fallback := strings.TrimSpace(cfg.EnvReader.Getenv("ELECTRON_CHROMIUM_PATH")); fallback != "" {
			env["BROWSER_EXECUTABLE_PATH"] = fallback
			_ = cfg.Telemetry.Record("playwright_chromium_fallback", map[string]interface{}{
				"service_id": svc.ID,
				"fallback":   fallback,
			})
		}
		return nil
	}

	// Resolve the specified path relative to bundle root.
	resolved := manifest.ResolvePath(cfg.BundlePath, chromePath)
	env["BROWSER_EXECUTABLE_PATH"] = resolved

	// Verify the path exists.
	if _, err := cfg.FS.Stat(resolved); err == nil {
		return nil
	}

	// Path doesn't exist; try Electron's Chromium as fallback.
	if fallback := strings.TrimSpace(cfg.EnvReader.Getenv("ELECTRON_CHROMIUM_PATH")); fallback != "" {
		env["BROWSER_EXECUTABLE_PATH"] = fallback
		_ = cfg.Telemetry.Record("playwright_chromium_fallback", map[string]interface{}{
			"service_id": svc.ID,
			"missing":    resolved,
			"fallback":   fallback,
		})
		return nil
	}

	// No fallback available; report missing asset.
	_ = cfg.Telemetry.Record("asset_missing", map[string]interface{}{
		"service_id": svc.ID,
		"path":       resolved,
		"reason":     "playwright_chromium",
	})
	return fmt.Errorf("playwright chromium missing for %s: %s", svc.ID, resolved)
}

// ServiceUsesPlaywright determines if a service requires Playwright setup.
// It checks for "playwright" in the service ID or a Playwright-specific
// environment key, including the browser executable key consumed by BAS.
func ServiceUsesPlaywright(svc manifest.Service) bool {
	// Check service ID.
	if strings.Contains(strings.ToLower(svc.ID), "playwright") {
		return true
	}
	// Check for Playwright environment variables.
	for k := range svc.Env {
		if k == "BROWSER_EXECUTABLE_PATH" || strings.HasPrefix(k, "PLAYWRIGHT_") {
			return true
		}
	}
	return false
}
