package execution

import (
	"context"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/profileplanner"
)

const (
	planHistoryWindow = 90 * 24 * time.Hour
	maxHistoryRows    = 2000
	// Comprehensive provider startup, dependency readiness, persistence, and
	// terminal projection routinely exceed a minute. Keep two minutes explicit
	// until exact comparable full-run evidence is available.
	additiveOrchestrationOverheadSeconds = 120
)

// ExecutionPlanner exposes scenario-aware plan previews for API/UI/CLI surfaces.
type ExecutionPlanner interface {
	Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*ExecutionPlanPreview, error)
}

type executionPlanBuilder interface {
	PreviewExecution(req orchestrator.SuiteExecutionRequest) (*orchestrator.ExecutionPlanPreview, error)
}

type phaseSampleReader interface {
	ListPhaseSamples(ctx context.Context, phaseNames []string, since time.Time, limit int) ([]PhaseDurationSample, error)
	ListPlanSamples(ctx context.Context, scenario string, since time.Time, limit int) ([]PlanDurationSample, error)
}

// ExecutionPlanService builds scenario-aware phase plans enriched with historical estimates.
type ExecutionPlanService struct {
	builder executionPlanBuilder
	samples phaseSampleReader
	now     func() time.Time
}

func NewExecutionPlanService(builder executionPlanBuilder, samples phaseSampleReader) *ExecutionPlanService {
	return &ExecutionPlanService{
		builder: builder,
		samples: samples,
		now:     time.Now,
	}
}

// Preview returns the actual selected phase plan plus timing guidance.
func (s *ExecutionPlanService) Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*ExecutionPlanPreview, error) {
	basePlan, err := s.previewBasePlan(req)
	if err != nil {
		return nil, err
	}

	preview := &ExecutionPlanPreview{
		ScenarioName:        basePlan.ScenarioName,
		PresetUsed:          basePlan.PresetUsed,
		Warnings:            append([]string(nil), basePlan.Warnings...),
		Phases:              make([]PlannedPhase, 0, len(basePlan.Phases)),
		NotApplicablePhases: make([]PlannedPhase, 0, len(basePlan.NotApplicablePhases)),
	}

	phaseNames := make([]string, 0, len(basePlan.Phases))
	for _, phase := range basePlan.Phases {
		phaseNames = append(phaseNames, phase.Name)
	}

	since := s.now().UTC().Add(-planHistoryWindow)
	var phaseSamples []PhaseDurationSample
	if len(phaseNames) > 0 {
		var err error
		phaseSamples, err = s.samples.ListPhaseSamples(ctx, phaseNames, since, maxHistoryRows)
		if err != nil {
			return nil, err
		}
	}

	estimator := profileplanner.NewEstimator(basePlan.ScenarioName, plannerSamples(phaseSamples))

	if profile, ok := adaptiveProfileForRequest(req, basePlan.PresetUsed); ok {
		profilePlan := profileplanner.PlanProfile(profile, plannerCandidates(basePlan.Phases), estimator)
		preview.PresetUsed = profilePlan.Profile.Name
		preview.Profile = &ProfilePlan{
			Name:          profilePlan.Profile.Name,
			Strategy:      string(profilePlan.Profile.Strategy),
			BudgetSeconds: profilePlan.Profile.BudgetSeconds,
		}
		preview.Summary.BudgetSeconds = profilePlan.Profile.BudgetSeconds
		preview.Summary.UnknownEstimateCount = profilePlan.UnknownEstimateCount

		for _, decision := range profilePlan.Selected {
			phase, ok := findBasePlannedPhase(basePlan.Phases, decision.Candidate.Name)
			if !ok {
				continue
			}
			previewPhase := plannedPhaseWithEstimate(phase, decision.Estimate)
			previewPhase.SelectionReasons = append([]string(nil), decision.Reasons...)
			preview.Phases = append(preview.Phases, previewPhase)
			addSelectedPhaseSummary(&preview.Summary, previewPhase)
		}
		for _, decision := range profilePlan.Omitted {
			phase, ok := findBasePlannedPhase(basePlan.Phases, decision.Candidate.Name)
			if !ok {
				continue
			}
			previewPhase := plannedPhaseWithEstimate(phase, decision.Estimate)
			previewPhase.SelectionStatus = "omitted"
			previewPhase.OmissionReasons = append([]string(nil), decision.Reasons...)
			preview.OmittedPhases = append(preview.OmittedPhases, previewPhase)
		}
		appendNotApplicablePhases(preview, basePlan.NotApplicablePhases)
		return s.applyRunLevelEstimate(ctx, req, basePlan, preview, since)
	}

	for _, phase := range basePlan.Phases {
		timeoutSeconds := clampNonNegative(phase.TimeoutSeconds)
		estimate := estimator.Estimate(phase.Name, timeoutSeconds)
		previewPhase := plannedPhaseWithEstimate(phase, estimate)
		preview.Phases = append(preview.Phases, previewPhase)
		addSelectedPhaseSummary(&preview.Summary, previewPhase)
	}
	appendNotApplicablePhases(preview, basePlan.NotApplicablePhases)
	return s.applyRunLevelEstimate(ctx, req, basePlan, preview, since)
}

