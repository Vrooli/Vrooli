package gpu

import (
	"fmt"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
)

// Applier applies GPU requirements to service environments.
type Applier struct {
	Status    Status
	Telemetry telemetry.Recorder
}

// NewApplier creates a new GPU requirement applier.
func NewApplier(status Status, telem telemetry.Recorder) *Applier {
	return &Applier{
		Status:    status,
		Telemetry: telem,
	}
}

// Apply enforces GPU requirements for a service.
// Sets BUNDLE_GPU_AVAILABLE and BUNDLE_GPU_MODE environment variables.
// Returns an error if GPU is required but not available.
func (a *Applier) Apply(env map[string]string, svc manifest.Service) error {
	req := svc.GPURequirement()
	env["BUNDLE_GPU_AVAILABLE"] = BoolToString(a.Status.Available)

	if req == "" {
		return nil
	}

	switch req {
	case "required":
		if !a.Status.Available {
			_ = a.Telemetry.Record("gpu_required_missing", map[string]interface{}{
				"service_id": svc.ID,
				"reason":     a.Status.Reason,
			})
			return fmt.Errorf("gpu required for %s: %s", svc.ID, a.Status.Reason)
		}
		env["BUNDLE_GPU_MODE"] = "gpu"

	case "optional_with_cpu_fallback":
		if a.Status.Available {
			env["BUNDLE_GPU_MODE"] = "gpu"
			return nil
		}
		env["BUNDLE_GPU_MODE"] = "cpu"
		_ = a.Telemetry.Record("gpu_fallback_cpu", map[string]interface{}{
			"service_id": svc.ID,
			"reason":     a.Status.Reason,
		})

	case "optional_but_warn":
		if a.Status.Available {
			env["BUNDLE_GPU_MODE"] = "gpu"
			return nil
		}
		env["BUNDLE_GPU_MODE"] = "cpu"
		_ = a.Telemetry.Record("gpu_optional_unavailable", map[string]interface{}{
			"service_id": svc.ID,
			"reason":     a.Status.Reason,
		})

	default:
		return fmt.Errorf("unknown gpu requirement %q for service %s", req, svc.ID)
	}
	return nil
}

// BoolToString converts a boolean to "true" or "false".
func BoolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
