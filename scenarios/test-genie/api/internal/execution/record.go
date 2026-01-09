package execution

import (
	"time"

	"github.com/google/uuid"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
)

// SuiteExecutionRecord captures a persisted execution outcome.
type SuiteExecutionRecord struct {
	ID             uuid.UUID
	SuiteRequestID *uuid.UUID
	ScenarioName   string
	PresetUsed     string
	Success        bool
	Phases         []phases.ExecutionResult
	StartedAt      time.Time
	CompletedAt    time.Time
}

// ToExecutionResult converts the repository record into the orchestrator payload shared with callers.
func (r SuiteExecutionRecord) ToExecutionResult() *orchestrator.SuiteExecutionResult {
	result := &orchestrator.SuiteExecutionResult{
		ExecutionID:  r.ID,
		ScenarioName: r.ScenarioName,
		StartedAt:    r.StartedAt,
		CompletedAt:  r.CompletedAt,
		Success:      r.Success,
		PresetUsed:   r.PresetUsed,
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
