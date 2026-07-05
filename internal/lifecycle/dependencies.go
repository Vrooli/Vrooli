package lifecycle

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/logx"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type dependencyDecision struct {
	policy            string
	freshnessPolicy   string
	skip              bool
	continueOnFailure bool
}

// Timeout/interval values for the dependency waits live in the lifecycle
// policy table (await.go: resourceReadyPolicy, dependencyLockPolicy).

// dependencyRestartReason explains why a dependency is being (re)started rather
// than reused. It is only ever called once the reuse fast-path has been
// rejected, so at least one of {not running, unhealthy, setup needed} always
// holds — there is no "nothing changed" arm to fall through to.
func dependencyRestartReason(running bool, healthy bool, setupNeeded bool, setupReasons []string) string {
	reasons := make([]string, 0, 3)
	switch {
	case !running:
		reasons = append(reasons, "not running")
	case !healthy:
		reasons = append(reasons, "unhealthy")
	}
	if setupNeeded {
		if len(setupReasons) > 0 {
			reasons = append(reasons, "setup needed: "+strings.Join(setupReasons, "; "))
		} else {
			reasons = append(reasons, "setup needed")
		}
	}
	return strings.Join(reasons, "; ")
}

func resourceDependencyStartReason(status resourcecontrol.Status) string {
	reasons := make([]string, 0, 3)
	if !status.Running {
		reasons = append(reasons, "not running")
	}
	if status.Healthy != nil && !*status.Healthy {
		reasons = append(reasons, "unhealthy")
	}
	if strings.TrimSpace(status.StatusCode) != "" && status.StatusCode != resourcecontrol.StatusCodeOK {
		reasons = append(reasons, "status_code="+status.StatusCode)
	}
	if len(reasons) == 0 {
		return "not ready"
	}
	return strings.Join(reasons, "; ")
}

