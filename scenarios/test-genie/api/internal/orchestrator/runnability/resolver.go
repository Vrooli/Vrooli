package runnability

import (
	"fmt"
	"sort"
	"strings"
)

// StandardResolver is the production runnability policy. It is stateless and
// pure: the verdict is a deterministic function of the phase capabilities and
// the resolved run context.
type StandardResolver struct{}

// NewResolver returns the production resolver.
func NewResolver() StandardResolver { return StandardResolver{} }

var _ Resolver = StandardResolver{}

// Resolve applies the runnability policy. The order of checks matters: missing
// resources and the self-host guard produce Skip; a non-routed DB-isolation
// phase that still runs produces RunDegraded; everything else Runs.
func (StandardResolver) Resolve(caps PhaseCapabilities, rc RunContext) Verdict {
	// 1. Required resources must be present regardless of surfaces/identity.
	if missing := missingResources(caps, rc); len(missing) > 0 {
		return Verdict{
			Kind: VerdictSkip,
			Reason: fmt.Sprintf("%s requires unavailable resource(s): %s",
				phaseLabel(caps), strings.Join(missing, ", ")),
			Remediation: fmt.Sprintf("Start or install the required resource(s): %s.", strings.Join(missing, ", ")),
		}
	}

	// 2. Does running this phase require a lifecycle mutation against the target?
	//    (a) a needed surface is not live → a start is required (a clobbering
	//        `scenario start --clean-stale`); (b) the phase isolates its DB via
	//        restart because the target is not routed-eligible.
	//
	//    When the phase defers its lifecycle decision (LifecycleDecisionDeferred),
	//    only the surface start counts here — the phase decides for itself, with
	//    runtime data the manifest lacks, whether a restart actually happens, and
	//    enforces the self-host guard internally.
	missingSurf := missingSurfaces(caps, rc)
	startNeeded := len(missingSurf) > 0
	restartForDB := !caps.LifecycleDecisionDeferred &&
		caps.MutatesLifecycle &&
		caps.DBIsolation == DBIsolationRoutedOrRestart &&
		!rc.RoutedEligible
	mutates := startNeeded || restartForDB

	// 3. Self-host guard: a mutation against our own scenario would SIGTERM the
	//    process running the suite. Never do it — reuse a live surface or skip.
	if mutates && rc.TargetIsSelf {
		return Verdict{
			Kind:        VerdictSkip,
			Reason:      joinReason(selfHostReason(caps, missingSurf, restartForDB), rc.RoutedReason),
			Remediation: selfHostRemediation(caps, restartForDB),
		}
	}

	// A phase that owns its own lifecycle decision has cleared the surface and
	// resource gates — let it run and decide routed-vs-restart-vs-skip itself.
	if caps.LifecycleDecisionDeferred {
		return Verdict{Kind: VerdictRun}
	}

	// 4. DB-isolation phase running on the restart fallback (non-self): it runs,
	//    but on the less-preferred path. Surface that as a degraded run so the
	//    operator sees the routed path was not taken and why.
	if caps.DBIsolation == DBIsolationRoutedOrRestart && !rc.RoutedEligible {
		return Verdict{
			Kind: VerdictRunDegraded,
			Reason: joinReason(
				fmt.Sprintf("%s obtains DB isolation via target restart (not routed-eligible)", phaseLabel(caps)),
				rc.RoutedReason),
			Remediation: "Migrate the target to the routed test-DB path (see docs/agent-system/routed-test-db.md) to avoid the restart.",
		}
	}

	// 5. Routed-eligible DB-isolation phase: runs in place via the routed path.
	if caps.DBIsolation == DBIsolationRoutedOrRestart && rc.RoutedEligible {
		return Verdict{
			Kind:   VerdictRun,
			Reason: joinReason(fmt.Sprintf("%s runs on the routed test-DB path — no restart", phaseLabel(caps)), rc.RoutedReason),
		}
	}

	// 6. Plain phase (static, or a surface phase whose surface is already live).
	return Verdict{Kind: VerdictRun}
}

// missingResources returns the sorted set of required resources absent from the
// context.
func missingResources(caps PhaseCapabilities, rc RunContext) []string {
	if len(caps.RequiredResources) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var missing []string
	for _, r := range caps.RequiredResources {
		name := strings.TrimSpace(r)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		if !rc.HasResource(name) {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func selfHostReason(caps PhaseCapabilities, missingSurf []string, restartForDB bool) string {
	switch {
	case len(missingSurf) > 0:
		return fmt.Sprintf(
			"%s needs the target's %s surface(s), which are not live; starting the target would terminate this self-test process",
			phaseLabel(caps), strings.Join(missingSurf, "+"))
	case restartForDB:
		return fmt.Sprintf(
			"%s isolates its database by restarting the target, which would terminate this self-test process",
			phaseLabel(caps))
	default:
		return fmt.Sprintf("%s would mutate the target lifecycle during a self-test", phaseLabel(caps))
	}
}

func selfHostRemediation(caps PhaseCapabilities, restartForDB bool) string {
	if restartForDB {
		return "Migrate test-genie to the routed test-DB path so the DB-isolation phase runs in place, or run this phase against a different target scenario."
	}
	return "Keep the target's required surfaces live before the self-test so they can be reused, or run this phase against a different target scenario."
}