// applyRunLevelEstimate first looks for an exact, same-scenario full-run key.
// It deliberately refuses legacy or mismatched descriptor/config rows: a phase
// set digest existed to make this comparison fail closed, not merely to appear
// in the durable run index. When no exact history exists, phase estimates remain
// useful but are truthfully labeled additive and include fixed overhead.
func (s *ExecutionPlanService) applyRunLevelEstimate(ctx context.Context, req orchestrator.SuiteExecutionRequest, base *orchestrator.ExecutionPlanPreview, preview *ExecutionPlanPreview, since time.Time) (*ExecutionPlanPreview, error) {
	if preview == nil || base == nil {
		return preview, nil
	}
	planSamples, err := s.samples.ListPlanSamples(ctx, base.ScenarioName, since, maxHistoryRows)
	if err != nil {
		return nil, err
	}
	comparable := comparableRunSamples(planSamples, base.PhaseSetDigest, base.DescriptorSnapshotDigest, base.ConfigurationFingerprint)
	if len(comparable) > 0 {
		estimate := profileplanner.EstimateComparableRun(comparable)
		preview.Summary.EstimatedDurationSeconds = estimate.DurationSeconds
		preview.Summary.EstimateSource = estimate.Source
		preview.Summary.EstimateConfidence = estimate.Confidence
		preview.Summary.EstimateSampleSize = estimate.SampleSize
		preview.Summary.EstimateMode = "comparable_full_run"
		return preview, nil
	}
	// A custom phase selection and a preset use the same fallback: selected
	// phases are real, but their historical total is absent or non-comparable.
	_ = req
	preview.Summary.EstimatedDurationSeconds += additiveOrchestrationOverheadSeconds
	preview.Summary.OrchestrationOverheadSeconds = additiveOrchestrationOverheadSeconds
	preview.Summary.EstimateSource = EstimateSourceBlendedHistory
	preview.Summary.EstimateConfidence = EstimateConfidenceLow
	preview.Summary.EstimateMode = "additive_phase_history"
	return preview, nil
}

func comparableRunSamples(samples []PlanDurationSample, phaseSetDigest, descriptorDigest, configurationFingerprint string) []profileplanner.RunSample {
	if phaseSetDigest == "" || descriptorDigest == "" || configurationFingerprint == "" {
		return nil
	}
	out := make([]profileplanner.RunSample, 0, len(samples))
	for _, sample := range samples {
		if sample.PhaseSetDigest != phaseSetDigest || sample.DescriptorSnapshotDigest != descriptorDigest || sample.ConfigurationFingerprint != configurationFingerprint {
			continue
		}
		out = append(out, profileplanner.RunSample{DurationSeconds: sample.DurationSeconds, TerminalOutcome: sample.TerminalOutcome, CompletedAt: sample.CompletedAt})
	}
	return out
}

func (s *ExecutionPlanService) previewBasePlan(req orchestrator.SuiteExecutionRequest) (*orchestrator.ExecutionPlanPreview, error) {
	if _, ok := adaptiveProfileForRequest(req, ""); !ok {
		return s.builder.PreviewExecution(req)
	}
	candidateReq := req
	candidateReq.Preset = phases.PresetComprehensive.String()
	return s.builder.PreviewExecution(candidateReq)
}

func adaptiveProfileForRequest(req orchestrator.SuiteExecutionRequest, presetUsed string) (profileplanner.Profile, bool) {
	if len(req.Phases) > 0 {
		return profileplanner.Profile{}, false
	}
	name := phases.NormalizeKey(req.Preset)
	if name == "" {
		name = phases.NormalizeKey(presetUsed)
	}
	switch name {
	case phases.PresetQuick.String():
		profile, _ := phases.AdaptiveProfile(name)
		return profileplanner.Profile{
			Name:          profile.Name.String(),
			BudgetSeconds: profile.BudgetSeconds,
			Strategy:      profileplanner.ProfileStrategy(profile.Strategy),
		}, true
	case phases.PresetSmoke.String():
		profile, _ := phases.AdaptiveProfile(name)
		return profileplanner.Profile{
			Name:          profile.Name.String(),
			BudgetSeconds: profile.BudgetSeconds,
			Strategy:      profileplanner.ProfileStrategy(profile.Strategy),
		}, true
	default:
		return profileplanner.Profile{}, false
	}
}

