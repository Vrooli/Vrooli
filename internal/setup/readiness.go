package setup

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

const (
	ReadinessStatusReady       = "ready"
	ReadinessStatusDegraded    = "degraded"
	ReadinessStatusMissing     = "missing"
	ReadinessStatusUnsupported = "unsupported"

	ReadinessSourceInProcess   = "in_process"
	ReadinessSourceUnavailable = "unavailable"
)

// SetupReadiness is setup's own verdict on whether the host is configured.
//
// Setup computes it in process rather than asking the onboarding API, because
// the onboarding scenario may have been stopped by the operator and a local
// completion check that needs a network call is a weaker contract than the
// marker it replaces. Blockers are addresses and item names only; no credential
// value ever reaches this document.
type SetupReadiness struct {
	Status   string   `json:"status"`
	Blockers []string `json:"blockers,omitempty"`
	Source   string   `json:"source"`
	// Reason explains an unavailable source. A verdict from a source that could
	// not be read is never reported as ready.
	Reason string `json:"reason,omitempty"`
}

// verifySetupReadiness reads the same two populations onboarding reads: the
// declared credential inventory and the resolved host requirements. It is the
// IO half; the decision itself is setupReadinessVerdict, which is pure so its
// rules can be asserted without a host census.
func verifySetupReadiness(root string, report vrooliruntime.Report, reportErr error) SetupReadiness {
	inventory, inventoryErr := credentialinventory.Collect(root)
	return setupReadinessVerdict(inventory, inventoryErr, report, reportErr)
}

func setupReadinessVerdict(inventory credentialinventory.Result, inventoryErr error, report vrooliruntime.Report, reportErr error) SetupReadiness {
	verdict := SetupReadiness{Status: ReadinessStatusReady, Source: ReadinessSourceInProcess}
	blockers := map[string]struct{}{}

	if inventoryErr != nil {
		return SetupReadiness{
			Status: ReadinessStatusUnsupported,
			Source: ReadinessSourceUnavailable,
			Reason: fmt.Sprintf("credential inventory could not be read: %v", inventoryErr),
		}
	}
	for _, address := range inventory.RequiredAbsent {
		blockers[address] = struct{}{}
	}

	if reportErr != nil {
		return SetupReadiness{
			Status: ReadinessStatusUnsupported,
			Source: ReadinessSourceUnavailable,
			Reason: fmt.Sprintf("host requirements could not be inspected: %v", reportErr),
		}
	}
	for _, name := range report.MissingRequired {
		if strings.TrimSpace(name) != "" {
			blockers[name] = struct{}{}
		}
	}

	if len(blockers) > 0 {
		verdict.Status = ReadinessStatusMissing
		verdict.Blockers = make([]string, 0, len(blockers))
		for name := range blockers {
			verdict.Blockers = append(verdict.Blockers, name)
		}
		slices.Sort(verdict.Blockers)
		return verdict
	}
	if len(inventory.RequiredAbsent) == 0 && len(report.MissingOptional) > 0 {
		verdict.Status = ReadinessStatusDegraded
	}
	return verdict
}

// readinessRemediation names the single command that closes the verdict.
func readinessRemediation(verdict SetupReadiness) string {
	switch verdict.Status {
	case ReadinessStatusReady:
		return "Setup and onboarding configuration are complete and verified."
	case ReadinessStatusDegraded:
		return "Setup and onboarding configuration are complete. Optional items remain unresolved; they are listed in `vrooli setup status`."
	default:
		named := ""
		if len(verdict.Blockers) > 0 {
			named = " Unresolved: " + strings.Join(verdict.Blockers, ", ") + "."
		}
		return "Configuration is not verified." + named + " Run `vrooli setup --include-optional --maintenance-window --sudo-mode=ask --onboarding=auto` and answer every question inside Vrooli."
	}
}
