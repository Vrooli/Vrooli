package host

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostcapability"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func NewKernelModuleDriftCheck(collector hostinventory.IntegrityCollector) checks.Check {
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
		return naResult("Kernel/module drift is not applicable on this platform", "native kernel module inventory", inv)
	}
	if inv.Kernel.Release != "" && !inv.Kernel.ModuleTreePresent {
		critical++
		evidence = append(evidence, map[string]any{"kind": "missing_running_kernel_module_tree", "kernelRelease": inv.Kernel.Release})
	}
	for _, drift := range inv.Packages.KernelModuleDrift {
		warning++
		evidence = append(evidence, map[string]any{"kind": "package_targets_other_kernel", "package": drift, "runningKernel": inv.Kernel.Release})
	}
	invariants, err := hostcapability.EmbeddedSafeguardInvariants("nvidia-driver")
	if err != nil {
		warning++
		evidence = append(evidence, map[string]any{"kind": "invariant_declaration_unavailable", "error": err.Error()})
	}
	registry := hostcapability.NewRegistry(hostcapability.AptProvider{}, hostcapability.DarwinProvider{})
	for _, driver := range inv.Packages.Drivers {
		facts := hostcapability.Facts{
			OS:              inv.Platform,
			VendorID:        driver.VendorID,
			DriverPackage:   firstDriverPackage(driver),
			KernelRelease:   inv.Kernel.Release,
			ExpectedPackage: driver.ExpectedModulePackage,
			PackageNames:    packageNames(driver.InstalledPackages),
		}
		if driver.Candidate != nil && driver.Candidate.Available {
			facts.CandidatePackageNames = []string{driver.Candidate.Name}
		}
		results := hostcapability.Evaluate(context.Background(), registry, invariants, facts)
		for _, result := range results {
			evidence = append(evidence, map[string]any{
				"kind":        "capability_invariant",
				"invariantId": result.InvariantID,
				"verdict":     result.Verdict,
				"reason":      result.Reason,
				"evidence":    result.Evidence,
			})
			switch result.Verdict {
			case hostcapability.Failed:
				critical++
			case hostcapability.Undetermined, hostcapability.SatisfiedStructurally:
				warning++
			}
			if result.InvariantID == "module-loadable-on-running-kernel" && result.Verdict == hostcapability.Failed {
				evidence = append(evidence, map[string]any{
					"kind":            "missing_nvidia_module_package",
					"severity":        "critical",
					"expectedPackage": result.Evidence["expectedPackage"],
					"runningKernel":   inv.Kernel.Release,
					"candidate":       driver.Candidate,
				})
			}
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

func firstDriverPackage(driver hostinventory.DriverPackageState) string {
	for _, packageInfo := range driver.InstalledPackages {
		if hostcapability.IsNvidiaDriverPackage(packageInfo.Name) {
			return packageInfo.Name
		}
	}
	return ""
}

func packageNames(packages []hostinventory.PackageInfo) []string {
	names := make([]string, 0, len(packages))
	for _, packageInfo := range packages {
		names = append(names, packageInfo.Name)
	}
	return names
}
