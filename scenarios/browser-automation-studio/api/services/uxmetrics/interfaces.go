// Package uxmetrics provides user experience metrics collection and analysis.
// It captures interaction data from workflow executions and computes friction
// scores to help identify usability issues.
//
// # Architecture
//
// The UX metrics system follows a decorator pattern:
//   - Collector wraps the existing EventSink to passively capture data
//   - Analyzer computes metrics from collected interaction traces
//   - Repository handles persistence to PostgreSQL
//   - Service orchestrates the subsystem
//
// # Entitlement
//
// UX metrics are gated to Pro tier and above. Handlers should check
// entitlement before calling service methods.
package uxmetrics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// StepOutcomeData is the subset of automation StepOutcome we consume.
// This provides dependency inversion - we don't depend on the full contracts package.
type StepOutcomeData struct {
	StepIndex   int
	NodeID      string
	StepType    string
	Success     bool
	DurationMs  int
	Position    *contracts.Point
	CursorTrail []contracts.Point
	StartedAt   time.Time
	CompletedAt *time.Time
}

// Collector is the ingestion interface for UX metrics data.
// It sits in the event pipeline and captures interaction data passively.
type Collector interface {
	// OnStepOutcome receives step completion data from the executor.
	OnStepOutcome(ctx context.Context, executionID uuid.UUID, outcome StepOutcomeData) error

	// OnCursorUpdate receives cursor position updates during recording.
	OnCursorUpdate(ctx context.Context, executionID uuid.UUID, stepIndex int, point contracts.TimedPoint) error

	// FlushExecution finalizes data collection for an execution.
	FlushExecution(ctx context.Context, executionID uuid.UUID) error
}

// Analyzer computes metrics from raw interaction data.
type Analyzer interface {
	// AnalyzeExecution computes all metrics for a completed execution.
	AnalyzeExecution(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)

	// AnalyzeStep computes metrics for a single step (for real-time).
	AnalyzeStep(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error)
}

// Repository handles persistence of UX metrics data.
type Repository interface {
	// Raw data persistence (write path)
	SaveInteractionTrace(ctx context.Context, trace *contracts.InteractionTrace) error
	SaveCursorPath(ctx context.Context, executionID uuid.UUID, path *contracts.CursorPath) error

	// Computed metrics persistence
	SaveExecutionMetrics(ctx context.Context, metrics *contracts.ExecutionMetrics) error

	// Query path
	GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)
	GetStepMetrics(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error)
	ListInteractionTraces(ctx context.Context, executionID uuid.UUID) ([]contracts.InteractionTrace, error)
	GetCursorPath(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.CursorPath, error)

	// Aggregation queries (for dashboards)
	GetWorkflowMetricsAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error)
}

// Service is the public facade for the UX metrics subsystem.
type Service interface {
	// Collector returns the collector for event pipeline integration.
	Collector() Collector

	// Analyzer returns the analyzer for on-demand metric computation.
	Analyzer() Analyzer

	// GetExecutionMetrics retrieves computed metrics (from cache/storage).
	GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)

	// ComputeAndSaveMetrics computes and persists metrics for an execution.
	ComputeAndSaveMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error)

	// GetWorkflowAggregate retrieves aggregated metrics across executions.
	GetWorkflowAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error)
}