func (r *Runner) ensureDependencies(item scenario.Scenario, opts StartOptions, ready map[string]struct{}, setupCache setupCheckCache, stack []string) ([]string, error) {
	if len(item.Manifest.Dependencies.Scenarios) == 0 {
		return nil, nil
	}

	failed := []string{}
	names := make([]string, 0, len(item.Manifest.Dependencies.Scenarios))
	for name := range item.Manifest.Dependencies.Scenarios {
		names = append(names, name)
	}
	sort.Strings(names)

	for index, dependencyName := range names {
		dependency := item.Manifest.Dependencies.Scenarios[dependencyName]
		decision := resolveDependencyDecision(dependency, opts.BestEffort)
		if decision.skip {
			r.logDebug("Skipping ignored dependency", logx.AttrScenario, item.Slug, logx.AttrDependency, dependencyName)
			continue
		}

		if _, ok := ready[dependencyName]; ok {
			r.logDebug("Dependency already ready", logx.AttrScenario, item.Slug, logx.AttrDependency, dependencyName)
			continue
		}
		if containsString(stack, dependencyName) {
			return nil, fmt.Errorf("circular scenario dependency detected: %s -> %s", strings.Join(stack, " -> "), dependencyName)
		}

		dependencyItem, err := r.loadScenario(dependencyName, "")
		if err != nil {
			if decision.continueOnFailure {
				r.logWarn("Dependency could not be loaded; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, dependencyName,
					logx.AttrOperation, "load_dependency",
				)
				failed = append(failed, dependencyName)
				continue
			}
			return nil, err
		}

		dependencyView, err := r.lookupRegistryRuntime(context.Background(), dependencyItem)
		if err != nil {
			if decision.continueOnFailure {
				r.logWarn("Dependency registry lookup failed; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, dependencyName,
					logx.AttrOperation, "lookup_dependency_runtime",
					"error", err.Error(),
				)
				failed = append(failed, dependencyName)
				continue
			}
			return nil, err
		}
		dependencyForceSetup := forceSetupFor(opts, dependencyItem.Slug)
		// Reuse is gated on FRESHNESS only: a running, healthy dependency must
		// not be bounced because a provisioning check (e.g. an empty data/
		// dir) reports "not populated". Provisioning is handled if the dep is
		// actually (re)started, never as a reason to restart a healthy one.
		freshnessStale, freshnessReasons, err := r.freshnessStaleCached(dependencyItem, dependencyForceSetup, setupCache)
		if err != nil {
			if decision.continueOnFailure {
				r.logWarn("Dependency setup check failed; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, dependencyName,
					logx.AttrOperation, "setup_needed_dependency",
					"error", err.Error(),
				)
				failed = append(failed, dependencyName)
				continue
			}
			return nil, err
		}
		strictHealthy := r.isRegistryRuntimeHealthy(dependencyItem, dependencyView)
		dependencyRunning := dependencyView.Authoritative
		if dependencyRunning && strictHealthy && !freshnessStale {
			r.publish(ProgressEvent{Kind: EventDependencyReused, Scenario: item.Slug, Dependency: dependencyName, Index: index + 1, Total: len(names)})
			r.logDebug("Dependency already running and healthy", logx.AttrScenario, item.Slug, logx.AttrDependency, dependencyName)
			ready[dependencyName] = struct{}{}
			continue
		}

		// Running + healthy + stale: the freshness_policy decides how (or
		// whether) to disrupt it. Not-running or unhealthy deps always fall
		// through to a full (re)start regardless of policy.
		if dependencyRunning && strictHealthy && freshnessStale {
			handled, err := r.applyDependencyFreshnessPolicy(item.Slug, dependencyItem, decision, dependencyView, freshnessReasons)
			if err != nil {
				if decision.continueOnFailure {
					r.logWarn("Dependency freshness handling failed; continuing in best-effort mode",
						logx.AttrScenario, item.Slug,
						logx.AttrDependency, dependencyName,
						logx.AttrOperation, "apply_freshness_policy",
						"error", err.Error(),
					)
					failed = append(failed, dependencyName)
					continue
				}
				return nil, err
			}
			if handled {
				ready[dependencyName] = struct{}{}
				continue
			}
		}

		reason := dependencyRestartReason(dependencyRunning, strictHealthy, freshnessStale, freshnessReasons)
		r.publish(ProgressEvent{Kind: EventDependencyStarting, Scenario: item.Slug, Dependency: dependencyName, Reason: reason, Index: index + 1, Total: len(names)})
		r.logInfo("Dependency start required",
			logx.AttrScenario, item.Slug,
			logx.AttrDependency, dependencyName,
			"reason", reason,
			"registry_running", dependencyRunning,
			"healthy", strictHealthy,
			"freshness_stale", freshnessStale,
			"freshness_reasons", freshnessReasons,
		)

		dependencyOpts := opts
		dependencyOpts.CustomPath = ""
		dependencyOpts.CleanStale = false
		// Dependencies are shared live infrastructure: a shadow scenario adopts
		// the live dependency instances rather than spinning variant-scoped
		// copies, so the parent's variant must never leak onto a dependency
		// start. dependencyItem is loaded as live above; clear the carried
		// variant too so the "deps are always live" invariant is explicit.
		dependencyOpts.Variant = ""

		dependencyReadyForReuse := func() (bool, error) {
			view, err := r.lookupRegistryRuntime(context.Background(), dependencyItem)
			if err != nil {
				return false, err
			}
			stale, _, err := r.freshnessStaleCached(dependencyItem, dependencyForceSetup, make(setupCheckCache))
			if err != nil {
				return false, err
			}
			return view.Authoritative && r.isRegistryRuntimeHealthy(dependencyItem, view) && !stale, nil
		}

		// Acquire the per-scenario single-flight lock for this transitive
		// dependency. Without it, two top-level scenario starts that
		// share a dependency (e.g. our swarm-manager start and autoheal's
		// app-monitor restart both needing workspace-sandbox) would race
		// on the dependency's port claims — one finishing its acquire +
		// release cycle while the other was mid-startup, surfacing as
		// "claim is no longer reservable" at bind time. The lock is
		// scoped to this dep call and released as soon as the dep is
		// running, before its siblings are started, so DAG fan-out is
		// not held under a single lock.
		depRelease, reusedAfterWait, lockErr := r.acquireDependencyScenarioLock(dependencyName, dependencyReadyForReuse)
		if lockErr != nil {
			if decision.continueOnFailure {
				r.logWarn("Dependency lock contention in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, dependencyName,
					logx.AttrOperation, "start_dependency",
					"error", lockErr.Error(),
				)
				failed = append(failed, dependencyName)
				continue
			}
			return nil, lockErr
		}
		if reusedAfterWait {
			r.publish(ProgressEvent{Kind: EventDependencyReused, Scenario: item.Slug, Dependency: dependencyName, AfterLockWait: true, Index: index + 1, Total: len(names)})
			ready[dependencyName] = struct{}{}
			continue
		}

		_, depErr := r.startScenario(dependencyItem, dependencyOpts, ready, setupCache, append(stack, dependencyName))
		depRelease()
		if err := depErr; err != nil {
			if decision.continueOnFailure {
				r.logWarn("Dependency failed to start; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, dependencyName,
					logx.AttrOperation, "start_dependency",
				)
				failed = append(failed, dependencyName)
				continue
			}
			return nil, err
		}
		ready[dependencyName] = struct{}{}
	}

	return failed, nil
}

