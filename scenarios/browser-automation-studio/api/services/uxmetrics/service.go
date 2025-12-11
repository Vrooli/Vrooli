package uxmetrics

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// service is the default implementation of the Service interface.
// It orchestrates the collector, analyzer, and repository to provide
// a unified facade for UX metrics operations.
type service struct {
	collector Collector
	analyzer  Analyzer
	repo      Repository
}

// NewService creates a new UX metrics service with the provided components.
func NewService(collector Collector, analyzer Analyzer, repo Repository) Service {
	return &service{
		collector: collector,
		analyzer:  analyzer,
		repo:      repo,
	}
}

// Collector returns the collector for event pipeline integration.
func (s *service) Collector() Collector {
	return s.collector
}

// Analyzer returns the analyzer for on-demand metric computation.
func (s *service) Analyzer() Analyzer {
	return s.analyzer
}

// GetExecutionMetrics retrieves computed metrics from storage.
// Returns nil, nil if metrics have not been computed for this execution.
func (s *service) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	return s.repo.GetExecutionMetrics(ctx, executionID)
}

// ComputeAndSaveMetrics computes metrics for an execution and persists them.
// This is typically called after an execution completes.
func (s *service) ComputeAndSaveMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	// Compute metrics from collected data
	metrics, err := s.analyzer.AnalyzeExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	// Persist computed metrics
	if err := s.repo.SaveExecutionMetrics(ctx, metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// GetWorkflowAggregate retrieves aggregated metrics across recent executions.
func (s *service) GetWorkflowAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error) {
	return s.repo.GetWorkflowMetricsAggregate(ctx, workflowID, limit)
}

// Compile-time interface check
var _ Service = (*service)(nil)
