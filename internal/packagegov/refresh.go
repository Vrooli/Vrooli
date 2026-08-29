package packagegov

import "slices"

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
		} else if !containsConsumerClass(action.ConsumerClasses, dep.ConsumerClass) {
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

func containsConsumerClass(items []ConsumerClass, candidate ConsumerClass) bool {
	for _, item := range items {
		if item == candidate {
			return true
		}
	}
	return false
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
