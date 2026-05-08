package host

import (
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/hostinventory"
)

func NewKernelModuleDriftCheck(collector hostinventory.Collector) checks.Check {
	return &inventoryCheck{
		id:          "host-kernel-module-drift",
		title:       "Host Kernel Module Drift",
		description: "Detects when the running kernel and installed module inventory no longer agree.",
		importance:  "Kernel/module drift can disable device drivers after updates and can contribute to hard crashes.",
		collector:   collector,
		run:         runKernelModuleDrift,
	}
}

func runKernelModuleDrift(inv hostinventory.HostInventory) checks.Result {
	var evidence []map[string]any
	critical := 0
	warning := 0
	if inv.Platform != "linux" {
		warning++
		evidence = append(evidence, map[string]any{"kind": "unsupported_platform", "platform": inv.Platform})
	}
	if inv.Kernel.Release != "" && !inv.Kernel.ModuleTreePresent {
		critical++
		evidence = append(evidence, map[string]any{"kind": "missing_running_kernel_module_tree", "kernelRelease": inv.Kernel.Release})
	}
	for _, drift := range inv.Packages.KernelModuleDrift {
		warning++
		evidence = append(evidence, map[string]any{"kind": "package_targets_other_kernel", "package": drift, "runningKernel": inv.Kernel.Release})
	}
	if len(evidence) == 0 {
		return okResult("Running kernel has matching module inventory", inv)
	}
	return checks.Result{
		Status:  statusFromCounts(critical, warning),
		Message: summarizeEvidence("Kernel/module inventory drift", critical, warning),
		Details: baseDetails(inv, evidence, []string{
			"Compare the running kernel release with installed module packages.",
			"Confirm required kernel module packages are installed for the running kernel before reboot-sensitive workloads run.",
		}),
	}
}
