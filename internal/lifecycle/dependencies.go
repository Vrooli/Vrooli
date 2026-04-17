package lifecycle

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/process"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	"github.com/vrooli/vrooli/internal/scenario"
)

type dependencyDecision struct {
	policy            string
	skip              bool
	continueOnFailure bool
}

const (
	resourceDependencyReadyTimeout  = 30 * time.Second
	resourceDependencyReadyInterval = 500 * time.Millisecond
)

func (r *Runner) ensureDependencies(item scenario.Scenario, opts StartOptions, ready map[string]struct{}, stack []string) ([]string, error) {
	deps := r.runtimeDeps()
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

		dependencyRecords, err := deps.readScenarioRecords(r.Home, dependencyName)
		if err != nil {
			return nil, err
		}
		dependencyRuntime := process.SummarizeScenario(dependencyName, dependencyRecords)
		dependencyForceSetup := opts.ForceSetup && opts.ForceSetupScenario == dependencyName
		setupNeeded, _, err := r.SetupNeeded(dependencyItem, dependencyForceSetup)
		if err != nil {
			return nil, err
		}
		if dependencyRuntime.ProcessCount > 0 && r.isScenarioHealthyStrict(dependencyItem, dependencyRuntime.Records) && !setupNeeded {
			r.logDebug("Dependency already running and healthy", logx.AttrScenario, item.Slug, logx.AttrDependency, dependencyName)
			ready[dependencyName] = struct{}{}
			continue
		}

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
			r.logDebug("Resource dependency already running and healthy", logx.AttrScenario, item.Slug, logx.AttrDependency, resourceName)
			continue
		}

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
