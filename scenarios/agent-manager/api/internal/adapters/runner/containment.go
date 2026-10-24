package runner

import (
	"context"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Containment mirrors the workspace-sandbox process-containment report
// (api/internal/types/types.go::SandboxContainment). It is the wire
// contract agent-manager reads to know what isolation a sandbox actually
// enforces, rather than inferring it from GOOS or the driver id.
type Containment struct {
	// Level is the required containment level ("none" | "preferred" |
	// "required").
	Level string
	// Backend is the containment backend that carries out execs ("bwrap"
	// on Linux, "none" when execution falls through to the direct path).
	Backend string
	// Enforcements lists the platform-neutral guarantees the backend
	// provides for this sandbox.
	Enforcements []string
	// WritePolicy is the persisted sandbox policy, not merely a backend
	// capability. It prevents reuse of an unconstrained sandbox for a run
	// whose creation or continuation requests a constrained workspace.
	WritePolicy *domain.WorkspaceWritePolicy
}

// Enforcement names — the platform-neutral vocabulary workspace-sandbox
// reports (api/internal/driver/probe.go). Duplicated here because the two
// scenarios are separate Go modules; the parity is a wire contract.
const (
	EnforcementFilesystemWriteContainment = "filesystem-write-containment"
	EnforcementWorkspaceWritePolicy       = "workspace-write-policy"
	EnforcementPolicyFiles                = "read-only-policy-files"
	EnforcementNetworkDeny                = "network-deny"
	EnforcementPIDNamespace               = "pid-namespace"
	EnforcementPathIllusion               = "path-illusion"

	// EnforcementNetworkLoopbackOnly is the loopback-only network
	// guarantee the "localhost"/vrooli-aware profile implies. No current
	// backend claims it — a "localhost" profile actually grants
	// unrestricted network — so its absence is surfaced on every
	// protected launch (see emitContainmentGapWarn).
	EnforcementNetworkLoopbackOnly = "network-loopback-only"
)

// protectedModeEnforcements are the guarantees a protected-mode run
// depends on: the agent's writes stay contained in the overlay and its
// backend supports network denial. Missing any of these refuses protected
// launch. Backend capability does not prove the selected process network mode.
var protectedModeEnforcements = []string{
	EnforcementFilesystemWriteContainment,
	EnforcementNetworkDeny,
}

// HasEnforcement reports whether the named enforcement is present.
func (c *Containment) HasEnforcement(name string) bool {
	if c == nil {
		return false
	}
	for _, e := range c.Enforcements {
		if e == name {
			return true
		}
	}
	return false
}

// MissingProtectedEnforcements returns the protected-mode-required
// enforcements this containment does NOT provide, in a stable order. A
// nil containment returns all of them. The launcher separately checks backend
// and level; these names alone are not proof of runtime enforcement.
func (c *Containment) MissingProtectedEnforcements() []string {
	var missing []string
	for _, want := range protectedModeEnforcements {
		if !c.HasEnforcement(want) {
			missing = append(missing, want)
		}
	}
	return missing
}

// SandboxContainmentReporter is optionally implemented by a
// SandboxLauncherFactory that can report a sandbox's enforced containment.
// Protected launches require this interface and a current report. Tracking
// callers can continue without a report, without claiming containment.
type SandboxContainmentReporter interface {
	// ContainmentFor reports the containment the given sandbox actually
	// enforces. Returns a nil report (ok=false) when it cannot be resolved.
	ContainmentFor(ctx context.Context, sandboxID uuid.UUID) (*Containment, bool)
}
