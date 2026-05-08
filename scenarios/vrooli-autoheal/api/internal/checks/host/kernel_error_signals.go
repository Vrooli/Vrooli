package host

import (
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/hostinventory"
)

func NewKernelErrorSignalsCheck(collector hostinventory.Collector) checks.Check {
	return &inventoryCheck{
		id:          "host-kernel-error-signals",
		title:       "Host Kernel Error Signals",
		description: "Detects recent high-signal kernel and device errors from the host log stream.",
		importance:  "Kernel reset, machine-check, bus, and filesystem signals often precede hard crashes.",
		collector:   collector,
		run:         runKernelErrorSignals,
	}
}

func runKernelErrorSignals(inv hostinventory.HostInventory) checks.Result {
	var evidence []map[string]any
	critical := 0
	warning := 0
	for _, signal := range inv.Signals {
		if signal.Severity == "critical" {
			critical++
		} else {
			warning++
		}
		evidence = append(evidence, map[string]any{"kind": "kernel_signal", "signal": signal})
	}
	if len(evidence) == 0 {
		return okResult("No recent high-signal kernel errors detected", inv)
	}
	return checks.Result{
		Status:  statusFromCounts(critical, warning),
		Message: summarizeEvidence("Recent kernel/device error signals", critical, warning),
		Details: baseDetails(inv, evidence, []string{
			"Review recent kernel signals alongside boot history and host inventory drift.",
			"Correlate repeated device reset or machine-check categories with workloads running before crashes.",
		}),
	}
}
