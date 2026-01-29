package recommendations

import (
	"context"

	"swarm-manager/internal/scenarios"
)

type stubScenarioSource struct {
	scenarios []scenarios.ScenarioSource
	err       error
}

func (s stubScenarioSource) List(_ context.Context) ([]scenarios.ScenarioSource, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.scenarios, nil
}

type stubCompletenessSource struct {
	scores map[string]int
	err    error
}

func (s stubCompletenessSource) Scores(_ context.Context) (map[string]int, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.scores == nil {
		return map[string]int{}, nil
	}
	return s.scores, nil
}

func newTestEngine(root string, sources []scenarios.ScenarioSource) *Engine {
	return NewEngineWithDeps(root, stubScenarioSource{scenarios: sources}, stubCompletenessSource{})
}

func makeScenarioSource(name, description, path string, tags ...string) scenarios.ScenarioSource {
	return scenarios.ScenarioSource{
		Name:        name,
		Description: description,
		Path:        path,
		Status:      "running",
		Tags:        tags,
	}
}
