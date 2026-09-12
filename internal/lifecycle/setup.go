package lifecycle

import (
	"context"
	"fmt"
	"slices"

	"github.com/vrooli/vrooli/internal/scenario"
)

type setupVerdicts struct {
	Stale       bool
	ByComponent map[string]artifactVerdict
	Reason      string
}

func (v setupVerdicts) Any() bool { return v.Stale }

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

// SetupNeeded reports whether a declared component build is stale.
func (r *Runner) SetupNeeded(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecks(item, force)
}

func (r *Runner) evaluateSetupChecks(item scenario.Scenario, force bool) (bool, []string, error) {
	return r.evaluateSetupChecksContext(context.Background(), item, force)
}

func (r *Runner) evaluateSetupChecksContext(ctx context.Context, item scenario.Scenario, force bool) (bool, []string, error) {
	verdicts, err := r.evaluateSetupVerdictsContext(ctx, item, force)
	if err != nil {
		return false, nil, err
	}
	if force {
		return true, []string{"Forced rebuild (restart)"}, nil
	}
	reasons := []string{}
	for _, verdict := range verdicts.ByComponent {
		if verdict.Stale {
			reasons = append(reasons, verdict.HumanReason)
		}
	}
	slices.Sort(reasons)
	return verdicts.Stale, reasons, nil
}

func (r *Runner) evaluateSetupVerdictsContext(ctx context.Context, item scenario.Scenario, force bool) (setupVerdicts, error) {
	verdicts := setupVerdicts{Stale: force, ByComponent: map[string]artifactVerdict{}}
	componentNames := make([]string, 0, len(item.Manifest.Components))
	for name := range item.Manifest.Components {
		componentNames = append(componentNames, name)
	}
	slices.Sort(componentNames)
	deps := r.hostProbeDeps()
	for _, name := range componentNames {
		if err := ctx.Err(); err != nil {
			return setupVerdicts{}, err
		}
		component := item.Manifest.Components[name]
		artifacts, err := componentFreshnessArtifactsContextWithName(ctx, item.Path, r.Root, item.Slug, name, component, deps)
		if err != nil {
			return setupVerdicts{}, fmt.Errorf("component %s freshness: %w", name, err)
		}
		for _, artifact := range artifacts {
			verdict, err := r.evaluateArtifactFreshnessContext(ctx, artifact, deps)
			if err != nil {
				return setupVerdicts{}, err
			}
			if verdict.Stale {
				verdicts.Stale = true
				if _, exists := verdicts.ByComponent[name]; !exists {
					verdicts.ByComponent[name] = artifactVerdict{Target: verdict.Target, Stale: true, Cause: verdict.Cause, File: verdict.File, HumanReason: fmt.Sprintf("Component %s: %s", name, verdict.HumanReason)}
				}
			}
		}
	}
	return verdicts, nil
}
