package packagegov

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type RefreshActionKind string

const (
	RefreshActionScenarioSetup     RefreshActionKind = "scenario_setup"
	RefreshActionRestartScenario   RefreshActionKind = "restart_running_consumer"
	RefreshActionRebuildGoConsumer RefreshActionKind = "rebuild_go_consumer"
	RefreshActionNoRuntimeRefresh  RefreshActionKind = "no_runtime_refresh"
	RefreshActionNoAction          RefreshActionKind = "no_action"
)

type RefreshAction struct {
	ConsumerName    string            `json:"consumer_name"`
	ConsumerClass   ConsumerClass     `json:"consumer_class"`
	ConsumerClasses []ConsumerClass   `json:"consumer_classes,omitempty"`
	ConsumerPath    string            `json:"consumer_path"`
	Action          RefreshActionKind `json:"action"`
	Dependents      []Dependent       `json:"dependents,omitempty"`
	Impact          string            `json:"impact,omitempty"`
	Reason          string            `json:"reason,omitempty"`
}

const (
	ImpactAffected   = "affected"
	ImpactUnaffected = "unaffected"
	ImpactUnknown    = "unknown"
)

type RefreshImpact struct {
	ConsumerName  string        `json:"consumer_name"`
	ConsumerPath  string        `json:"consumer_path"`
	ConsumerClass ConsumerClass `json:"consumer_class"`
	Status        string        `json:"status"`
	Matches       []string      `json:"matches,omitempty"`
	Reason        string        `json:"reason,omitempty"`
}

type ArtifactRefreshPlan struct {
	Actions []RefreshAction `json:"actions"`
	Impacts []RefreshImpact `json:"impacts"`
}

var rclImportPattern = regexp.MustCompile(`@vrooli/react-component-library/([A-Za-z][A-Za-z0-9-]*)(/([0-9]+\.[0-9]+\.[0-9]+|[0-9]+))?`)