func plannerSamples(samples []PhaseDurationSample) []profileplanner.Sample {
	out := make([]profileplanner.Sample, 0, len(samples))
	for _, sample := range samples {
		out = append(out, profileplanner.Sample{
			ScenarioName:    sample.ScenarioName,
			PhaseName:       sample.PhaseName,
			Status:          sample.Status,
			DurationSeconds: sample.DurationSeconds,
			CompletedAt:     sample.CompletedAt,
		})
	}
	return out
}

func plannerCandidates(phases []orchestrator.PlannedPhase) []profileplanner.Candidate {
	out := make([]profileplanner.Candidate, 0, len(phases))
	for index, phase := range phases {
		out = append(out, profileplanner.Candidate{
			Name:           phase.Name,
			DisplayName:    phase.DisplayName,
			TimeoutSeconds: clampNonNegative(phase.TimeoutSeconds),
			Policy:         phase.Policy,
			Order:          index,
		})
	}
	return out
}

func plannedPhaseWithEstimate(phase orchestrator.PlannedPhase, estimate profileplanner.Estimate) PlannedPhase {
	return PlannedPhase{
		Name:                     phase.Name,
		DisplayName:              phase.DisplayName,
		Description:              phase.Description,
		Provider:                 phase.Provider,
		Source:                   phase.Source,
		Optional:                 phase.Optional,
		EstimatedDurationSeconds: estimate.DurationSeconds,
		TimeoutSeconds:           clampNonNegative(phase.TimeoutSeconds),
		EstimateSource:           estimate.Source,
		EstimateConfidence:       estimate.Confidence,
		EstimateSampleSize:       estimate.SampleSize,
		EstimateUnknown:          estimate.Unknown,
		SelectionStatus:          phase.SelectionStatus,
		ApplicabilityStatus:      phase.ApplicabilityStatus,
		ApplicabilityReasons:     append([]applicability.Reason(nil), phase.ApplicabilityReasons...),
		ProviderReadiness:        phase.ProviderReadiness,
		Freshness:                phase.Freshness,
		Policy:                   phase.Policy,
		DocPath:                  phase.DocPath,
		DescriptorPath:           phase.DescriptorPath,
		FindingSource:            phase.FindingSource,
		ProfileMembership:        append([]string(nil), phase.ProfileMembership...),
		FreshnessRequirement:     phase.FreshnessRequirement,
		PhaseClass:               phase.PhaseClass,
		RuntimeClass:             phase.RuntimeClass,
		Dimensions:               append([]string(nil), phase.Dimensions...),
	}
}

func appendNotApplicablePhases(preview *ExecutionPlanPreview, phases []orchestrator.PlannedPhase) {
	for _, phase := range phases {
		preview.NotApplicablePhases = append(preview.NotApplicablePhases, PlannedPhase{
			Name:                 phase.Name,
			DisplayName:          phase.DisplayName,
			Description:          phase.Description,
			Provider:             phase.Provider,
			Source:               phase.Source,
			Optional:             phase.Optional,
			TimeoutSeconds:       clampNonNegative(phase.TimeoutSeconds),
			SelectionStatus:      phase.SelectionStatus,
			ApplicabilityStatus:  phase.ApplicabilityStatus,
			ApplicabilityReasons: append([]applicability.Reason(nil), phase.ApplicabilityReasons...),
			ProviderReadiness:    phase.ProviderReadiness,
			Freshness:            phase.Freshness,
			Policy:               phase.Policy,
			DocPath:              phase.DocPath,
			DescriptorPath:       phase.DescriptorPath,
			FindingSource:        phase.FindingSource,
			ProfileMembership:    append([]string(nil), phase.ProfileMembership...),
			FreshnessRequirement: phase.FreshnessRequirement,
			PhaseClass:           phase.PhaseClass,
			RuntimeClass:         phase.RuntimeClass,
			Dimensions:           append([]string(nil), phase.Dimensions...),
		})
	}
}

func addSelectedPhaseSummary(summary *ExecutionPlanSummary, phase PlannedPhase) {
	summary.PhaseCount++
	summary.EstimatedDurationSeconds += phase.EstimatedDurationSeconds
	summary.TimeoutSeconds += phase.TimeoutSeconds
}

func findBasePlannedPhase(items []orchestrator.PlannedPhase, name string) (orchestrator.PlannedPhase, bool) {
	key := phases.NormalizeKey(name)
	for _, item := range items {
		if phases.NormalizeKey(item.Name) == key {
			return item, true
		}
	}
	return orchestrator.PlannedPhase{}, false
}

func clampNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

var _ ExecutionPlanner = (*ExecutionPlanService)(nil)
