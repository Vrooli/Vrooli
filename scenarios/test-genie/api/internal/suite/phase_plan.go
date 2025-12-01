package suite

import (
	"fmt"
)

type phasePlan struct {
	Definitions []phaseDefinition
	Selected    []phaseDefinition
	PresetUsed  string
}

func (o *SuiteOrchestrator) buildPhasePlan(env PhaseEnvironment, cfg *testingConfig, req SuiteExecutionRequest) (*phasePlan, error) {
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
		return nil, NewValidationError("no phases selected for execution")
	}

	return &phasePlan{
		Definitions: defs,
		Selected:    selected,
		PresetUsed:  presetUsed,
	}, nil
}
