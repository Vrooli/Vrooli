package bundleruntime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// gpuStatus holds the result of GPU detection.
type gpuStatus struct {
	Available bool   // Whether a usable GPU was detected
	Method    string // Detection method used (env_override, nvidia-smi, system_profiler, wmic, probe)
	Reason    string // Human-readable explanation
}

// detectGPU probes the system for GPU availability.
// Detection can be overridden via the BUNDLE_GPU_AVAILABLE environment variable.
// On Linux, it checks for nvidia-smi. On macOS, it uses system_profiler.
// On Windows, it queries wmic.
func detectGPU() gpuStatus {
	// Allow environment override for testing or forcing behavior.
	override := strings.TrimSpace(os.Getenv("BUNDLE_GPU_AVAILABLE"))
	switch strings.ToLower(override) {
	case "1", "true", "yes", "on":
		return gpuStatus{Available: true, Method: "env_override", Reason: "forced available via BUNDLE_GPU_AVAILABLE"}
	case "0", "false", "no", "off":
		return gpuStatus{Available: false, Method: "env_override", Reason: "forced unavailable via BUNDLE_GPU_AVAILABLE"}
	}

	// Check for NVIDIA GPU via nvidia-smi (works on Linux, Windows, and some macOS setups).
	if path, err := exec.LookPath("nvidia-smi"); err == nil {
		if out, err := runCommandWithTimeout(2*time.Second, path, "--query-gpu=name", "--format=csv,noheader"); err == nil && len(bytes.TrimSpace(out)) > 0 {
			return gpuStatus{Available: true, Method: "nvidia-smi", Reason: "nvidia gpu detected"}
		}
	}

	// Platform-specific fallbacks.
	switch runtime.GOOS {
	case "darwin":
		return detectGPUDarwin()
	case "windows":
		return detectGPUWindows()
	}

	return gpuStatus{Available: false, Method: "probe", Reason: "no GPU detected"}
}

// detectGPUDarwin checks for GPU on macOS using system_profiler.
func detectGPUDarwin() gpuStatus {
	path, err := exec.LookPath("system_profiler")
	if err != nil {
		return gpuStatus{Available: false, Method: "probe", Reason: "system_profiler not found"}
	}
	out, err := runCommandWithTimeout(3*time.Second, path, "SPDisplaysDataType")
	if err != nil {
		return gpuStatus{Available: false, Method: "system_profiler", Reason: "system_profiler failed"}
	}
	lower := bytes.ToLower(out)
	if bytes.Contains(lower, []byte("chipset model")) || bytes.Contains(lower, []byte("gpu")) {
		return gpuStatus{Available: true, Method: "system_profiler", Reason: "GPU reported by system_profiler"}
	}
	return gpuStatus{Available: false, Method: "system_profiler", Reason: "no GPU info in system_profiler output"}
}

// detectGPUWindows checks for GPU on Windows using wmic.
func detectGPUWindows() gpuStatus {
	path, err := exec.LookPath("wmic")
	if err != nil {
		return gpuStatus{Available: false, Method: "probe", Reason: "wmic not found"}
	}
	out, err := runCommandWithTimeout(3*time.Second, path, "path", "win32_VideoController", "get", "name")
	if err != nil {
		return gpuStatus{Available: false, Method: "wmic", Reason: "wmic query failed"}
	}
	lower := bytes.ToLower(out)
	if bytes.Contains(lower, []byte("nvidia")) || bytes.Contains(lower, []byte("amd")) || bytes.Contains(lower, []byte("intel")) {
		return gpuStatus{Available: true, Method: "wmic", Reason: "GPU reported by wmic"}
	}
	return gpuStatus{Available: false, Method: "wmic", Reason: "no recognized GPU in wmic output"}
}

// runCommandWithTimeout executes a command with a timeout.
func runCommandWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, err
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
func (s *Supervisor) GPUStatus() gpuStatus {
	return s.gpuStatus
}
