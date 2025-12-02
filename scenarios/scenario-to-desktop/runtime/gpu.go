package bundleruntime

import (
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/manifest"
)

// applyGPURequirement enforces GPU requirements for a service.
// Delegates to gpu.Applier.
func (s *Supervisor) applyGPURequirement(env map[string]string, svc manifest.Service) error {
	applier := gpu.NewApplier(s.gpuStatus, s.telemetry)
	return applier.Apply(env, svc)
}

// GPUStatus returns the current GPU detection status.
func (s *Supervisor) GPUStatus() GPUStatus {
	return s.gpuStatus
}

// boolToString converts a boolean to "true" or "false".
// Re-exported from gpu package for tests.
func boolToString(v bool) string {
	return gpu.BoolToString(v)
}
