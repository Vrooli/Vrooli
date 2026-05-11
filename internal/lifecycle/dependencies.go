package lifecycle

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	"github.com/vrooli/vrooli/internal/scenario"
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type dependencyDecision struct {
	policy            string
	skip              bool
	continueOnFailure bool
}

const (
	resourceDependencyReadyTimeout  = 30 * time.Second
	resourceDependencyReadyInterval = 500 * time.Millisecond
)

func dependencyRestartReason(processCount int, healthy bool, setupNeeded bool, setupReasons []string) string {
	reasons := make([]string, 0, 3)
	switch {
	case processCount == 0:
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
	if len(reasons) == 0 {
		return "state changed"
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

func (r *Runner) ensureDependencies(item scenario.Scenario, opts StartOptions, ready map[string]struct{}, stack []string) ([]string, error) {
	if len(item.Manifest.Dependencies.Scenarios) == 0 {
		return nil, nil
	}

	failed := []string{}
	names := make([]string, 0, len(item.Manifest.Dependencies.Scenarios))
	for name := range item.Manifest.Dependencies.Scenarios {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, dependencyName := range names {
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
			return nil, err
		}
		dependencyForceSetup := opts.ForceSetup && opts.ForceSetupScenario == dependencyName
		setupNeeded, setupReasons, err := r.SetupNeeded(dependencyItem, dependencyForceSetup)
		if err != nil {
			return nil, err
		}
		strictHealthy := r.isRegistryRuntimeHealthy(dependencyItem, dependencyView)
		dependencyRunning := dependencyView.Authoritative
		if dependencyRunning && strictHealthy && !setupNeeded {
			r.progressf("%s: dependency %s already running; reusing existing process", item.Slug, dependencyName)
			r.logDebug("Dependency already running and healthy", logx.AttrScenario, item.Slug, logx.AttrDependency, dependencyName)
			ready[dependencyName] = struct{}{}
			continue
		}

		reason := dependencyRestartReason(boolToInt(dependencyRunning), strictHealthy, setupNeeded, setupReasons)
		r.progressf("%s: starting dependency %s (%s)", item.Slug, dependencyName, reason)
		r.logInfo("Dependency start required",
			logx.AttrScenario, item.Slug,
			logx.AttrDependency, dependencyName,
			"reason", reason,
			"registry_running", dependencyRunning,
			"healthy", strictHealthy,
			"setup_needed", setupNeeded,
			"setup_reasons", setupReasons,
		)

		dependencyOpts := opts
		dependencyOpts.CustomPath = ""
		dependencyOpts.CleanStale = false

		if _, err := r.startScenario(dependencyItem, dependencyOpts, ready, append(stack, dependencyName)); err != nil {
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

func resolveDependencyDecision(dependency scenario.Dependency, bestEffort bool) dependencyDecision {
	policy := dependency.NormalizedStartupPolicy()
	return dependencyDecision{
		policy:            policy,
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
			r.progressf("%s: resource dependency %s already running; reusing existing service", item.Slug, resourceName)
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
		r.progressf("%s: starting resource dependency %s (%s)", item.Slug, resourceName, reason)
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
	r.progressf("%s: ensuring %s dependency config", scenarioSlug, resourceName)
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
	deadline := deps.now().Add(resourceDependencyReadyTimeout)

	var lastStatus resourcecontrol.Status
	var lastErr error
	for {
		status, err := deps.resourceStatus(resourceName, false)
		if err == nil {
			lastStatus = status
			if resourceDependencyReady(status) {
				return status, nil
			}
		} else {
			lastErr = err
		}

		if !deps.now().Before(deadline) {
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

		deps.sleep(resourceDependencyReadyInterval)
	}
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
