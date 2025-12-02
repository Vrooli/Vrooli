package bundleruntime

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"scenario-to-desktop-runtime/manifest"
)

// applyPlaywrightConventions sets up environment variables for Playwright-based services.
// This includes setting PLAYWRIGHT_DRIVER_PORT, PLAYWRIGHT_DRIVER_URL, ENGINE,
// and handling PLAYWRIGHT_CHROMIUM_PATH with fallback to ELECTRON_CHROMIUM_PATH.
func (s *Supervisor) applyPlaywrightConventions(svc manifest.Service, env map[string]string) error {
	if !serviceUsesPlaywright(svc) {
		return nil
	}

	// Set driver port if not already specified.
	if _, ok := env["PLAYWRIGHT_DRIVER_PORT"]; !ok {
		if port, err := s.resolvePort(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_PORT"] = strconv.Itoa(port)
		}
	}

	// Set driver URL if not already specified.
	if _, ok := env["PLAYWRIGHT_DRIVER_URL"]; !ok {
		if port, err := s.resolvePort(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_URL"] = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	}

	// Default engine to playwright.
	if _, ok := env["ENGINE"]; !ok {
		env["ENGINE"] = "playwright"
	}

	// Handle Chromium path with fallback.
	chromePath := strings.TrimSpace(env["PLAYWRIGHT_CHROMIUM_PATH"])
	if chromePath == "" {
		// No path specified; try Electron's Chromium as fallback.
		if fallback := strings.TrimSpace(os.Getenv("ELECTRON_CHROMIUM_PATH")); fallback != "" {
			env["PLAYWRIGHT_CHROMIUM_PATH"] = fallback
			_ = s.recordTelemetry("playwright_chromium_fallback", map[string]interface{}{
				"service_id": svc.ID,
				"fallback":   fallback,
			})
		}
		return nil
	}

	// Resolve the specified path relative to bundle root.
	resolved := manifest.ResolvePath(s.opts.BundlePath, chromePath)
	env["PLAYWRIGHT_CHROMIUM_PATH"] = resolved

	// Verify the path exists.
	if _, err := os.Stat(resolved); err == nil {
		return nil
	}

	// Path doesn't exist; try Electron's Chromium as fallback.
	if fallback := strings.TrimSpace(os.Getenv("ELECTRON_CHROMIUM_PATH")); fallback != "" {
		env["PLAYWRIGHT_CHROMIUM_PATH"] = fallback
		_ = s.recordTelemetry("playwright_chromium_fallback", map[string]interface{}{
			"service_id": svc.ID,
			"missing":    resolved,
			"fallback":   fallback,
		})
		return nil
	}

	// No fallback available; report missing asset.
	_ = s.recordTelemetry("asset_missing", map[string]interface{}{
		"service_id": svc.ID,
		"path":       resolved,
		"reason":     "playwright_chromium",
	})
	return fmt.Errorf("playwright chromium missing for %s: %s", svc.ID, resolved)
}

// serviceUsesPlaywright determines if a service requires Playwright setup.
// It checks for "playwright" in the service ID or PLAYWRIGHT_* environment variables.
func serviceUsesPlaywright(svc manifest.Service) bool {
	// Check service ID.
	if strings.Contains(strings.ToLower(svc.ID), "playwright") {
		return true
	}
	// Check for Playwright environment variables.
	for k := range svc.Env {
		if strings.HasPrefix(k, "PLAYWRIGHT_") {
			return true
		}
	}
	return false
}
