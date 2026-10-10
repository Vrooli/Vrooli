package agentsession

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// LiveCensus reads this host's agent slice.
func LiveCensus() ([]agentscope.Entry, error) {
	path, err := platformgo.SliceCgroup(agentscope.Slice)
	if err != nil {
		return nil, err
	}
	return agentscope.NewReader().Census(agentscope.Ref(path))
}

// liveFrozen reads a session scope's freeze state through the platform seam:
// a unit name is resolved under the agent slice, a "<kind>:<identity>" ref
// is used as written, anything else is "n/a".
func liveFrozen(host *hostFacts) func(scope string) string {
	return func(scope string) string {
		scope = strings.TrimSpace(scope)
		var ref platformgo.ScopeRef
		var err error
		switch {
		case scope == "" || scope == "none":
			return "n/a"
		case strings.Contains(scope, ":"):
			ref, err = platformgo.ParseScopeRef(scope)
		default:
			unit := scope
			if !strings.HasSuffix(unit, agentscope.ScopeSuffix) {
				unit += agentscope.ScopeSuffix
			}
			ref, err = host.scopeRef(unit)
		}
		if err != nil {
			return "n/a"
		}
		frozen, err := platformgo.ScopeFrozen(ref)
		if err != nil {
			return "unknown"
		}
		if frozen {
			return "yes"
		}
		return "no"
	}
}

// LiveReconciler binds a reconciler to this host and, when store is not nil,
// to the registry for backfilling owners.
func LiveReconciler(home string, store OwnerStore, idleAfter time.Duration) Reconciler {
	host := &hostFacts{}
	return Reconciler{
		Census:    LiveCensus,
		Alive:     platformgo.IsPIDRunning,
		Frozen:    liveFrozen(host),
		Host:      host,
		Probes:    LiveProbes(home),
		Store:     store,
		IdleAfter: idleAfter,
	}
}

// ReapRequest is one unattended reap.
type ReapRequest struct {
	Home      string
	DBPath    string
	Policy    ReapPolicy
	Initiator string
	DryRun    bool
}

// ReapOrphaned reconciles the host and stops every eligible orphaned session.
// It is the periodic path (the runtime supervisor, when enabled); the
// operator path is `vrooli agent reap --orphaned`, which renders the same
// plan.
func ReapOrphaned(ctx context.Context, request ReapRequest) ([]Result, error) {
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: request.Home, DBPath: request.DBPath})
	if err != nil {
		return nil, fmt.Errorf("open runtime registry: %w", err)
	}
	defer store.Close()
	leases, err := store.ListEditorLeases(ctx, false)
	if err != nil {
		return nil, err
	}
	guard := scenarioruntime.StartingLeaseGuard{PIDRunning: platformgo.IsPIDRunning}
	if host, hostErr := (hostsession.DefaultProvider{}).Current(ctx, ""); hostErr == nil {
		guard.CurrentBootID = host.BootID
	}
	leases = scenarioruntime.LiveEditorLeases(leases, guard)
	reconciler := LiveReconciler(request.Home, store, 0)
	sessions := reconciler.Sessions(ctx, leases)
	plan := Plan(sessions, request.Policy, time.Now().UTC())
	executor := Executor{
		Stopper:   NewPlatformStopper(),
		Audit:     store,
		Initiator: request.Initiator,
		Recheck: func(runCtx context.Context, session Session) Liveness {
			return Check(runCtx, session.Owner, reconciler.Probes, time.Now())
		},
	}
	return executor.Execute(ctx, plan, request.DryRun), nil
}
