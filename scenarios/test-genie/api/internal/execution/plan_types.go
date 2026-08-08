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
	Name                        string                 `json:"name"`
	DisplayName                 string                 `json:"displayName,omitempty"`
	Description                 string                 `json:"description,omitempty"`
	Provider                    string                 `json:"provider,omitempty"`
	Source                      string                 `json:"source,omitempty"`
	Optional                    bool                   `json:"optional"`
	EstimatedDurationSeconds    int                    `json:"estimatedDurationSeconds"`
	TimeoutSeconds              int                    `json:"timeoutSeconds"`
	EstimateSource              EstimateSource         `json:"estimateSource"`
	EstimateConfidence          EstimateConfidence     `json:"estimateConfidence"`
	EstimateSampleSize          int                    `json:"estimateSampleSize"`
	EstimatePointSampleCount    int                    `json:"estimatePointSampleCount,omitempty"`
	EstimateCensoredSampleCount int                    `json:"estimateCensoredSampleCount,omitempty"`
	EstimateExcludedSampleCount int                    `json:"estimateExcludedSampleCount,omitempty"`
	EstimateUnknown             bool                   `json:"estimateUnknown,omitempty"`
	SelectionStatus             string                 `json:"selectionStatus,omitempty"`
	SelectionReasons            []string               `json:"selectionReasons,omitempty"`
	OmissionReasons             []string               `json:"omissionReasons,omitempty"`
	ApplicabilityStatus         applicability.Status   `json:"applicabilityStatus,omitempty"`
	ApplicabilityReasons        []applicability.Reason `json:"applicabilityReasons,omitempty"`
	ProviderReadiness           string                 `json:"providerReadiness,omitempty"`
	Freshness                   string                 `json:"freshness,omitempty"`
	Policy                      phasepolicy.Policy     `json:"policy,omitempty"`
	DocPath                     string                 `json:"docPath,omitempty"`
	DescriptorPath              string                 `json:"descriptorPath,omitempty"`
	FindingSource               string                 `json:"findingSource,omitempty"`
	ProfileMembership           []string               `json:"profileMembership,omitempty"`
	FreshnessRequirement        string                 `json:"freshnessRequirement,omitempty"`
	PhaseClass                  string                 `json:"phaseClass,omitempty"`
	RuntimeClass                string                 `json:"runtimeClass,omitempty"`
	ConcurrencyMode             string                 `json:"concurrencyMode,omitempty"`
	ConcurrencyGroup            string                 `json:"concurrencyGroup,omitempty"`
	Dimensions                  []string               `json:"dimensions,omitempty"`
	RequiredResources           []string               `json:"requiredResources,omitempty"`
}

// ExecutionPlanSummary captures total timing guidance for a plan.
type ExecutionPlanSummary struct {
	PhaseCount                       int                `json:"phaseCount"`
	EstimatedDurationSeconds         int                `json:"estimatedDurationSeconds"`
	TimeoutSeconds                   int                `json:"timeoutSeconds"`
	BudgetSeconds                    int                `json:"budgetSeconds,omitempty"`
	UnknownEstimateCount             int                `json:"unknownEstimateCount,omitempty"`
	EstimateSource                   EstimateSource     `json:"estimateSource,omitempty"`
	EstimateConfidence               EstimateConfidence `json:"estimateConfidence,omitempty"`
	EstimateSampleSize               int                `json:"estimateSampleSize,omitempty"`
	EstimateMode                     string             `json:"estimateMode,omitempty"`
	OrchestrationOverheadSeconds     int                `json:"orchestrationOverheadSeconds,omitempty"`
	RequiredEstimatedDurationSeconds int                `json:"requiredEstimatedDurationSeconds,omitempty"`
	BudgetOverflowSeconds            int                `json:"budgetOverflowSeconds,omitempty"`
	BudgetExceededByRequired         bool               `json:"budgetExceededByRequired,omitempty"`
	BudgetFitMode                    string             `json:"budgetFitMode,omitempty"`
	BudgetConditions                 []string           `json:"budgetConditions,omitempty"`
}

type ProfilePlan struct {
	Name          string `json:"name"`
	Strategy      string `json:"strategy"`
	BudgetSeconds int    `json:"budgetSeconds"`
}

// ExecutionPlanPreview is the scenario-aware preflight response for CLI/UI surfaces.
type ExecutionPlanPreview struct {
	ScenarioName             string               `json:"scenarioName"`
	PresetUsed               string               `json:"presetUsed,omitempty"`
	Profile                  *ProfilePlan         `json:"profile,omitempty"`
	Phases                   []PlannedPhase       `json:"phases"`
	OmittedPhases            []PlannedPhase       `json:"omittedPhases,omitempty"`
	NotApplicablePhases      []PlannedPhase       `json:"notApplicablePhases,omitempty"`
	Summary                  ExecutionPlanSummary `json:"summary"`
	Warnings                 []string             `json:"warnings,omitempty"`
	PhaseSetDigest           string               `json:"phaseSetDigest,omitempty"`
	DescriptorSnapshotDigest string               `json:"descriptorSnapshotDigest,omitempty"`
	ConfigurationFingerprint string               `json:"configurationFingerprint,omitempty"`
}

// PhaseDurationSample is a flattened historical duration observation for one phase.
type PhaseDurationSample struct {
	ScenarioName string
	PhaseName    string
	Status       string
	// DurationMilliseconds is the persisted phase-history unit. Conversion to
	// planner seconds happens once in plannerSamples.
	DurationMilliseconds int64
	// DurationSeconds is retained for old rows and fixture readers; it is not
	// used when milliseconds are available.
	DurationSeconds int
	CompletedAt     time.Time
}

// PlanDurationSample is a terminal full-run observation. Legacy rows with an
// empty comparability key deliberately cannot be treated as exact matches.
type PlanDurationSample struct {
	ScenarioName              string
	PhaseSetDigest            string
	DescriptorSnapshotDigest  string
	ConfigurationFingerprint  string
	TerminalOutcome           string
	DurationSeconds           int
	PhaseDurationMilliseconds int64
	StartedAt                 time.Time
	CompletedAt               time.Time
}
