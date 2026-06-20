package execution

import (
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"

	"github.com/google/uuid"
)

// SuiteExecutionRecord captures a persisted execution outcome.
type SuiteExecutionRecord struct {
	ID                  uuid.UUID
	SuiteRequestID      *uuid.UUID
	ScenarioName        string
	PresetUsed          string
	RequestedPreset     string
	RequestedPhases     []string
	RequestedSkipPhases []string
	PlannedPhases       []string
	FailFast            bool
	Success             bool
	// TerminalOutcome classifies the run-level result (passed | failed |
	// errored | aborted | timeout). It is the reliability ledger's denominator
	// vocabulary; catastrophic runs (no result produced) persist a row carrying
	// errored/aborted/timeout. Empty on records read from pre-migration rows
	// before backfill.
	TerminalOutcome TerminalOutcome
	Phases          []phases.ExecutionResult
	StartedAt       time.Time
	CompletedAt     time.Time
}

// ToExecutionResult converts the repository record into the orchestrator payload shared with callers.
func (r SuiteExecutionRecord) ToExecutionResult() *orchestrator.SuiteExecutionResult {
	result := &orchestrator.SuiteExecutionResult{
		ExecutionID:         r.ID,
		ScenarioName:        r.ScenarioName,
		StartedAt:           r.StartedAt,
		CompletedAt:         r.CompletedAt,
		Success:             r.Success,
		PresetUsed:          r.PresetUsed,
		RequestedPreset:     r.RequestedPreset,
		RequestedPhases:     append([]string(nil), r.RequestedPhases...),
		RequestedSkipPhases: append([]string(nil), r.RequestedSkipPhases...),
		PlannedPhases:       append([]string(nil), r.PlannedPhases...),
		FailFast:            r.FailFast,
	}
	if r.SuiteRequestID != nil {
		id := *r.SuiteRequestID
		result.SuiteRequestID = &id
	}
	if len(r.Phases) > 0 {
		result.Phases = append([]orchestrator.PhaseExecutionResult(nil), r.Phases...)
	}
	result.PhaseSummary = orchestrator.SummarizePhases(result.Phases)
	return result
}
