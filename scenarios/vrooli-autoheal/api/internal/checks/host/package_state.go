package host

import (
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func NewPackageStateCheck(collector hostinventory.IntegrityCollector) checks.Check {
	return &inventoryCheck{
		id:          "host-package-state",
		title:       "Host Package State",
		description: "Detects package-manager states that can leave host capabilities partially updated.",
		importance:  "Broken, held, or mismatched host packages can leave Vrooli running on a stale or unsafe capability set.",
		collector:   collector,
		run:         runPackageState,
	}
}

func runPackageState(inv hostinventory.HostInventory) checks.Result {
	if inv.ProbeStatus["host"] == hostinventory.IntegrityProbeUnsupported {
		return naResult("Host package state is not applicable on this platform", "native package-manager adapter", inv)
	}
	var evidence []map[string]any
	warning := 0
	if inv.Packages.Manager == "" {
		warning++
		evidence = append(evidence, map[string]any{"kind": "unsupported_package_manager"})
	}
	for _, item := range inv.Packages.BrokenOrHeld {
		warning++
		evidence = append(evidence, map[string]any{"kind": "broken_or_held_package", "package": item})
	}
	if len(evidence) == 0 {
		return okResult("Package manager state has no host-integrity warnings", inv)
	}
	return checks.Result{
		Status:  checks.StatusWarning,
		Message: summarizeEvidence("Package-manager host integrity warning", 0, warning),
		Details: baseDetails(inv, evidence, []string{
			"Inspect package-manager health and held packages before applying host updates.",
			"Resolve interrupted or broken package configuration outside autoheal.",
		}),
	}
}
