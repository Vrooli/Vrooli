package host

import (
	"strings"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/hostinventory"
)

func NewRuntimeIntegrityCheck(collector hostinventory.Collector) checks.Check {
	return &inventoryCheck{
		id:          "host-runtime-integrity",
		title:       "Host Runtime Integrity",
		description: "Detects runtime tools that exist but cannot communicate with their host capability stack.",
		importance:  "A callable binary with a failing backing driver or daemon is a strong capability-drift signal.",
		collector:   collector,
		run:         runRuntimeIntegrity,
	}
}

func runRuntimeIntegrity(inv hostinventory.HostInventory) checks.Result {
	var evidence []map[string]any
	critical := 0
	warning := 0
	for _, runtime := range inv.Runtimes {
		if runtime.Path == "" {
			continue
		}
		if runtime.Callable {
			continue
		}
		item := map[string]any{"kind": "runtime_not_callable", "runtime": runtime}
		if strings.Contains(strings.ToLower(runtime.Name), "smi") {
			critical++
		} else {
			warning++
		}
		evidence = append(evidence, item)
	}
	if len(evidence) == 0 {
		return okResult("Detected host runtime tools are callable", inv)
	}
	return checks.Result{
		Status:  statusFromCounts(critical, warning),
		Message: summarizeEvidence("Runtime/device stack mismatch", critical, warning),
		Details: baseDetails(inv, evidence, []string{
			"Compare runtime command failures with device binding and kernel module state.",
			"Check whether the runtime expects a driver or daemon that is missing for the current boot.",
		}),
	}
}
