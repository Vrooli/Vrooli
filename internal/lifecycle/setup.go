package lifecycle

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/scenario"
)

// setupCheckCache memoizes SetupNeeded results within a single top-level Start
// invocation. Dependency trees frequently re-evaluate the same scenario's setup
// conditions (once for the reuse decision, once before running the setup phase,
// and once per parent that shares the dependency), each re-walking the same
// replace dirs. The cache is created per Start and threaded through the start
// path; nil means "no memoization" (each call recomputes).
type setupCheckCache map[string]setupCheckResult

type setupCheckResult struct {
	needed  bool
	reasons []string
}

// forceSetupFor reports whether the scenario identified by slug should be
// force-rebuilt. A blank ForceSetupScenario means "force every scenario in this
// start"; a specific value forces only that scenario (and its self-restart).
// This is the single definition of force-setup scope, shared by the top-level
// start path and the dependency loop.
func forceSetupFor(opts StartOptions, slug string) bool {
	return opts.ForceSetup && (opts.ForceSetupScenario == "" || opts.ForceSetupScenario == slug)
}

// setupNeededCached returns SetupNeeded, memoized through cache when non-nil.
// The cache key includes the variant and the force flag so a forced and an
// unforced evaluation of the same scenario never alias.
func (r *Runner) setupNeededCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	return r.evaluateSetupChecksCached(item, force, false, cache)
}

// freshnessStaleCached evaluates only the freshness checks (binaries/ui-bundle),
// memoized through cache when non-nil. This is the verb the reuse decision
// consults: a running, healthy instance is only stopped+rebuilt when a
// freshness check is stale, never because a provisioning check (data, files,
// directories, resources, dependencies) reports "not yet populated".
func (r *Runner) freshnessStaleCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	return r.evaluateSetupChecksCached(item, force, true, cache)
}

func (r *Runner) evaluateSetupChecksCached(item scenario.Scenario, force, freshnessOnly bool, cache setupCheckCache) (bool, []string, error) {
	if cache == nil {
		return r.evaluateSetupChecks(item, force, freshnessOnly)
	}
	key := fmt.Sprintf("%s@%s|force=%t|fresh=%t", item.Slug, item.Variant, force, freshnessOnly)
	if cached, ok := cache[key]; ok {
		return cached.needed, append([]string(nil), cached.reasons...), nil
	}
	needed, reasons, err := r.evaluateSetupChecks(item, force, freshnessOnly)
	if err != nil {
		return false, nil, err
	}
	cache[key] = setupCheckResult{needed: needed, reasons: append([]string(nil), reasons...)}
	return needed, append([]string(nil), reasons...), nil
}

// SetupNeeded reports whether the scenario's setup phase must run before/at
// start. It evaluates every declared setup condition — both provisioning
// (ensure-if-missing) and freshness (rebuild-if-changed). It is the gate for
// running the setup phase on an actual (re)start, NOT the gate for bouncing a
// healthy running process (that is freshnessStaleCached).
func (r *Runner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecks(item, force, false)
}

// isFreshnessCheck reports whether a setup condition type is a freshness verb
// (rebuild-if-content-changed) as opposed to a provisioning verb
// (ensure-if-missing). Only freshness checks may restart a running healthy
// process; provisioning checks (resources, dependencies, data, files,
// directories) ensure state exists and must never trigger a bounce. The "cli"
// type is handled (and short-circuited to false) by evaluateSetupCheck.
func isFreshnessCheck(checkType string) bool {
	switch strings.TrimSpace(checkType) {
	case "", "binaries", "ui-bundle":
		return true
	default:
		return false
	}
}

func (r *Runner) evaluateSetupChecks(item scenario.Scenario, force, freshnessOnly bool) (bool, []string, error) {
	reasons := []string{}
	if force {
		reasons = append(reasons, "Forced rebuild (restart)")
	}

	checks := item.Manifest.Lifecycle.Setup.Condition
	if checks == nil || len(checks.Checks) == 0 {
		return force, reasons, nil
	}

	setupNeeded := force
	for _, check := range checks.Checks {
		if freshnessOnly && !isFreshnessCheck(check.Type) {
			continue
		}
		needed, reason, err := r.evaluateSetupCheck(item, check)
		if err != nil {
			return false, nil, err
		}
		if needed {
			setupNeeded = true
			if reason != "" {
				reasons = append(reasons, reason)
			}
		}
	}
	return setupNeeded, reasons, nil
}

func (r *Runner) evaluateSetupCheck(item scenario.Scenario, check scenario.ConditionCheck) (bool, string, error) {
	switch strings.TrimSpace(check.Type) {
	case "", "binaries":
		return r.binariesFreshness(item, check)
	case "cli":
		// Runtime lifecycle freshness intentionally ignores installed CLI state.
		//
		// Scenario CLI freshness is owned by internal/cliinstall and enforced at
		// command boundaries (for example `vrooli scenario ...`), where the CLI
		// can be refreshed before execution. Treating stale installed CLIs as a
		// runtime setup input caused dependency restart loops because scenario
		// setup phases generally build API/UI artifacts, not scenario CLIs.
		return false, "", nil
	case "ui-bundle":
		return r.uiBundleFreshness(item, check)
	case "resources":
		return resourcesNeedSetup(r.Home, item.Path, check), "Resources not populated", nil
	case "dependencies":
		return dependenciesNeedSetup(item.Path, check), "Dependencies not installed", nil
	case "data":
		return dataNeedsSetup(item.Path, check), "Data directory missing", nil
	case "files":
		return filesNeedSetup(item.Path, check), "Required files missing", nil
	case "directories":
		return directoriesNeedSetup(item.Path, check), "Missing directories", nil
	default:
		return false, "", fmt.Errorf("unsupported setup condition type %q: only native lifecycle setup checks are supported", check.Type)
	}
}
