package bundleruntime

import (
	"scenario-to-desktop-runtime/assets"
	"scenario-to-desktop-runtime/manifest"
)

// ensureAssets verifies all required assets for a service exist and are valid.
// Delegates to assets.Verifier.
func (s *Supervisor) ensureAssets(svc manifest.Service) error {
	v := assets.NewVerifier(s.opts.BundlePath, s.fs, s.telemetry)
	return v.EnsureAssets(svc)
}
