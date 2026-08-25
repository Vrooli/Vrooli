package lifecycle

import "github.com/vrooli/vrooli/internal/scenario"

// Plan → execute decomposition of startScenario (plan Phase 2). The runtime
// state is OBSERVED once (observeRuntime, all the IO), the start decision is
// PLANNED purely (planStart, table-tested without processes), and the steps
// are EXECUTED by executeStart. Restart shares the pipeline: it is a start
// whose options carry stopFirst.

// runtimeObservation is everything planStart needs to know about the world:
// the registry view of this instance plus the health and freshness verdicts
// (both only evaluated for an authoritative instance — freshness first, then
// the health probe, preserving the historical evaluation order and cost).
type runtimeObservation struct {
	View           registryRuntimeView
	Healthy        bool
	FreshnessStale bool
}

func (r *Runner) observeRuntime(item scenario.Scenario, forceSetup bool, session *startSession) (runtimeObservation, error) {
	view, err := r.lookupRegistryRuntime(session.context(), item)
	if err != nil {
		return runtimeObservation{}, err
	}
	obs := runtimeObservation{View: view}
	if !view.Authoritative {
		return obs, nil
	}
	// Reuse is gated on FRESHNESS only (not provisioning): a healthy running
	// instance is kept unless a binaries/ui-bundle check is stale.
	stale, _, err := r.freshnessStaleCached(item, forceSetup, session)
	if err != nil {
		return runtimeObservation{}, err
	}
	obs.FreshnessStale = stale
	obs.Healthy = r.isRegistryRuntimeHealthy(item, view)
	return obs, nil
}

func (o runtimeObservation) planInput() startPlanInput {
	return startPlanInput{
		RegistryPresent:       o.View.Present,
		RegistryAuthoritative: o.View.Authoritative,
		Healthy:               o.Healthy,
		FreshnessStale:        o.FreshnessStale,
	}
}

// startPlanInput is the pure decision input for planStart.
type startPlanInput struct {
	// RegistryPresent: some registry instance row exists for this instance
	// (authoritative or not).
	RegistryPresent bool
	// RegistryAuthoritative: the registry instance survived reconciliation
	// (lease fresh, evidence consistent) and speaks for a live runtime.
	RegistryAuthoritative bool
	// Healthy: the authoritative instance passed the strict data-plane probe
	// (isRegistryRuntimeHealthy). Meaningless unless RegistryAuthoritative.
	Healthy bool
	// FreshnessStale: the freshness engine reports a checked artifact stale.
	// Meaningless unless RegistryAuthoritative (only evaluated then).
	FreshnessStale bool
}

// startDecision is the reuse-vs-restart arbitration for one instance.
type startDecision int

const (
	// decisionFreshStart brings the instance up with no prior teardown.
	decisionFreshStart startDecision = iota
	// decisionReuseRunning keeps the live instance and returns AlreadyRunning.
	decisionReuseRunning
	// decisionStopThenStart tears the existing instance down (plus settle
	// delay) before starting: it is running-but-unfit or a stale registry
	// row whose claims would collide.
	decisionStopThenStart
)

// startPlan is planStart's output: the decision plus the reason a running
// instance was not reused (empty for reuse/fresh).
type startPlan struct {
	Decision startDecision
	// RestartReason explains a decisionStopThenStart on an authoritative
	// instance ("unhealthy", "stale", "unhealthy; stale") or names the
	// non-authoritative cleanup ("stale registry instance").
	RestartReason string
}

// planStart is the pure reuse-vs-restart-vs-fresh decision, table-tested
// without processes. Policy:
//   - authoritative + healthy + fresh → reuse
//   - authoritative but unhealthy or stale → stop, settle, start
//   - present but non-authoritative → stop (cleanup leftover claims), settle, start
//   - absent → fresh start
func planStart(in startPlanInput) startPlan {
	if in.RegistryAuthoritative {
		if in.Healthy && !in.FreshnessStale {
			return startPlan{Decision: decisionReuseRunning}
		}
		reason := ""
		if !in.Healthy {
			reason = "unhealthy"
		}
		if in.FreshnessStale {
			if reason != "" {
				reason += "; "
			}
			reason += "stale"
		}
		return startPlan{Decision: decisionStopThenStart, RestartReason: reason}
	}
	if in.RegistryPresent {
		return startPlan{Decision: decisionStopThenStart, RestartReason: "stale registry instance"}
	}
	return startPlan{Decision: decisionFreshStart}
}
