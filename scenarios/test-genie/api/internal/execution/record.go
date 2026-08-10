package execution

import (
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"

	"github.com/google/uuid"
)

// SuiteExecutionRecord captures a persisted execution outcome.
type SuiteExecutionRecord struct {
	ID                       uuid.UUID
	RunID                    string
	ScenarioName             string
	TargetKind               string
	TargetID                 string
	PresetUsed               string
	RequestedPreset          string
	RequestedPhases          []string
	RequestedSkipPhases      []string
	PlannedPhases            []string
	PhaseSetDigest           string
	DescriptorSnapshotDigest string
	ConfigurationFingerprint string
	HostOS                   string
	HostArch                 string
	HostNode                 string
	HostFactDigest           string
	FailFast                 bool
	SchedulerDecision        string
	Success                  bool
	// TerminalOutcome classifies the run-level result (passed | failed |
	// errored | aborted | timeout). It is the reliability ledger's denominator
	// vocabulary; catastrophic runs (no result produced) persist a row carrying
	// errored/aborted/timeout. Empty on records read from pre-migration rows
	// before backfill.
	TerminalOutcome   TerminalOutcome
	Phases            []phases.ExecutionResult
	PreparationStages []orchestrator.PreparationStage
	StartedAt         time.Time
	CompletedAt       time.Time
}

// ToExecutionResult converts the repository record into the orchestrator payload shared with callers.
func (r SuiteExecutionRecord) ToExecutionResult() *orchestrator.SuiteExecutionResult {
	result := &orchestrator.SuiteExecutionResult{
		ExecutionID:              r.ID,
		RunID:                    r.RunID,
		ScenarioName:             r.ScenarioName,
		TargetKind:               r.TargetKind,
		TargetID:                 r.TargetID,
		StartedAt:                r.StartedAt,
		CompletedAt:              r.CompletedAt,
		Success:                  r.Success,
		PresetUsed:               r.PresetUsed,
		RequestedPreset:          r.RequestedPreset,
		RequestedPhases:          append([]string(nil), r.RequestedPhases...),
		RequestedSkipPhases:      append([]string(nil), r.RequestedSkipPhases...),
		PlannedPhases:            append([]string(nil), r.PlannedPhases...),
		PhaseSetDigest:           r.PhaseSetDigest,
		DescriptorSnapshotDigest: r.DescriptorSnapshotDigest,
		ConfigurationFingerprint: r.ConfigurationFingerprint,
		FailFast:                 r.FailFast,
		SchedulerDecision:        r.SchedulerDecision,
	}
	if len(r.Phases) > 0 {
		result.Phases = append([]orchestrator.PhaseExecutionResult(nil), r.Phases...)
	}
	if len(r.PreparationStages) > 0 {
		result.PreparationStages = append([]orchestrator.PreparationStage(nil), r.PreparationStages...)
	}
	result.PhaseSummary = orchestrator.SummarizePhases(result.Phases)
	return result
}