func (r *Runner) acquireDependencyScenarioLock(dependencyName string, dependencyReady func() (bool, error)) (func(), bool, error) {
	release, err := r.acquireScenarioLock(dependencyName)
	if err == nil {
		return release, false, nil
	}
	if !errors.Is(err, ErrScenarioBusy) {
		return nil, false, err
	}

	lastErr := err
	r.logInfo("Dependency lifecycle lock is busy; waiting for concurrent invocation to finish",
		logx.AttrDependency, dependencyName,
		"wait_timeout", dependencyLockPolicy.Timeout.String(),
		"error", err.Error(),
	)

	var acquired func()
	reused := false
	// The caller just failed an acquire; re-attempts are spaced one interval
	// apart, so the first tick only checks readiness and the first re-acquire
	// happens after the first sleep.
	firstTick := true
	awaitErr := Await(r.awaitClock(), dependencyLockPolicy, func() (bool, error) {
		if dependencyReady != nil {
			ready, readyErr := dependencyReady()
			if readyErr == nil && ready {
				r.logInfo("Dependency became ready while lifecycle lock was held",
					logx.AttrDependency, dependencyName,
				)
				reused = true
				return true, nil
			}
			if readyErr != nil {
				r.logDebug("Dependency readiness check failed while waiting for lifecycle lock",
					logx.AttrDependency, dependencyName,
					"error", readyErr.Error(),
				)
			}
		}
		if firstTick {
			firstTick = false
			return false, nil
		}
		release, acquireErr := r.acquireScenarioLock(dependencyName)
		if acquireErr == nil {
			acquired = release
			return true, nil
		}
		if !errors.Is(acquireErr, ErrScenarioBusy) {
			return false, acquireErr
		}
		lastErr = acquireErr
		return false, nil
	})
	if awaitErr == nil {
		if reused {
			return nil, true, nil
		}
		return acquired, false, nil
	}
	if errors.Is(awaitErr, ErrAwaitExpired) {
		return nil, false, fmt.Errorf("dependency %q lifecycle lock remained busy for %s: %w", dependencyName, dependencyLockPolicy.Timeout, lastErr)
	}
	return nil, false, awaitErr
}

