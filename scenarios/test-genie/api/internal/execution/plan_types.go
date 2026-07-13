package execution

import (
	"time"

	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/profileplanner"
)

// EstimateSource describes where a phase estimate came from.
type EstimateSource = profileplanner.EstimateSource

const (
	EstimateSourceScenarioHistory EstimateSource = profileplanner.EstimateSourceScenarioHistory
	EstimateSourceBlendedHistory  EstimateSource = profileplanner.EstimateSourceBlendedHistory
	EstimateSourceGlobalHistory   EstimateSource = profileplanner.EstimateSourceGlobalHistory
	EstimateSourceUnknown         EstimateSource = profileplanner.EstimateSourceUnknown
)

// EstimateConfidence summarizes how trustworthy the estimate is.
type EstimateConfidence = profileplanner.EstimateConfidence

const (
	EstimateConfidenceHigh   EstimateConfidence = profileplanner.EstimateConfidenceHigh
	EstimateConfidenceMedium EstimateConfidence = profileplanner.EstimateConfidenceMedium
	EstimateConfidenceLow    EstimateConfidence = profileplanner.EstimateConfidenceLow
)

// PlannedPhase describes a selected phase with timing guidance for operators.
type PlannedPhase struct {
	Name                     string                 `json:"name"`
	DisplayName              string                 `json:"displayName,omitempty"`
	Description              string                 `json:"description,omitempty"`
	Provider                 string                 `json:"provider,omitempty"`
	Source                   string                 `json:"source,omitempty"`
	Optional                 bool                   `json:"optional"`
	EstimatedDurationSeconds int                    `json:"estimatedDurationSeconds"`
	TimeoutSeconds           int                    `json:"timeoutSeconds"`
	EstimateSource           EstimateSource         `json:"estimateSource"`
	EstimateConfidence       EstimateConfidence     `json:"estimateConfidence"`
	EstimateSampleSize       int                    `json:"estimateSampleSize"`
	EstimateUnknown          bool                   `json:"estimateUnknown,omitempty"`
	SelectionStatus          string                 `json:"selectionStatus,omitempty"`
	SelectionReasons         []string               `json:"selectionReasons,omitempty"`
	OmissionReasons          []string               `json:"omissionReasons,omitempty"`
	ApplicabilityStatus      applicability.Status   `json:"applicabilityStatus,omitempty"`
	ApplicabilityReasons     []applicability.Reason `json:"applicabilityReasons,omitempty"`
	ProviderReadiness        string                 `json:"providerReadiness,omitempty"`
	Freshness                string                 `json:"freshness,omitempty"`
	Policy                   phasepolicy.Policy     `json:"policy,omitempty"`
	DocPath                  string                 `json:"docPath,omitempty"`
	DescriptorPath           string                 `json:"descriptorPath,omitempty"`
	FindingSource            string                 `json:"findingSource,omitempty"`
	ProfileMembership        []string               `json:"profileMembership,omitempty"`
	FreshnessRequirement     string                 `json:"freshnessRequirement,omitempty"`
	PhaseClass               string                 `json:"phaseClass,omitempty"`
	RuntimeClass             string                 `json:"runtimeClass,omitempty"`
	Dimensions               []string               `json:"dimensions,omitempty"`
}

// ExecutionPlanSummary captures total timing guidance for a plan.
type ExecutionPlanSummary struct {
	PhaseCount                   int                `json:"phaseCount"`
	EstimatedDurationSeconds     int                `json:"estimatedDurationSeconds"`
	TimeoutSeconds               int                `json:"timeoutSeconds"`
	BudgetSeconds                int                `json:"budgetSeconds,omitempty"`
	UnknownEstimateCount         int                `json:"unknownEstimateCount,omitempty"`
	EstimateSource               EstimateSource     `json:"estimateSource,omitempty"`
	EstimateConfidence           EstimateConfidence `json:"estimateConfidence,omitempty"`
	EstimateSampleSize           int                `json:"estimateSampleSize,omitempty"`
	EstimateMode                 string             `json:"estimateMode,omitempty"`
	OrchestrationOverheadSeconds int                `json:"orchestrationOverheadSeconds,omitempty"`
}

type ProfilePlan struct {
	Name          string `json:"name"`
	Strategy      string `json:"strategy"`
	BudgetSeconds int    `json:"budgetSeconds"`
}

// ExecutionPlanPreview is the scenario-aware preflight response for CLI/UI surfaces.
type ExecutionPlanPreview struct {
	ScenarioName        string               `json:"scenarioName"`
	PresetUsed          string               `json:"presetUsed,omitempty"`
	Profile             *ProfilePlan         `json:"profile,omitempty"`
	Phases              []PlannedPhase       `json:"phases"`
	OmittedPhases       []PlannedPhase       `json:"omittedPhases,omitempty"`
	NotApplicablePhases []PlannedPhase       `json:"notApplicablePhases,omitempty"`
	Summary             ExecutionPlanSummary `json:"summary"`
	Warnings            []string             `json:"warnings,omitempty"`
}

// PhaseDurationSample is a flattened historical duration observation for one phase.
type PhaseDurationSample struct {
	ScenarioName    string
	PhaseName       string
	Status          string
	DurationSeconds int
	CompletedAt     time.Time
}

// PlanDurationSample is a terminal full-run observation. Legacy rows with an
// empty comparability key deliberately cannot be treated as exact matches.
type PlanDurationSample struct {
	ScenarioName             string
	PhaseSetDigest           string
	DescriptorSnapshotDigest string
	ConfigurationFingerprint string
	TerminalOutcome          string
	DurationSeconds          int
	StartedAt                time.Time
	CompletedAt              time.Time
}
