package host

import (
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
)

func NewCapabilityDriftCheck(collector hostinventory.Collector) checks.Check {
	return &inventoryCheck{
		id:          "host-capability-drift",
		title:       "Host Capability Drift",
		description: "Summarizes cross-signal host capability drift from inventory evidence.",
		importance:  "Multiple weaker host signals become important when they point to the same capability surface changing.",
		collector:   collector,
		run:         runCapabilityDrift,
	}
}

func runCapabilityDrift(inv hostinventory.HostInventory) checks.Result {
	var evidence []map[string]any
	critical := 0
	warning := 0
	if inv.Kernel.Release != "" && !inv.Kernel.ModuleTreePresent {
		critical++
		evidence = append(evidence, map[string]any{"kind": "missing_module_tree", "kernelRelease": inv.Kernel.Release})
	}
	for _, runtime := range inv.Runtimes {
		if runtime.Path != "" && !runtime.Callable {
			warning++
			evidence = append(evidence, map[string]any{"kind": "runtime_failure", "runtime": runtime})
		}
	}
	for _, device := range inv.Devices {
		if capabilityDevice(device) && device.BoundDriver == "" && len(device.AvailableModules) > 0 {
			warning++
			evidence = append(evidence, map[string]any{"kind": "device_binding_gap", "device": device})
		}
	}
	if len(inv.Signals) >= 5 {
		warning++
		evidence = append(evidence, map[string]any{"kind": "kernel_signal_volume", "count": len(inv.Signals)})
	}
	if len(evidence) == 0 {
		return okResult("No host capability drift detected", inv)
	}
	return checks.Result{
		Status:  statusFromCounts(critical, warning),
		Message: summarizeEvidence("Host capability drift", critical, warning),
		Details: baseDetails(inv, evidence, []string{
			"Use host inventory, package state, and recent kernel signals together when triaging crashes.",
			"Treat this as an early warning before attempting workload-specific fixes.",
		}),
	}
}
