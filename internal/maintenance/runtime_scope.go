package maintenance

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/network"
)

// RuntimeScopeRef identifies managed serving processes, not the independent
// executors they admitted. The lifecycle caller supplies every active registry
// instance/ref/claim and legacy managed process record while holding its lock.
type RuntimeScopeRef struct {
	Scenario, Variant  string
	InstanceIDs        []string
	PIDs, PGIDs, Ports []int
}

var scopeListeners = network.CaptureTCPListenerPorts

func (c *Controller) RequireRuntimeScopeAbsent(parent context.Context, ref RuntimeScopeRef) error {
	if ref.Scenario == "" || len(ref.InstanceIDs) == 0 {
		return fmt.Errorf("managed runtime identity is incomplete")
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	entries, identities, err := observeScopeProcesses(ctx)
	if err != nil {
		return err
	}
	for pid, entry := range entries {
		if strings.HasPrefix(entry.State, "Z") {
			continue
		}
		identity := identities[pid]
		// Run-labelled processes are independent executors, never authorized
		// signal targets. They still block if they share a recorded service
		// PID/group; otherwise restoring their absent API owner preserves them.
		servingIdentity := identity.RunID == "" && len(identity.Tags) == 0 &&
			(slices.Contains(ref.InstanceIDs, identity.InstanceID) ||
				(identity.Scenario == ref.Scenario && (identity.Variant == ref.Variant || (identity.Variant == "" && ref.Variant == "live"))))
		if slices.Contains(ref.PIDs, pid) || (entry.PGID > 0 && slices.Contains(ref.PGIDs, entry.PGID)) || servingIdentity {
			return fmt.Errorf("managed runtime process %d is still present", pid)
		}
	}
	listeners := scopeListeners()
	for _, port := range ref.Ports {
		state := listeners.Listening(port)
		if !state.Known || state.Listening {
			return fmt.Errorf("managed runtime port %d is listening or unverified", port)
		}
	}
	return ctx.Err()
}
