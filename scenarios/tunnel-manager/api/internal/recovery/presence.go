package recovery

import (
	"context"
	"encoding/json"
	"fmt"

	"tunnel-manager/internal/cmdrunner"
)

// UnitPresence reports whether the Vrooli-managed cloudflared resource exists
// on the host. It is the self-gate that keeps default-on recovery dormant on a
// tunnel-less host.
//
// Declared at the recovery consumer so the engine never owns host service
// mechanics; the API wires the production implementation through the control
// plane command seam.
type UnitPresence interface {
	// CloudflaredUnitPresent reports whether the managed cloudflared resource is
	// registered. Presence is the gate — not live readiness —
	// because /ready is false exactly when recovery is most needed.
	CloudflaredUnitPresent(ctx context.Context) bool
}

// alwaysPresent is the nil-presence fallback: it reports the resource as present
// so the gate is a no-op. NewService substitutes it when callers pass a nil
// UnitPresence (tests exercising the restart/backoff paths that don't care
// about the gate).
type alwaysPresent struct{}

func (alwaysPresent) CloudflaredUnitPresent(context.Context) bool { return true }

type unavailableLifecycle struct{}

func (unavailableLifecycle) CloudflaredUnitPresent(context.Context) bool { return false }
func (unavailableLifecycle) Restart(context.Context) error {
	return fmt.Errorf("managed cloudflared lifecycle is unavailable")
}

// ManagedServiceLifecycle is the resource lifecycle seam used by recovery.
type ManagedServiceLifecycle interface {
	UnitPresence
	Restart(context.Context) error
}

type controlPlaneLifecycle struct {
	runner cmdrunner.Runner
}

// NewControlPlaneLifecycle constructs the production lifecycle adapter. The
// control plane owns resource supervision; tunnel-manager never invokes an OS
// service manager or privilege escalation directly.
func NewControlPlaneLifecycle(runner cmdrunner.Runner) ManagedServiceLifecycle {
	return controlPlaneLifecycle{runner: runner}
}

func (p controlPlaneLifecycle) CloudflaredUnitPresent(ctx context.Context) bool {
	out, err := p.runner(ctx, "vrooli", "resource", "status", "cloudflared", "--json")
	if err != nil {
		return false
	}
	var status struct {
		Installed bool `json:"installed"`
		Running   bool `json:"running"`
		Resource  struct {
			Resource struct {
				Registered bool `json:"registered"`
			} `json:"resource"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return false
	}
	return status.Resource.Resource.Registered || status.Installed || status.Running
}

func (p controlPlaneLifecycle) Restart(ctx context.Context) error {
	_, err := p.runner(ctx, "vrooli", "resource", "restart", "cloudflared")
	return err
}
