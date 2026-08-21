package lifecycle

import (
	"fmt"
	"sort"
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

// setupNeededCached returns component build freshness, memoized through cache.
func (r *Runner) setupNeededCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	return r.evaluateSetupChecksCached(item, force, cache)
}

// freshnessStaleCached is the reuse-decision spelling of the same component
// build freshness authority used by setupNeededCached.
func (r *Runner) freshnessStaleCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	return r.evaluateSetupChecksCached(item, force, cache)
}

func (r *Runner) evaluateSetupChecksCached(item scenario.Scenario, force bool, cache setupCheckCache) (bool, []string, error) {
	if cache == nil {
		return r.evaluateSetupChecks(item, force)
	}
	key := fmt.Sprintf("%s@%s|force=%t", item.Slug, item.Variant, force)
	if cached, ok := cache[key]; ok {
		return cached.needed, append([]string(nil), cached.reasons...), nil
	}
	needed, reasons, err := r.evaluateSetupChecks(item, force)
	if err != nil {
		return false, nil, err
	}
	cache[key] = setupCheckResult{needed: needed, reasons: append([]string(nil), reasons...)}
	return needed, append([]string(nil), reasons...), nil
}

// SetupNeeded reports whether a declared component build is stale.
func (r *Runner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecks(item, force)
}

func (r *Runner) evaluateSetupChecks(item scenario.Scenario, force bool) (bool, []string, error) {
	reasons := []string{}
	if force {
		reasons = append(reasons, "Forced rebuild (restart)")
	}
	setupNeeded := force
	componentNames := make([]string, 0, len(item.Manifest.Components))
	for name := range item.Manifest.Components {
		componentNames = append(componentNames, name)
	}
	sort.Strings(componentNames)
	deps := defaultHostProbeDeps()
	for _, name := range componentNames {
		component := item.Manifest.Components[name]
		if strings.TrimSpace(component.Build.Reuse) != "" {
			continue
		}
		artifacts, err := componentFreshnessArtifacts(item.Path, r.Root, component, deps)
		if err != nil {
			return false, nil, fmt.Errorf("component %s freshness: %w", name, err)
		}
		for _, artifact := range artifacts {
			verdict, err := r.evaluateArtifactFreshness(artifact, deps)
			if err != nil {
				return false, nil, err
			}
			if verdict.Stale {
				setupNeeded = true
				reasons = append(reasons, fmt.Sprintf("Component %s: %s", name, verdict.HumanReason))
			}
		}
	}

	return setupNeeded, reasons, nil
}
