package orchestrator

import (
	"strings"

	"test-genie/internal/shared"

	workspacepkg "test-genie/internal/orchestrator/workspace"
)

// PlannedPhase describes a selected phase before execution starts.
type PlannedPhase struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Optional       bool   `json:"optional"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

// ExecutionPlanPreview captures the actual selected phase plan for a request.
type ExecutionPlanPreview struct {
	ScenarioName string         `json:"scenarioName"`
	PresetUsed   string         `json:"presetUsed,omitempty"`
	Phases       []PlannedPhase `json:"phases"`
	Warnings     []string       `json:"warnings,omitempty"`
}

type executionPlanContext struct {
	env    workspacepkg.Environment
	config *workspacepkg.Config
	plan   *phasePlan
}

func (o *SuiteOrchestrator) loadExecutionPlanContext(req SuiteExecutionRequest) (*executionPlanContext, error) {
	scenario := strings.TrimSpace(req.ScenarioName)
	if scenario == "" {
		return nil, shared.NewValidationError("scenarioName is required")
	}

	ws, err := workspacepkg.NewWithOptions(o.scenariosRoot, scenario, workspacepkg.Options{
		ScenarioPath:           req.ScenarioPath,
		LogicalRepoRoot:        req.LogicalRepoRoot,
		LogicalScenarioRelPath: req.LogicalScenarioRelPath,
	})
	if err != nil {
		return nil, err
	}

	ws.SetRuntimeURLs(req.UIURL, req.APIURL, resolveBrowserlessURL(req.BrowserlessURL))

	env := ws.Environment()
	config, err := workspacepkg.LoadTestingConfig(env.ScenarioDir)
	if err != nil {
		return nil, err
	}

	plan, err := o.buildPhasePlan(env, config, req)
	if err != nil {
		return nil, err
	}

	return &executionPlanContext{
		env:    env,
		config: config,
		plan:   plan,
	}, nil
}

// PreviewExecution resolves the request into the actual phase plan that would run.
func (o *SuiteOrchestrator) PreviewExecution(req SuiteExecutionRequest) (*ExecutionPlanPreview, error) {
	ctx, err := o.loadExecutionPlanContext(req)
	if err != nil {
		return nil, err
	}

	preview := &ExecutionPlanPreview{
		ScenarioName: ctx.env.ScenarioName,
		PresetUsed:   ctx.plan.PresetUsed,
		Warnings:     buildPlanWarnings(ctx.plan),
		Phases:       make([]PlannedPhase, 0, len(ctx.plan.Selected)),
	}

	for _, def := range ctx.plan.Selected {
		timeout := def.Timeout
		if timeout <= 0 {
			timeout = o.phaseTimeout
		}

		phasePreview := PlannedPhase{
			Name:           def.Name.String(),
			Optional:       def.Optional,
			TimeoutSeconds: int(timeout.Seconds()),
		}

		if o.catalog != nil {
			if spec, ok := o.catalog.Lookup(def.Name.String()); ok {
				phasePreview.Description = spec.Description
			}
		}

		preview.Phases = append(preview.Phases, phasePreview)
	}

	return preview, nil
}
