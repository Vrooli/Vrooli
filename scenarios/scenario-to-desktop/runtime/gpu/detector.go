// Package gpu provides GPU detection and requirement enforcement for the bundle runtime.
package gpu

import (
	"bytes"
	"context"
	goruntime "runtime"
	"strings"
	"time"

	"scenario-to-desktop-runtime/infra"
)

// Status holds the result of GPU detection.
type Status struct {
	Available bool   // Whether a usable GPU was detected
	Method    string // Detection method used (env_override, nvidia-smi, system_profiler, wmic, probe)
	Reason    string // Human-readable explanation
}

// Detector abstracts GPU detection for testing.
type Detector interface {
	// Detect probes the system for GPU availability.
	Detect() Status
}

// RealDetector implements Detector using system commands.
type RealDetector struct {
	CommandRunner infra.CommandRunner
	EnvReader     infra.EnvReader
}

// NewDetector creates a new RealDetector with the given dependencies.
func NewDetector(cmdRunner infra.CommandRunner, envReader infra.EnvReader) *RealDetector {
	return &RealDetector{
		CommandRunner: cmdRunner,
		EnvReader:     envReader,
	}
}

// Detect implements Detector interface.
// It probes the system for GPU availability.
// Detection can be overridden via the BUNDLE_GPU_AVAILABLE environment variable.
// On Linux, it checks for nvidia-smi. On macOS, it uses system_profiler.
// On Windows, it queries wmic.
func (d *RealDetector) Detect() Status {
	// Allow environment override for testing or forcing behavior.
	override := strings.TrimSpace(d.EnvReader.Getenv("BUNDLE_GPU_AVAILABLE"))
	switch strings.ToLower(override) {
	case "1", "true", "yes", "on":
		return Status{Available: true, Method: "env_override", Reason: "forced available via BUNDLE_GPU_AVAILABLE"}
	case "0", "false", "no", "off":
		return Status{Available: false, Method: "env_override", Reason: "forced unavailable via BUNDLE_GPU_AVAILABLE"}
	}

	// Check for NVIDIA GPU via nvidia-smi (works on Linux, Windows, and some macOS setups).
	if path, err := d.CommandRunner.LookPath("nvidia-smi"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if out, err := d.CommandRunner.Output(ctx, path, "--query-gpu=name", "--format=csv,noheader"); err == nil && len(bytes.TrimSpace(out)) > 0 {
			return Status{Available: true, Method: "nvidia-smi", Reason: "nvidia gpu detected"}
		}
	}

	// Platform-specific fallbacks.
	switch goruntime.GOOS {
	case "darwin":
		return d.detectGPUDarwin()
	case "windows":
		return d.detectGPUWindows()
	}

	return Status{Available: false, Method: "probe", Reason: "no GPU detected"}
}

// detectGPUDarwin checks for GPU on macOS using system_profiler.
func (d *RealDetector) detectGPUDarwin() Status {
	path, err := d.CommandRunner.LookPath("system_profiler")
	if err != nil {
		return Status{Available: false, Method: "probe", Reason: "system_profiler not found"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := d.CommandRunner.Output(ctx, path, "SPDisplaysDataType")
	if err != nil {
		return Status{Available: false, Method: "system_profiler", Reason: "system_profiler failed"}
	}
	lower := bytes.ToLower(out)
	if bytes.Contains(lower, []byte("chipset model")) || bytes.Contains(lower, []byte("gpu")) {
		return Status{Available: true, Method: "system_profiler", Reason: "GPU reported by system_profiler"}
	}
	return Status{Available: false, Method: "system_profiler", Reason: "no GPU info in system_profiler output"}
}

// detectGPUWindows checks for GPU on Windows using wmic.
func (d *RealDetector) detectGPUWindows() Status {
	path, err := d.CommandRunner.LookPath("wmic")
	if err != nil {
		return Status{Available: false, Method: "probe", Reason: "wmic not found"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := d.CommandRunner.Output(ctx, path, "path", "win32_VideoController", "get", "name")
	if err != nil {
		return Status{Available: false, Method: "wmic", Reason: "wmic query failed"}
	}
	lower := bytes.ToLower(out)
	if bytes.Contains(lower, []byte("nvidia")) || bytes.Contains(lower, []byte("amd")) || bytes.Contains(lower, []byte("intel")) {
		return Status{Available: true, Method: "wmic", Reason: "GPU reported by wmic"}
	}
	return Status{Available: false, Method: "wmic", Reason: "no recognized GPU in wmic output"}
}

// Ensure RealDetector implements Detector.
var _ Detector = (*RealDetector)(nil)
