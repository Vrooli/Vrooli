package recovery

import (
	"context"
	"strings"

	"tunnel-manager/internal/cmdrunner"
)

// UnitPresence reports whether the cloudflared systemd unit exists on the
// host. It is the self-gate that keeps default-on recovery dormant on a
// tunnel-less host: with no unit to manage there is nothing to restart, so
// Evaluate must not count failures, flap the circuit breaker, or shell out.
//
// Declared at the recovery consumer (like HealthChecker) so the engine never
// imports a host/systemd package directly; main wires the production
// implementation through the cmdrunner seam.
type UnitPresence interface {
	// CloudflaredUnitPresent reports whether systemd knows about
	// cloudflared.service. Presence is the gate — not live readiness —
	// because /ready is false exactly when recovery is most needed.
	CloudflaredUnitPresent(ctx context.Context) bool
}

// alwaysPresent is the nil-presence fallback: it reports the unit as present
// so the gate is a no-op. NewService substitutes it when callers pass a nil
// UnitPresence (tests exercising the restart/backoff paths that don't care
// about the gate).
type alwaysPresent struct{}

func (alwaysPresent) CloudflaredUnitPresent(context.Context) bool { return true }

// systemctlUnitPresence is the production UnitPresence. It asks systemd
// whether cloudflared.service is a known unit file via the shared cmdrunner
// seam — the same boundary recovery uses to actuate the restart — so it
// catches units under both /etc/systemd/system and /lib/systemd/system
// without hardcoding paths (mirrors the ollama-resource-controls detection).
type systemctlUnitPresence struct {
	runner cmdrunner.Runner
}

// NewSystemctlUnitPresence constructs the production UnitPresence.
func NewSystemctlUnitPresence(runner cmdrunner.Runner) UnitPresence {
	return systemctlUnitPresence{runner: runner}
}

func (p systemctlUnitPresence) CloudflaredUnitPresent(ctx context.Context) bool {
	out, err := p.runner(ctx, "systemctl", "list-unit-files", "--no-pager", "--no-legend", "cloudflared.service")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "cloudflared.service")
}
