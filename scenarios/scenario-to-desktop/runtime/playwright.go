package bundleruntime

import (
	"scenario-to-desktop-runtime/assets"
	"scenario-to-desktop-runtime/manifest"
)

// applyPlaywrightConventions sets up environment variables for Playwright-based services.
// Delegates to assets.ApplyPlaywrightConventions.
func (s *Supervisor) applyPlaywrightConventions(svc manifest.Service, env map[string]string) error {
	cfg := assets.PlaywrightConfig{
		BundlePath: s.opts.BundlePath,
		FS:         s.fs,
		EnvReader:  s.envReader,
		Ports:      s.portAllocator,
		Telemetry:  s.telemetry,
	}
	return assets.ApplyPlaywrightConventions(cfg, svc, env)
}

// serviceUsesPlaywright determines if a service requires Playwright setup.
// Delegates to assets.ServiceUsesPlaywright.
func serviceUsesPlaywright(svc manifest.Service) bool {
	return assets.ServiceUsesPlaywright(svc)
}
