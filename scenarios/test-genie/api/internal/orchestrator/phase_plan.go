package orchestrator

import (
	"fmt"

	"test-genie/internal/orchestrator/phases"
	workspacepkg "test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

type phasePlan struct {
	Definitions       []phases.Definition
	Selected          []phases.Definition
	PresetUsed        string
	GlobalToggles     PhaseToggleConfig
	DisabledByDefault []phaseDisableNotice
	ExplicitDisabled  []phaseDisableNotice
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

	globalToggles, err := o.GlobalPhaseToggles()
	if err != nil {
		return nil, fmt.Errorf("load global phase toggles: %w", err)
	}

	available := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		available[def.Name.Key()] = struct{}{}
	}

	presets := o.loadPresets(env.TestDir, cfg, available)
	selected, presetUsed, notices, err := selectPhases(defs, presets, req, globalToggles)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		if len(notices.Skipped) > 0 {
			return nil, shared.NewValidationError("no phases selected for execution; requested phases are globally disabled")
		}
		return nil, shared.NewValidationError("no phases selected for execution")
	}

	return &phasePlan{
		Definitions:       defs,
		Selected:          selected,
		PresetUsed:        presetUsed,
		GlobalToggles:     globalToggles,
		DisabledByDefault: notices.Skipped,
		ExplicitDisabled:  notices.Explicit,
	}, nil
}

type phaseDisableNotice struct {
	Name   string
	Toggle PhaseToggle
}
