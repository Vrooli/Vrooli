package runner

import (
	"context"

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
}

// Enforcement names — the platform-neutral vocabulary workspace-sandbox
// reports (api/internal/driver/probe.go). Duplicated here because the two
// scenarios are separate Go modules; the parity is a wire contract.
const (
	EnforcementFilesystemWriteContainment = "filesystem-write-containment"
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
// network is denied. Missing any of these means protected mode is
// degraded (the run still proceeds, but the gap is surfaced loudly).
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
// nil containment (or backend "none") returns all of them. An empty
// result means protected mode is fully honored.
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
// The selector type-asserts the factory to this interface so protected-mode
// capability probing degrades gracefully when the factory cannot report
// (e.g. test doubles).
type SandboxContainmentReporter interface {
	// ContainmentFor reports the containment the given sandbox actually
	// enforces. Returns a nil report (ok=false) when it cannot be resolved.
	ContainmentFor(ctx context.Context, sandboxID uuid.UUID) (*Containment, bool)
}
