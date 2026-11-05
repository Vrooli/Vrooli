package lifecycle

import (
	"context"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// preflightStartTree enforces the host requirements of the start tree (see
// startTreeScenarioPaths) and records the covered paths, so executeStart does
// not repeat them. Optional host capabilities are skipped by runtime.Enforce
// unless an operator opts in, so a restart whose tree declares only optional
// requirements does not pay the host-catalog cost.
func (r *Runner) preflightStartTree(item scenario.Scenario, opts *StartOptions) error {
	treePaths, err := r.startTreeScenarioPaths(opts.Context, item)
	if err != nil {
		return err
	}
	restart := opts.stopFirst || opts.restartAfterStop
	if !restart || scenarioTreeHasRequiredHostRequirements(item, treePaths) {
		if err := r.enforceScenarioHostRequirementsTree(item, treePaths); err != nil {
			return err
		}
	}
	opts.hostRequirementsPreflighted = make(map[string]struct{}, len(treePaths))
	for _, path := range treePaths {
		opts.hostRequirementsPreflighted[path] = struct{}{}
	}
	return nil
}

// prepareRestartBeforeStop does everything an explicit restart can do while
// the current instance still serves: resolve the source layout (an engaged
// live instance materializes its serving tree), preflight host requirements,
// and build replacement artifacts when the build is stale or forced. Builds
// publish through staged swaps, so the running instance keeps its files until
// the new ones are complete. Any failure returns before the stop: the running
// instance keeps serving and the error names the failed step.
func (r *Runner) prepareRestartBeforeStop(name string, opts *StartOptions) error {
	ctx := ctxOrBackground(opts.Context)
	item, err := r.loadScenario(name, opts.CustomPath)
	if err != nil {
		return err
	}
	item.Variant = opts.Variant
	if item, err = r.withSourceLayout(item); err != nil {
		return err
	}
	if err := r.preflightStartTree(item, opts); err != nil {
		return fmt.Errorf("preflight host requirements for scenario %q: %w; the running instance was not stopped", name, err)
	}
	view, err := r.lookupRegistryRuntime(ctx, item)
	if err != nil {
		return err
	}
	if !view.Authoritative || string(view.Instance.Status) == scenarioruntime.StatusFailed {
		// Nothing healthy is serving: there is nothing to keep up while
		// building, so the ordinary post-stop start builds.
		return nil
	}
	force := forceSetupFor(*opts, item.Slug)
	needed, reasons, err := r.setupNeededCached(item, force, nil)
	if err != nil {
		return err
	}
	if !needed {
		return nil
	}
	if err := r.admitVariantBuild(ctx, item); err != nil {
		return fmt.Errorf("%w; the running instance was not stopped", err)
	}
	r.logInfo("Building replacement artifacts before stopping the running instance",
		logx.AttrScenario, recordSlug(item), "reasons", reasons)
	if err := r.prepareReplacementArtifacts(ctx, item, view, force, lifecycleOperationRestart); err != nil {
		return fmt.Errorf("%w; the running instance was not stopped", err)
	}
	if force {
		// The forced rebuild is done; the start after the stop reuses it.
		opts.ForceSetup = false
		opts.ForceSetupScenario = ""
	}
	return nil
}

func (r *Runner) prepareReplacementArtifacts(ctx context.Context, item scenario.Scenario, view registryRuntimeView, force bool, operation string) error {
	env := envFromRuntimeView(item.Manifest, view)
	if operation == "" {
		operation = "stage"
	}
	if _, err := r.runWithLifecycleLog(startLifecycleLogContext(instanceLogName(item), operation, "setup"), func(logWriter, childWriter io.Writer) error {
		_, execErr := r.executePhaseDetailed(ctx, item, "setup", env, logWriter, childWriter, force)
		return execErr
	}); err != nil {
		return fmt.Errorf("prepare replacement artifacts for %s: %w", item.Slug, err)
	}
	return nil
}
