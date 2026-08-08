package orchestrator

import (
	"strings"

	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/shared"
	"test-genie/internal/targetmodel"

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
	ConcurrencyMode      string                 `json:"concurrencyMode,omitempty"`
	ConcurrencyGroup     string                 `json:"concurrencyGroup,omitempty"`
	Dimensions           []string               `json:"dimensions,omitempty"`
	// RequiredResources is the provider/resource footprint of this phase. It
	// is surfaced during admission so singleton external services can be
	// serialized without making unrelated suites globally single-threaded.
	RequiredResources []string `json:"requiredResources,omitempty"`
}

// ExecutionPlanPreview captures the actual selected phase plan for a request.
type ExecutionPlanPreview struct {
	ScenarioName string `json:"scenarioName"`
	TargetKind   string `json:"targetKind,omitempty"`
	TargetID     string `json:"targetId,omitempty"`
	PresetUsed   string `json:"presetUsed,omitempty"`
	// PhaseSetDigest and DescriptorSnapshotDigest identify the exact selected
	// phase shape and provider/descriptor contract used for this preview. They
	// let the timing planner fail closed instead of treating a changed suite as
	// comparable to an older run.
	PhaseSetDigest           string         `json:"phaseSetDigest,omitempty"`
	DescriptorSnapshotDigest string         `json:"descriptorSnapshotDigest,omitempty"`
	ConfigurationFingerprint string         `json:"configurationFingerprint,omitempty"`
	Phases                   []PlannedPhase `json:"phases"`
	NotApplicablePhases      []PlannedPhase `json:"notApplicablePhases,omitempty"`
	Warnings                 []string       `json:"warnings,omitempty"`
}

type executionPlanContext struct {
	env    workspacepkg.Environment
	config *workspacepkg.Config
	plan   *phasePlan
}

func (o *SuiteOrchestrator) loadExecutionPlanContext(req SuiteExecutionRequest) (*executionPlanContext, error) {
	expression := strings.TrimSpace(req.Target)
	legacyScenario := strings.TrimSpace(req.ScenarioName)
	if expression == "" && legacyScenario == "" {
		return nil, shared.NewValidationError("target or scenarioName is required")
	}

	var ws *workspacepkg.ScenarioWorkspace
	var err error
	if expression == "" {
		// Keep the fixture/test and legacy API path independent of the repository
		// contract. A bare ScenarioName is already a validated scenario request;
		// generalized callers use Target and take the contract-backed path below.
		ws, err = workspacepkg.NewWithOptions(o.scenariosRoot, legacyScenario, workspacepkg.Options{
			ScenarioPath:           req.ScenarioPath,
			LogicalRepoRoot:        req.LogicalRepoRoot,
			LogicalScenarioRelPath: req.LogicalScenarioRelPath,
		})
	} else {
		target, resolveErr := targetmodel.Resolve(o.projectRoot, expression)
		if resolveErr != nil {
			return nil, shared.NewValidationError(resolveErr.Error())
		}
		if target.HasRuntime() {
			ws, err = workspacepkg.NewWithOptions(o.scenariosRoot, target.ID, workspacepkg.Options{
				ScenarioPath:           req.ScenarioPath,
				LogicalRepoRoot:        req.LogicalRepoRoot,
				LogicalScenarioRelPath: req.LogicalScenarioRelPath,
			})
		} else {
			ws, err = workspacepkg.NewTarget(o.projectRoot, target)
		}
	}
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
		TargetKind:   ctx.env.TargetKind,
		TargetID:     ctx.env.TargetID,
		PresetUsed:   ctx.plan.PresetUsed,
		Warnings:     buildPlanWarnings(ctx.plan),
		Phases:       make([]PlannedPhase, 0, len(ctx.plan.Selected)),
	}
	descriptorSnapshot, err := buildRunDescriptorSnapshot(ctx.plan)
	if err != nil {
		return nil, err
	}
	selected := phaseDefinitionNames(ctx.plan.Selected)
	preview.PhaseSetDigest = phases.PhaseSetDigest(selected)
	preview.DescriptorSnapshotDigest = descriptorSnapshot.Digest
	preview.ConfigurationFingerprint = ExecutionConfigurationFingerprint(req, descriptorSnapshot.Digest)

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
		ConcurrencyMode:      def.Concurrency.Mode,
		ConcurrencyGroup:     def.ProviderScenario,
		Dimensions:           append([]string(nil), def.Dimensions...),
		RequiredResources:    append([]string(nil), def.Capabilities.RequiredResources...),
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