// PlanRefreshForArtifact narrows package dependents using the exact exported
// paths changed by one verified artifact. An absent change set or an
// unreadable consumer is deliberately conservative: unknown consumers remain
// refresh targets instead of being silently classified as unaffected.
func PlanRefreshForArtifact(root string, pkg Package, dependents []Dependent, target string, changedExports []string) (ArtifactRefreshPlan, error) {
	targets := MatchDependents(dependents, target)
	grouped := make(map[string][]Dependent)
	for _, dependent := range targets {
		key := dependent.ConsumerName + "\x00" + dependent.ConsumerPath
		grouped[key] = append(grouped[key], dependent)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	plan := ArtifactRefreshPlan{}
	changed := make(map[string]struct{}, len(changedExports))
	for _, export := range changedExports {
		export = strings.TrimPrefix(strings.TrimSpace(export), "./")
		if export != "" {
			changed[export] = struct{}{}
		}
	}
	for _, key := range keys {
		entries := grouped[key]
		first := entries[0]
		impact := RefreshImpact{ConsumerName: first.ConsumerName, ConsumerPath: first.ConsumerPath, ConsumerClass: first.ConsumerClass}
		if len(changed) == 0 {
			impact.Status = ImpactUnknown
			impact.Reason = "artifact did not provide an exact changed-export set"
		} else if first.ConsumerClass != ConsumerScenarioUI && first.ConsumerClass != ConsumerScenarioTest && first.ConsumerClass != ConsumerTemplateUI {
			impact.Status = ImpactUnknown
			impact.Reason = "consumer class has no exact-export source scanner"
		} else {
			matches, scanErr := matchingRCLExports(first.ConsumerPath, changed)
			if scanErr != nil {
				impact.Status = ImpactUnknown
				impact.Reason = "consumer source scan failed: " + scanErr.Error()
			} else if len(matches) > 0 {
				impact.Status = ImpactAffected
				impact.Matches = matches
			} else {
				impact.Status = ImpactUnaffected
			}
		}
		plan.Impacts = append(plan.Impacts, impact)
		if impact.Status == ImpactUnaffected {
			continue
		}
		action := refreshActionForConsumer(pkg.Manifest.Package.Refresh.Strategy, first.ConsumerClass)
		classes := make([]ConsumerClass, 0, len(entries))
		for _, entry := range entries {
			if !slices.Contains(classes, entry.ConsumerClass) {
				classes = append(classes, entry.ConsumerClass)
			}
		}
		plan.Actions = append(plan.Actions, RefreshAction{
			ConsumerName: first.ConsumerName, ConsumerClass: first.ConsumerClass, ConsumerClasses: classes,
			ConsumerPath: first.ConsumerPath, Action: action, Dependents: entries,
			Impact: impact.Status, Reason: impact.Reason,
		})
	}
	_ = root // retained in the signature for future closure-ledger joins.
	return plan, nil
}

func matchingRCLExports(root string, changed map[string]struct{}) ([]string, error) {
	matches := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "node_modules", "dist", "build", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || !strings.Contains(filepath.Base(path), ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".mjs" && ext != ".cjs" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range rclImportPattern.FindAllStringSubmatch(string(data), -1) {
			export := match[1] + match[2]
			if _, ok := changed[export]; ok {
				matches[export] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for match := range matches {
		out = append(out, "./"+match)
	}
	sort.Strings(out)
	return out, nil
}

func PlanRefresh(pkg Package, dependents []Dependent, target string) []RefreshAction {
	targets := MatchDependents(dependents, target)
	grouped := make(map[string]*RefreshAction, len(targets))
	order := make([]string, 0, len(targets))
	for _, dep := range targets {
		key := dep.ConsumerName + "\x00" + dep.ConsumerPath
		action, ok := grouped[key]
		if !ok {
			action = &RefreshAction{
				ConsumerName:    dep.ConsumerName,
				ConsumerClass:   dep.ConsumerClass,
				ConsumerClasses: []ConsumerClass{dep.ConsumerClass},
				ConsumerPath:    dep.ConsumerPath,
				Action:          refreshActionForConsumer(pkg.Manifest.Package.Refresh.Strategy, dep.ConsumerClass),
			}
			grouped[key] = action
			order = append(order, key)
		} else if !slices.Contains(action.ConsumerClasses, dep.ConsumerClass) {
			action.ConsumerClasses = append(action.ConsumerClasses, dep.ConsumerClass)
		}
		action.Dependents = append(action.Dependents, dep)
	}
	slices.Sort(order)

	actions := make([]RefreshAction, 0, len(order))
	for _, key := range order {
		actions = append(actions, *grouped[key])
	}
	return actions
}

func refreshActionForConsumer(strategy RefreshStrategy, class ConsumerClass) RefreshActionKind {
	switch strategy {
	case RefreshScenarioSetup, RefreshGenerateThenSetup:
		switch class {
		case ConsumerScenarioUI, ConsumerScenarioAPI, ConsumerScenarioCLI, ConsumerScenarioTest:
			return RefreshActionScenarioSetup
		default:
			return RefreshActionNoRuntimeRefresh
		}
	case RefreshRestartConsumers:
		switch class {
		case ConsumerScenarioUI, ConsumerScenarioAPI, ConsumerScenarioCLI, ConsumerScenarioTest:
			return RefreshActionRestartScenario
		case ConsumerTemplateAPI, ConsumerTemplateCLI, ConsumerResourceRuntime:
			return RefreshActionRebuildGoConsumer
		default:
			return RefreshActionNoRuntimeRefresh
		}
	case RefreshRebuildCLI:
		switch class {
		case ConsumerScenarioAPI, ConsumerScenarioCLI, ConsumerScenarioTest, ConsumerTemplateAPI, ConsumerTemplateCLI, ConsumerResourceRuntime:
			return RefreshActionRebuildGoConsumer
		default:
			return RefreshActionNoRuntimeRefresh
		}
	case RefreshNone:
		return RefreshActionNoAction
	default:
		return RefreshActionNoAction
	}
}
