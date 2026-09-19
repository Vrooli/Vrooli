package runtime

import (
	"context"
	"sync"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	capabilityParameterA = 30
)

// capabilityFactsFn is the seam through which the generic tool handler learns the
// host's hardware facts for evaluating a tool's `requires` capability gate.
// Tests override it to simulate a CPU-only host, a specific arch, etc. It is
// collected at most once per process and memoized — capability gating is a
// read-only predicate and the underlying probes (nvidia-smi, /proc) are not
// free.
var capabilityFactsFn = defaultCapabilityFacts

var (
	capabilityFactsOnce  sync.Once
	capabilityFactsValue hostreqspec.CapabilityFacts
)

func defaultCapabilityFacts() hostreqspec.CapabilityFacts {
	capabilityFactsOnce.Do(func() {
		snap, err := hostinventory.Collect(context.Background())
		if err != nil {
			// A collection failure must not turn into a false "requirement met":
			// leave facts zero so GPU/VRAM gates fall to not-applicable (the
			// conservative, CPU-fallback-preserving outcome).
			capabilityFactsValue = hostreqspec.CapabilityFacts{Arch: snap.Arch}
			return
		}
		capabilityFactsValue = capabilityFactsFromSnapshot(snap)
	})
	return capabilityFactsValue
}

// capabilityFactsFromSnapshot projects a host inventory snapshot onto the
// dependency-free CapabilityFacts the evaluator consumes.
func capabilityFactsFromSnapshot(snap hostinventory.Snapshot) hostreqspec.CapabilityFacts {
	facts := hostreqspec.CapabilityFacts{
		Arch:              snap.Arch,
		RAMGb:             bytesToGiB(snap.Memory.TotalBytes),
		InitSystem:        snap.InitSystem,
		SessionType:       snap.SessionType,
		DisplayManager:    snap.DisplayManager,
		WaylandAttainable: snap.Wayland.Attainable,
	}
	var maxVRAM uint64
	for _, gpu := range snap.GPUs {
		facts.HasGPU = true
		if gpu.VRAMBytes > maxVRAM {
			maxVRAM = gpu.VRAMBytes
		}
	}
	facts.MaxVRAMGb = bytesToGiB(maxVRAM)
	return facts
}

func bytesToGiB(b uint64) float64 {
	return float64(b) / (1 << capabilityParameterA)
}

// effectiveCapability returns the capability gate that applies to a tool: the
// manifest's `requires` (the platform-defined gate) takes precedence; absent
// that, the scenario declaration's `requires` is honored. Either being nil/zero
// means "no gate".
func effectiveCapability(manifest *hostreqspec.CapabilityRequirement, declaration *hostreqspec.CapabilityRequirement) *hostreqspec.CapabilityRequirement {
	if !manifest.IsZero() {
		return manifest
	}
	if !declaration.IsZero() {
		return declaration
	}
	return nil
}