// applyDependencyFreshnessPolicy handles a running, healthy, but stale
// dependency according to its freshness_policy. It returns handled=true when the
// running process was kept (reuse_running, or rebuild_only after rebuilding the
// artifact in place); handled=false means the caller should proceed to a full
// restart (restart_when_stale). Arbitration (option 1, reduce-only) degrades a
// restart_when_stale edge to rebuild_only when other live scenarios depend on
// the same instance, so this start never bounces a process others rely on.
func (r *Runner) applyDependencyFreshnessPolicy(startingScenario string, dep scenario.Scenario, decision dependencyDecision, view registryRuntimeView, freshnessReasons []string) (bool, error) {
	policy := decision.freshnessPolicy
	if strings.TrimSpace(policy) == "" {
		policy = scenario.DependencyFreshnessPolicyRestartWhenStale
	}
	reasonStr := strings.Join(freshnessReasons, "; ")

	if policy == scenario.DependencyFreshnessPolicyRestartWhenStale &&
		r.dependencyHasOtherLiveConsumers(context.Background(), dep.Slug, startingScenario) {
		r.logInfo("Freshness arbitration: degrading restart to rebuild-only (shared dependency has other live consumers)",
			logx.AttrScenario, dep.Slug, "freshness_reason", reasonStr)
		policy = scenario.DependencyFreshnessPolicyRebuildOnly
	}

	switch policy {
	case scenario.DependencyFreshnessPolicyReuseRunning:
		r.publish(ProgressEvent{Kind: EventDependencyStalePolicy, Scenario: startingScenario, Dependency: dep.Slug, Policy: scenario.DependencyFreshnessPolicyReuseRunning, Reason: reasonStr})
		r.logWarn("Reusing stale dependency per freshness_policy",
			logx.AttrScenario, dep.Slug, "freshness_policy", scenario.DependencyFreshnessPolicyReuseRunning, "freshness_reason", reasonStr)
		return true, nil
	case scenario.DependencyFreshnessPolicyRebuildOnly:
		r.publish(ProgressEvent{Kind: EventDependencyStalePolicy, Scenario: startingScenario, Dependency: dep.Slug, Policy: scenario.DependencyFreshnessPolicyRebuildOnly, Reason: reasonStr})
		r.logInfo("Rebuilding stale dependency without restart",
			logx.AttrScenario, dep.Slug, "freshness_policy", scenario.DependencyFreshnessPolicyRebuildOnly, "freshness_reason", reasonStr)
		if err := r.rebuildDependencyArtifacts(dep, view); err != nil {
			return false, err
		}
		return true, nil
	default: // restart_when_stale
		return false, nil
	}
}

// rebuildDependencyArtifacts re-runs the dependency's setup phase (rebuilding
// its artifacts) without stopping the running process, then re-stamps the
// freshness manifest. The setup env reuses the running instance's bound ports so
// build steps that reference port env vars still resolve.
func (r *Runner) rebuildDependencyArtifacts(item scenario.Scenario, view registryRuntimeView) error {
	env := envFromRuntimeView(item.Manifest, view)
	if _, err := r.runWithLifecycleLog(startLifecycleLogContext(item.Slug, "rebuild", "setup"), func(logWriter, childWriter io.Writer) error {
		_, execErr := r.ExecutePhaseDetailed(item, "setup", env, nil, logWriter, childWriter)
		return execErr
	}); err != nil {
		return err
	}
	r.stampScenarioFreshness(item)
	return nil
}

func envFromRuntimeView(manifest scenario.ServiceManifest, view registryRuntimeView) map[string]string {
	env := map[string]string{}
	for name, port := range view.Ports {
		if p, ok := manifest.Ports[name]; ok && strings.TrimSpace(p.EnvVar) != "" {
			env[p.EnvVar] = strconv.Itoa(port)
		}
	}
	return env
}

// dependencyHasOtherLiveConsumers reports whether any active scenario other than
// the one currently starting (and other than the dependency itself) declares a
// non-ignored dependency on depSlug. Used by reduce-only arbitration. On any
// enumeration failure it returns true — the safe choice is to never bounce a
// shared dependency when we cannot prove it is unshared.
func (r *Runner) dependencyHasOtherLiveConsumers(ctx context.Context, depSlug, startingScenario string) bool {
	deps := r.runtimeDeps()
	store, err := deps.runtimeRegistry(ctx, r.Home)
	if err != nil {
		return true
	}
	defer store.Close()

	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return true
	}

	checked := map[string]bool{}
	for _, inst := range instances {
		if inst.Scenario == depSlug || inst.Scenario == startingScenario || checked[inst.Scenario] {
			continue
		}
		checked[inst.Scenario] = true
		consumer, err := r.loadScenario(inst.Scenario, "")
		if err != nil {
			continue
		}
		dep, ok := consumer.Manifest.Dependencies.Scenarios[depSlug]
		if ok && dep.NormalizedStartupPolicy() != scenario.DependencyStartupPolicyIgnore {
			return true
		}
	}
	return false
}

