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
// resources and the self-host guard produce Skip; a routed DB-isolation phase
// that cannot prove routed eligibility is refused fail-closed (the restart
// fallback was deleted); everything else Runs.
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

	// 2. Self-host guard: a needed-but-absent surface requires a clobbering
	//    `scenario start`, which against our own scenario would SIGTERM the
	//    process running the suite. Never do it — reuse a live surface or skip.
	//
	//    When the phase defers its lifecycle decision (LifecycleDecisionDeferred),
	//    only the surface start is gated here — the phase decides routed-vs-refuse
	//    for itself with runtime data the manifest lacks, and enforces the
	//    self-host guard internally.
	missingSurf := missingSurfaces(caps, rc)
	startNeeded := len(missingSurf) > 0
	if startNeeded && rc.TargetIsSelf {
		return Verdict{
			Kind:        VerdictSkip,
			Reason:      joinReason(selfHostReason(caps, missingSurf), rc.RoutedReason),
			Remediation: selfHostRemediation(caps),
		}
	}

	// A phase that owns its own lifecycle decision has cleared the surface and
	// resource gates — let it run and decide routed-vs-refuse-vs-skip itself.
	if caps.LifecycleDecisionDeferred {
		return Verdict{Kind: VerdictRun}
	}

	// 3. Routed DB-isolation phase whose routed eligibility is not proven: the
	//    restart-based fallback was deleted, so it cannot obtain an isolated test
	//    database and is refused fail-closed rather than run mutations against
	//    real data.
	if caps.DBIsolation == DBIsolationRouted && !rc.RoutedEligible {
		return Verdict{
			Kind: VerdictSkip,
			Reason: joinReason(
				fmt.Sprintf("%s cannot prove routed test-DB isolation and the restart fallback was removed", phaseLabel(caps)),
				rc.RoutedReason),
			Remediation: "Wire the routed test-DB seams (run storage-health to see which are missing) so the phase can isolate in place.",
		}
	}

	// 4. Routed-eligible DB-isolation phase: runs in place via the routed path.
	if caps.DBIsolation == DBIsolationRouted && rc.RoutedEligible {
		return Verdict{
			Kind:   VerdictRun,
			Reason: joinReason(fmt.Sprintf("%s runs on the routed test-DB path — no restart", phaseLabel(caps)), rc.RoutedReason),
		}
	}

	// 5. Plain phase (static, or a surface phase whose surface is already live).
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

func selfHostReason(caps PhaseCapabilities, missingSurf []string) string {
	if len(missingSurf) > 0 {
		return fmt.Sprintf(
			"%s needs the target's %s surface(s), which are not live; starting the target would terminate this self-test process",
			phaseLabel(caps), strings.Join(missingSurf, "+"))
	}
	return fmt.Sprintf("%s would mutate the target lifecycle during a self-test", phaseLabel(caps))
}

func selfHostRemediation(caps PhaseCapabilities) string {
	return "Keep the target's required surfaces live before the self-test so they can be reused, or run this phase against a different target scenario."
}
