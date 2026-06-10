package resources

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// gpuOverrideEnvVar is the user-facing override for GPU probe behavior.
// Valid values: "auto" (default), "on", "off".
const gpuOverrideEnvVar = "VROOLI_GPU"

// gpuProbeFunc returns true when the named GPU probe indicates a usable GPU
// is available for compose overlays to be applied.
type gpuProbeFunc func(ctx context.Context, probe string) bool

// gpuProbe is indirected for tests.
var gpuProbe gpuProbeFunc = runGPUProbe

var collectHostInventory = hostinventory.Collect

// gpuOverride returns the VROOLI_GPU override value, lowercased and trimmed.
// Returns "auto" when unset.
func gpuOverride() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(gpuOverrideEnvVar)))
	if v == "" {
		return "auto"
	}
	return v
}

// shouldUseGPU decides whether to apply the overlay compose file and env
// overrides declared in a resource manifest's gpu block. The rules:
//
//   - No gpu block or empty probe: never use GPU overlay.
//   - VROOLI_GPU=on:  force overlay regardless of probe.
//   - VROOLI_GPU=off: skip overlay regardless of probe.
//   - VROOLI_GPU=auto (default): run the named probe; use overlay only if it passes.
func shouldUseGPU(ctx context.Context, probe string) bool {
	probe = strings.TrimSpace(probe)
	if probe == "" {
		return false
	}
	switch gpuOverride() {
	case "on":
		return true
	case "off":
		return false
	}
	return gpuProbe(ctx, probe)
}

// runGPUProbe dispatches to probe-specific logic.
func runGPUProbe(ctx context.Context, probe string) bool {
	switch probe {
	case "nvidia":
		return nvidiaProbe(ctx)
	default:
		return false
	}
}

// nvidiaProbe reports whether an nvidia GPU is addressable by docker. It
// uses the shared host inventory authority so resource overlays do not own
// private host-probe logic.
func nvidiaProbe(ctx context.Context) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	snapshot, err := collectHostInventory(probeCtx)
	if err != nil {
		return false
	}
	return snapshot.HasDockerAddressableNvidiaGPU()
}