func resolveDependencyDecision(dependency scenario.Dependency, bestEffort bool) dependencyDecision {
	policy := dependency.NormalizedStartupPolicy()
	return dependencyDecision{
		policy:            policy,
		freshnessPolicy:   dependency.NormalizedFreshnessPolicy(),
		skip:              policy == scenario.DependencyStartupPolicyIgnore,
		continueOnFailure: bestEffort || policy == scenario.DependencyStartupPolicyTryStart,
	}
}

func (r *Runner) ensureResourceDependencies(item scenario.Scenario, opts StartOptions) ([]string, error) {
	deps := r.runtimeDeps()
	if len(item.Manifest.Dependencies.Resources) == 0 {
		return nil, nil
	}

	failed := []string{}
	names := make([]string, 0, len(item.Manifest.Dependencies.Resources))
	for name := range item.Manifest.Dependencies.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, resourceName := range names {
		dependency := item.Manifest.Dependencies.Resources[resourceName]
		decision := resolveDependencyDecision(dependency, opts.BestEffort)
		if decision.skip {
			r.logDebug("Skipping ignored resource dependency", logx.AttrScenario, item.Slug, logx.AttrDependency, resourceName)
			continue
		}

		status, err := deps.resourceStatus(resourceName, false)
		if err != nil {
			if decision.continueOnFailure {
				r.logWarn("Resource dependency status failed; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, resourceName,
					logx.AttrOperation, "status_resource_dependency",
				)
				failed = append(failed, resourceName)
				continue
			}
			return nil, fmt.Errorf("status resource dependency %s: %w", resourceName, err)
		}
		if resourceDependencyReady(status) {
			r.publish(ProgressEvent{Kind: EventResourceReused, Scenario: item.Slug, Dependency: resourceName})
			r.logDebug("Resource dependency already running and healthy", logx.AttrScenario, item.Slug, logx.AttrDependency, resourceName)
			if err := r.ensureResourceConfig(item.Slug, resourceName, dependency.Config, decision); err != nil {
				if decision.continueOnFailure {
					failed = append(failed, resourceName)
					continue
				}
				return nil, err
			}
			continue
		}

		reason := resourceDependencyStartReason(status)
		r.publish(ProgressEvent{Kind: EventResourceStarting, Scenario: item.Slug, Dependency: resourceName, Reason: reason})
		r.logInfo("Resource dependency start required",
			logx.AttrScenario, item.Slug,
			logx.AttrDependency, resourceName,
			"reason", reason,
			"running", status.Running,
			"healthy", status.Healthy,
			"status_code", status.StatusCode,
		)

		if err := r.enforceResourceHostRequirements(resourceName); err != nil {
			if decision.continueOnFailure {
				r.logWarn("Resource dependency host requirements failed; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, resourceName,
					logx.AttrOperation, "enforce_resource_host_requirements",
				)
				failed = append(failed, resourceName)
				continue
			}
			return nil, fmt.Errorf("enforce host requirements for resource dependency %s: %w", resourceName, err)
		}
		if err := deps.runResource(resourceName, []string{"start"}, r.consoleOut(), r.consoleErr()); err != nil {
			if decision.continueOnFailure {
				r.logWarn("Resource dependency failed to start; continuing in best-effort mode",
					logx.AttrScenario, item.Slug,
					logx.AttrDependency, resourceName,
					logx.AttrOperation, "start_resource_dependency",
				)
				failed = append(failed, resourceName)
				continue
			}
			return nil, fmt.Errorf("start resource dependency %s: %w", resourceName, err)
		}

		_, err = r.waitForResourceDependencyReady(resourceName)
		if err == nil {
			if ensureErr := r.ensureResourceConfig(item.Slug, resourceName, dependency.Config, decision); ensureErr != nil {
				if decision.continueOnFailure {
					failed = append(failed, resourceName)
					continue
				}
				return nil, ensureErr
			}
			continue
		}
		if decision.continueOnFailure {
			r.logWarn("Resource dependency remained unavailable after start attempt; continuing in best-effort mode",
				logx.AttrScenario, item.Slug,
				logx.AttrDependency, resourceName,
				logx.AttrOperation, "verify_started_resource_dependency",
			)
			failed = append(failed, resourceName)
			continue
		}
		return nil, err
	}

	return failed, nil
}

