package orchestrator

import (
	"strings"

	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/shared"

	workspacepkg "test-genie/internal/orchestrator/workspace"
)

// PlannedPhase describes a selected phase before execution starts.
type PlannedPhase struct {
	Name                 string                 `json:"name"`
	DisplayName          string                 `json:"displayName,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Optional             bool                   `json:"optional"`
	TimeoutSeconds       int                    `json:"timeoutSeconds"`
	SelectionStatus      string                 `json:"selectionStatus,omitempty"`
	ApplicabilityStatus  applicability.Status   `json:"applicabilityStatus,omitempty"`
	ApplicabilityReasons []applicability.Reason `json:"applicabilityReasons,omitempty"`
	ProviderReadiness    string                 `json:"providerReadiness,omitempty"`
	Freshness            string                 `json:"freshness,omitempty"`
	Policy               phasepolicy.Policy     `json:"policy,omitempty"`
	DescriptorPath       string                 `json:"descriptorPath,omitempty"`
}

// ExecutionPlanPreview captures the actual selected phase plan for a request.
type ExecutionPlanPreview struct {
	ScenarioName        string         `json:"scenarioName"`
	PresetUsed          string         `json:"presetUsed,omitempty"`
	Phases              []PlannedPhase `json:"phases"`
	NotApplicablePhases []PlannedPhase `json:"notApplicablePhases,omitempty"`
	Warnings            []string       `json:"warnings,omitempty"`
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

	ws.SetRuntimeURLs(req.UIURL, req.APIURL)
	ws.SetClaims(o.claims)

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
		preview.Phases = append(preview.Phases, o.plannedPhasePreview(def, ctx.plan, "selected"))
	}
	for _, notice := range ctx.plan.NotApplicable {
		preview.NotApplicablePhases = append(preview.NotApplicablePhases, o.notApplicablePhasePreview(notice))
	}

	return preview, nil
}

func (o *SuiteOrchestrator) plannedPhasePreview(def phases.Definition, plan *phasePlan, selectionStatus string) PlannedPhase {
	timeout := def.Timeout
	if timeout <= 0 {
		timeout = o.phaseTimeout
	}
	phasePreview := PlannedPhase{
		Name:              def.Name.String(),
		DisplayName:       def.DisplayName,
		Optional:          def.Optional,
		TimeoutSeconds:    int(timeout.Seconds()),
		SelectionStatus:   selectionStatus,
		Policy:            def.Policy,
		ProviderReadiness: string(def.Policy.ProviderReadiness),
		Freshness:         string(def.Policy.Freshness),
	}
	if notice, ok := planApplicabilityNotice(plan, def.Name.Key()); ok {
		phasePreview.ApplicabilityStatus = notice.Result.Status
		phasePreview.ApplicabilityReasons = append([]applicability.Reason(nil), notice.Result.Reasons...)
		phasePreview.DescriptorPath = notice.Descriptor.Path
	}
	if o.catalog != nil {
		if spec, ok := o.catalog.Lookup(def.Name.String()); ok {
			if phasePreview.DisplayName == "" {
				phasePreview.DisplayName = spec.DisplayName
			}
			phasePreview.Description = spec.Description
		}
	}
	if phasePreview.Description == "" {
		if entry, ok := o.descriptorEntry(def.Name.String()); ok {
			phasePreview.Description = entry.Spec.Description
		}
	}
	return phasePreview
}

func (o *SuiteOrchestrator) notApplicablePhasePreview(notice phaseApplicabilityNotice) PlannedPhase {
	preview := o.plannedPhasePreview(notice.Definition, &phasePlan{NotApplicable: []phaseApplicabilityNotice{notice}}, "not_applicable")
	preview.ApplicabilityStatus = notice.Result.Status
	preview.ApplicabilityReasons = append([]applicability.Reason(nil), notice.Result.Reasons...)
	preview.DescriptorPath = notice.Descriptor.Path
	return preview
}

func planApplicabilityNotice(plan *phasePlan, key string) (phaseApplicabilityNotice, bool) {
	if plan == nil {
		return phaseApplicabilityNotice{}, false
	}
	if notice, ok := plan.Applicability[key]; ok {
		return notice, true
	}
	for _, notice := range plan.NotApplicable {
		if notice.Definition.Name.Key() == key {
			return notice, true
		}
	}
	return phaseApplicabilityNotice{}, false
}
