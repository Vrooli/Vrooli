package lifecycle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

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
		required := dependency.Required
		// Legacy array-based manifests did not serialize explicit type/required fields
		// for scenario dependencies; preserve the historical "required by default" behavior.
		if !required && dependency.Type == "" && dependency.StartupPolicy == "" {
			required = true
		}
		startupPolicy := dependency.StartupPolicy
		if startupPolicy == "" {
			if required {
				startupPolicy = "must_start"
			} else {
				startupPolicy = "ignore"
			}
		}
		if startupPolicy == "ignore" {
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
			if opts.BestEffort || startupPolicy == "try_start" {
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

		dependencyRecords, err := readScenarioRecords(r.Home, dependencyName)
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
			if opts.BestEffort || startupPolicy == "try_start" {
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
