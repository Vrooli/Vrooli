package orchestrator

import (
	"strings"

	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/shared"

	workspacepkg "test-genie/internal/orchestrator/workspace"

	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
)

// PlannedPhase describes a selected phase before execution starts.
type PlannedPhase struct {
	Name                 string                 `json:"name"`
	DisplayName          string                 `json:"displayName,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Provider             string                 `json:"provider,omitempty"`
	Source               string                 `json:"source,omitempty"`
	Optional             bool                   `json:"optional"`
	TimeoutSeconds       int                    `json:"timeoutSeconds"`
	SelectionStatus      string                 `json:"selectionStatus,omitempty"`
	ApplicabilityStatus  applicability.Status   `json:"applicabilityStatus,omitempty"`
	ApplicabilityReasons []applicability.Reason `json:"applicabilityReasons,omitempty"`
	ProviderReadiness    string                 `json:"providerReadiness,omitempty"`
	Freshness            string                 `json:"freshness,omitempty"`
	Policy               phasepolicy.Policy     `json:"policy,omitempty"`
	DocPath              string                 `json:"docPath,omitempty"`
	DescriptorPath       string                 `json:"descriptorPath,omitempty"`
	FindingSource        string                 `json:"findingSource,omitempty"`
	ProfileMembership    []string               `json:"profileMembership,omitempty"`
	FreshnessRequirement string                 `json:"freshnessRequirement,omitempty"`
	PhaseClass           string                 `json:"phaseClass,omitempty"`
	RuntimeClass         string                 `json:"runtimeClass,omitempty"`
	Dimensions           []string               `json:"dimensions,omitempty"`
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
		Name:                 def.Name.String(),
		DisplayName:          def.DisplayName,
		Provider:             def.ProviderScenario,
		Optional:             def.Optional,
		TimeoutSeconds:       int(timeout.Seconds()),
		SelectionStatus:      selectionStatus,
		Policy:               def.Policy,
		ProviderReadiness:    string(def.Policy.ProviderReadiness),
		Freshness:            string(def.Policy.Freshness),
		ProfileMembership:    append([]string(nil), def.ProfileMembership...),
		FreshnessRequirement: def.FreshnessRequirement,
		PhaseClass:           def.PhaseClass,
		RuntimeClass:         def.RuntimeClass,
		Dimensions:           append([]string(nil), def.Dimensions...),
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
			phasePreview.Source = spec.Source
			phasePreview.DocPath = spec.Doc
			phasePreview.FindingSource = findingid.SourceToken(spec.FindingSource)
			if phasePreview.Provider == "" && spec.Delegated != nil {
				phasePreview.Provider = spec.Delegated.ProviderScenario
			}
			if len(phasePreview.ProfileMembership) == 0 {
				phasePreview.ProfileMembership = append([]string(nil), spec.ProfileMembership...)
			}
			if phasePreview.FreshnessRequirement == "" {
				phasePreview.FreshnessRequirement = spec.FreshnessRequirement
			}
			if phasePreview.PhaseClass == "" {
				phasePreview.PhaseClass = spec.PhaseClass
			}
			if phasePreview.RuntimeClass == "" {
				phasePreview.RuntimeClass = spec.RuntimeClass
			}
			if len(phasePreview.Dimensions) == 0 {
				phasePreview.Dimensions = append([]string(nil), spec.Dimensions...)
			}
		}
	}
	if phasePreview.Description == "" {
		if entry, ok := o.descriptorEntry(def.Name.String()); ok {
			if spec, ok := phases.SpecFromRegistryEntry(entry); ok {
				phasePreview.Description = spec.Description
			}
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
