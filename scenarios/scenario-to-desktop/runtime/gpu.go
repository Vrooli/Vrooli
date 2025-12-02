package bundleruntime

import (
	"bytes"
	"context"
	"fmt"
	goruntime "runtime"
	"strings"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// Detect implements GPUDetector interface for RealGPUDetector.
// It probes the system for GPU availability.
// Detection can be overridden via the BUNDLE_GPU_AVAILABLE environment variable.
// On Linux, it checks for nvidia-smi. On macOS, it uses system_profiler.
// On Windows, it queries wmic.
func (d *RealGPUDetector) Detect() GPUStatus {
	// Allow environment override for testing or forcing behavior.
	override := strings.TrimSpace(d.EnvReader.Getenv("BUNDLE_GPU_AVAILABLE"))
	switch strings.ToLower(override) {
	case "1", "true", "yes", "on":
		return GPUStatus{Available: true, Method: "env_override", Reason: "forced available via BUNDLE_GPU_AVAILABLE"}
	case "0", "false", "no", "off":
		return GPUStatus{Available: false, Method: "env_override", Reason: "forced unavailable via BUNDLE_GPU_AVAILABLE"}
	}

	// Check for NVIDIA GPU via nvidia-smi (works on Linux, Windows, and some macOS setups).
	if path, err := d.CommandRunner.LookPath("nvidia-smi"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if out, err := d.CommandRunner.Output(ctx, path, "--query-gpu=name", "--format=csv,noheader"); err == nil && len(bytes.TrimSpace(out)) > 0 {
			return GPUStatus{Available: true, Method: "nvidia-smi", Reason: "nvidia gpu detected"}
		}
	}

	// Platform-specific fallbacks.
	switch goruntime.GOOS {
	case "darwin":
		return d.detectGPUDarwin()
	case "windows":
		return d.detectGPUWindows()
	}

	return GPUStatus{Available: false, Method: "probe", Reason: "no GPU detected"}
}

// detectGPUDarwin checks for GPU on macOS using system_profiler.
func (d *RealGPUDetector) detectGPUDarwin() GPUStatus {
	path, err := d.CommandRunner.LookPath("system_profiler")
	if err != nil {
		return GPUStatus{Available: false, Method: "probe", Reason: "system_profiler not found"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := d.CommandRunner.Output(ctx, path, "SPDisplaysDataType")
	if err != nil {
		return GPUStatus{Available: false, Method: "system_profiler", Reason: "system_profiler failed"}
	}
	lower := bytes.ToLower(out)
	if bytes.Contains(lower, []byte("chipset model")) || bytes.Contains(lower, []byte("gpu")) {
		return GPUStatus{Available: true, Method: "system_profiler", Reason: "GPU reported by system_profiler"}
	}
	return GPUStatus{Available: false, Method: "system_profiler", Reason: "no GPU info in system_profiler output"}
}

// detectGPUWindows checks for GPU on Windows using wmic.
func (d *RealGPUDetector) detectGPUWindows() GPUStatus {
	path, err := d.CommandRunner.LookPath("wmic")
	if err != nil {
		return GPUStatus{Available: false, Method: "probe", Reason: "wmic not found"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := d.CommandRunner.Output(ctx, path, "path", "win32_VideoController", "get", "name")
	if err != nil {
		return GPUStatus{Available: false, Method: "wmic", Reason: "wmic query failed"}
	}
	lower := bytes.ToLower(out)
	if bytes.Contains(lower, []byte("nvidia")) || bytes.Contains(lower, []byte("amd")) || bytes.Contains(lower, []byte("intel")) {
		return GPUStatus{Available: true, Method: "wmic", Reason: "GPU reported by wmic"}
	}
	return GPUStatus{Available: false, Method: "wmic", Reason: "no recognized GPU in wmic output"}
}

// applyGPURequirement enforces GPU requirements for a service.
// Sets BUNDLE_GPU_AVAILABLE and BUNDLE_GPU_MODE environment variables.
// Returns an error if GPU is required but not available.
func (s *Supervisor) applyGPURequirement(env map[string]string, svc manifest.Service) error {
	req := svc.GPURequirement()
	env["BUNDLE_GPU_AVAILABLE"] = boolToString(s.gpuStatus.Available)

	if req == "" {
		return nil
	}

	switch req {
	case "required":
		if !s.gpuStatus.Available {
			_ = s.recordTelemetry("gpu_required_missing", map[string]interface{}{
				"service_id": svc.ID,
				"reason":     s.gpuStatus.Reason,
			})
			return fmt.Errorf("gpu required for %s: %s", svc.ID, s.gpuStatus.Reason)
		}
		env["BUNDLE_GPU_MODE"] = "gpu"

	case "optional_with_cpu_fallback":
		if s.gpuStatus.Available {
			env["BUNDLE_GPU_MODE"] = "gpu"
			return nil
		}
		env["BUNDLE_GPU_MODE"] = "cpu"
		_ = s.recordTelemetry("gpu_fallback_cpu", map[string]interface{}{
			"service_id": svc.ID,
			"reason":     s.gpuStatus.Reason,
		})

	case "optional_but_warn":
		if s.gpuStatus.Available {
			env["BUNDLE_GPU_MODE"] = "gpu"
			return nil
		}
		env["BUNDLE_GPU_MODE"] = "cpu"
		_ = s.recordTelemetry("gpu_optional_unavailable", map[string]interface{}{
			"service_id": svc.ID,
			"reason":     s.gpuStatus.Reason,
		})

	default:
		return fmt.Errorf("unknown gpu requirement %q for service %s", req, svc.ID)
	}
	return nil
}

// GPUStatus returns the current GPU detection status.
func (s *Supervisor) GPUStatus() GPUStatus {
	return s.gpuStatus
}

// boolToString converts a boolean to "true" or "false".
func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
