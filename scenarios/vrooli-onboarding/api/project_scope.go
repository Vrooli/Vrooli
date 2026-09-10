package main

import (
	"strings"
)

// projectScopeAvailable reports whether the resolved roots describe a
// repository install rather than a desktop bundle.
//
// The distinction is deliberate and is not an accident of a missing file. A
// desktop bundle stages only scenario, resource, tool, and safeguard manifests;
// it must not inherit Vrooli's development-machine setup requirements, and
// internal/hostreq states the same intent through ResolveOptions.ExcludeRoot.
func projectScopeAvailable() bool {
	roots, err := resolveRoots()
	return err == nil && strings.TrimSpace(roots.RepoRoot) != ""
}

// hostRequirementScenarioModels returns the read models for every scenario the
// apply run will actually start.
//
// The host consent step and the apply plan must be derived from the same set,
// or the operator consents to one list of host changes and setup performs a
// different one. The closure is that set: it is the operator's selection plus
// the scenario dependencies those selections pull in, which is exactly what
// apply starts. Passing every discovered read model instead would include host
// tools and safeguards from scenarios the operator never selected.
func hostRequirementScenarioModels(root string, models []ScenarioReadModel, state OperatorState) ([]ScenarioReadModel, error) {
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]ScenarioReadModel, len(models))
	for _, model := range models {
		byName[model.Name] = model
	}
	selected := make([]ScenarioReadModel, 0, len(closure.Scenarios))
	for _, member := range closure.Scenarios {
		if model, found := byName[member.Name]; found {
			selected = append(selected, model)
		}
	}
	return selected, nil
}
