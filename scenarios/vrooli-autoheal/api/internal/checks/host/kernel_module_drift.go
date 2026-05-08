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
		return okResult("Kernel/module drift check is not applicable on this platform", inv)
	}
	if inv.Kernel.Release != "" && !inv.Kernel.ModuleTreePresent {
		critical++
		evidence = append(evidence, map[string]any{"kind": "missing_running_kernel_module_tree", "kernelRelease": inv.Kernel.Release})
	}
	for _, drift := range inv.Packages.KernelModuleDrift {
		warning++
		evidence = append(evidence, map[string]any{"kind": "package_targets_other_kernel", "package": drift, "runningKernel": inv.Kernel.Release})
	}
	for _, driver := range inv.Packages.Drivers {
		if driver.Vendor != "nvidia" {
			continue
		}
		evidence = append(evidence, map[string]any{"kind": "nvidia_kernel_package_state", "driver": driver, "runningKernel": inv.Kernel.Release})
		if driver.MissingModulePackage != "" {
			severity := "warning"
			if driver.Candidate != nil && driver.Candidate.Available {
				severity = "critical"
				critical++
			} else {
				warning++
			}
			evidence = append(evidence, map[string]any{
				"kind":            "missing_nvidia_module_package",
				"severity":        severity,
				"expectedPackage": driver.MissingModulePackage,
				"runningKernel":   inv.Kernel.Release,
				"candidate":       driver.Candidate,
			})
		}
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
			"When a missing NVIDIA module package has an apt candidate, generate an operator-approved remediation script instead of running package commands automatically.",
		}),
	}
}
