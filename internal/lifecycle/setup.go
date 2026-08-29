package lifecycle

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/scenario"
)

// forceSetupFor reports whether the scenario identified by slug should be
// force-rebuilt. A blank ForceSetupScenario means "force every scenario in this
// start"; a specific value forces only that scenario (and its self-restart).
// This is the single definition of force-setup scope, shared by the top-level
// start path and the dependency loop.
func forceSetupFor(opts StartOptions, slug string) bool {
	return opts.ForceSetup && (opts.ForceSetupScenario == "" || opts.ForceSetupScenario == slug)
}

// setupNeededCached returns component build freshness, memoized through cache.
func (r *Runner) setupNeededCached(item scenario.Scenario, force bool, session *startSession) (bool, []string, error) {
	ctx := context.Background()
	if session != nil {
		ctx = session.context()
	}
	if session == nil {
		return r.evaluateSetupChecksContext(ctx, item, force)
	}
	return session.setupNeeded(item.Slug+"@"+item.Variant, force, func() (bool, []string, error) {
		return r.evaluateSetupChecksContext(ctx, item, force)
	})
}

// freshnessStaleCached is the reuse-decision spelling of the same component
// build freshness authority used by setupNeededCached.
func (r *Runner) freshnessStaleCached(item scenario.Scenario, force bool, session *startSession) (bool, []string, error) {
	ctx := context.Background()
	if session != nil {
		ctx = session.context()
	}
	if session == nil {
		return r.evaluateSetupChecksContext(ctx, item, force)
	}
	return session.setupNeeded(item.Slug+"@"+item.Variant, force, func() (bool, []string, error) {
		return r.evaluateSetupChecksContext(ctx, item, force)
	})
}

// SetupNeeded reports whether a declared component build is stale.
func (r *Runner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecks(item, force)
}

func (r *Runner) evaluateSetupChecks(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecksContext(context.Background(), item, force)
}

func (r *Runner) evaluateSetupChecksContext(ctx context.Context, item scenario.Scenario, force bool) (bool, []string, error) {
	reasons := []string{}
	if force {
		reasons = append(reasons, "Forced rebuild (restart)")
	}
	setupNeeded := force
	componentNames := make([]string, 0, len(item.Manifest.Components))
	for name := range item.Manifest.Components {
		componentNames = append(componentNames, name)
	}
	slices.Sort(componentNames)
	deps := defaultHostProbeDeps()
	for _, name := range componentNames {
		if err := ctx.Err(); err != nil {
			return false, nil, err
		}
		component := item.Manifest.Components[name]
		if strings.TrimSpace(component.Build.Reuse) != "" {
			continue
		}
		artifacts, err := componentFreshnessArtifactsContext(ctx, item.Path, r.Root, component, deps)
		if err != nil {
			return false, nil, fmt.Errorf("component %s freshness: %w", name, err)
		}
		for _, artifact := range artifacts {
			verdict, err := r.evaluateArtifactFreshnessContext(ctx, artifact, deps)
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
