package orchestrator

import (
	"fmt"

	"test-genie/internal/orchestrator/phases"
	workspacepkg "test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

type phasePlan struct {
	Definitions []phases.Definition
	Selected    []phases.Definition
	PresetUsed  string
}

func (o *SuiteOrchestrator) buildPhasePlan(env workspacepkg.Environment, cfg *workspacepkg.Config, req SuiteExecutionRequest) (*phasePlan, error) {
	defs, err := o.discoverPhaseDefinitions(env)
	if err != nil {
		return nil, err
	}
	defs = o.applyTestingConfig(defs, cfg)
	if len(defs) == 0 {
		return nil, fmt.Errorf("scenario '%s' has no enabled phase definitions", env.ScenarioName)
	}

	available := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		available[def.Name.Key()] = struct{}{}
	}

	presets := o.loadPresets(env.TestDir, cfg, available)
	selected, presetUsed, err := selectPhases(defs, presets, req)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, shared.NewValidationError("no phases selected for execution")
	}

	return &phasePlan{
		Definitions: defs,
		Selected:    selected,
		PresetUsed:  presetUsed,
	}, nil
}
