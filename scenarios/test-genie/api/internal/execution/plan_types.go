package execution

import "time"

// EstimateSource describes where a phase estimate came from.
type EstimateSource string

const (
	EstimateSourceScenarioHistory EstimateSource = "scenario_history"
	EstimateSourceBlendedHistory  EstimateSource = "blended_history"
	EstimateSourceGlobalHistory   EstimateSource = "global_history"
	EstimateSourceTimeoutFallback EstimateSource = "timeout_fallback"
)

// EstimateConfidence summarizes how trustworthy the estimate is.
type EstimateConfidence string

const (
	EstimateConfidenceHigh   EstimateConfidence = "high"
	EstimateConfidenceMedium EstimateConfidence = "medium"
	EstimateConfidenceLow    EstimateConfidence = "low"
)

// PlannedPhase describes a selected phase with timing guidance for operators.
type PlannedPhase struct {
	Name                     string             `json:"name"`
	Description              string             `json:"description,omitempty"`
	Optional                 bool               `json:"optional"`
	EstimatedDurationSeconds int                `json:"estimatedDurationSeconds"`
	TimeoutSeconds           int                `json:"timeoutSeconds"`
	EstimateSource           EstimateSource     `json:"estimateSource"`
	EstimateConfidence       EstimateConfidence `json:"estimateConfidence"`
	EstimateSampleSize       int                `json:"estimateSampleSize"`
}

// ExecutionPlanSummary captures total timing guidance for a plan.
type ExecutionPlanSummary struct {
	PhaseCount               int `json:"phaseCount"`
	EstimatedDurationSeconds int `json:"estimatedDurationSeconds"`
	TimeoutSeconds           int `json:"timeoutSeconds"`
}

// ExecutionPlanPreview is the scenario-aware preflight response for CLI/UI surfaces.
type ExecutionPlanPreview struct {
	ScenarioName string               `json:"scenarioName"`
	PresetUsed   string               `json:"presetUsed,omitempty"`
	Phases       []PlannedPhase       `json:"phases"`
	Summary      ExecutionPlanSummary `json:"summary"`
	Warnings     []string             `json:"warnings,omitempty"`
}

// PhaseDurationSample is a flattened historical duration observation for one phase.
type PhaseDurationSample struct {
	ScenarioName    string
	PhaseName       string
	Status          string
	DurationSeconds int
	CompletedAt     time.Time
}