// ensureResourceConfig calls the resource CLI's `ensure` verb with the raw
// dependency config blob, if the dependency declared extra keys AND the
// resource's manifest advertises `supports_ensure`. Errors are wrapped with
// context and returned to the caller, which applies the dependency's
// continueOnFailure policy.
func (r *Runner) ensureResourceConfig(scenarioSlug, resourceName string, cfg json.RawMessage, decision dependencyDecision) error {
	if len(cfg) == 0 {
		return nil
	}
	deps := r.runtimeDeps()
	manifest, err := deps.resourceManifest(resourceName)
	if err != nil {
		if decision.continueOnFailure {
			r.logWarn("Resource manifest load failed; skipping ensure",
				logx.AttrScenario, scenarioSlug,
				logx.AttrDependency, resourceName,
				logx.AttrOperation, "load_resource_manifest",
			)
			return nil
		}
		return fmt.Errorf("load resource manifest %s: %w", resourceName, err)
	}
	if !manifest.Capabilities.SupportsEnsure {
		r.logDebug("Resource does not advertise supports_ensure; skipping ensure",
			logx.AttrScenario, scenarioSlug, logx.AttrDependency, resourceName)
		return nil
	}

	encoded := base64.StdEncoding.EncodeToString(cfg)
	args := []string{"ensure", "--config-base64", encoded}
	r.publish(ProgressEvent{Kind: EventResourceEnsureConfig, Scenario: scenarioSlug, Dependency: resourceName})
	r.logInfo("Running resource ensure",
		logx.AttrScenario, scenarioSlug,
		logx.AttrDependency, resourceName,
		logx.AttrOperation, "ensure_resource_config",
		"config_bytes", len(cfg),
	)
	if err := deps.runResourceCLI(resourceName, args, r.consoleOut(), r.consoleErr()); err != nil {
		r.logWarn("Resource ensure failed",
			logx.AttrScenario, scenarioSlug,
			logx.AttrDependency, resourceName,
			logx.AttrOperation, "ensure_resource_config",
			"error", err.Error(),
		)
		return fmt.Errorf("ensure resource dependency %s: %w", resourceName, err)
	}
	return nil
}

func (r *Runner) waitForResourceDependencyReady(resourceName string) (resourcecontrol.Status, error) {
	deps := r.runtimeDeps()

	var lastStatus resourcecontrol.Status
	var lastErr error
	err := Await(r.awaitClock(), resourceReadyPolicy, func() (bool, error) {
		status, statusErr := deps.resourceStatus(resourceName, false)
		if statusErr != nil {
			// Transient probe failures are retried through the policy bound;
			// only the last one is surfaced at expiry.
			lastErr = statusErr
			return false, nil
		}
		lastStatus = status
		return resourceDependencyReady(status), nil
	})
	if err == nil {
		return lastStatus, nil
	}
	if lastErr != nil {
		return lastStatus, fmt.Errorf("status started resource dependency %s: %w", resourceName, lastErr)
	}
	return lastStatus, fmt.Errorf(
		"resource dependency %s is not ready after start (running=%t health=%q status_code=%q)",
		resourceName,
		lastStatus.Running,
		lastStatus.Health,
		lastStatus.StatusCode,
	)
}

func resourceDependencyReady(status resourcecontrol.Status) bool {
	if !status.Running {
		return false
	}
	if status.Healthy != nil {
		return *status.Healthy
	}
	return status.StatusCode == "" || status.StatusCode == resourcecontrol.StatusCodeOK
}
