// This file implements workflow lifecycle service operations.
package orchestration

import (
	"context"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// WorkflowService is the immutable workflow catalog and execution boundary.
// Its 17 operations intentionally stay below the per-domain interface limit.
type WorkflowService interface {
	ValidateWorkflow(context.Context, []byte) (*WorkflowValidationResult, error)
	ReconcileScenarioWorkflows(context.Context, ReconcileScenarioWorkflowsRequest) (*ReconcileScenarioWorkflowsResult, error)
	ListWorkflowRevisions(context.Context, string, string, ListOptions) ([]*domain.WorkflowRevision, error)
	GetWorkflowRevision(context.Context, string, string, string) (*domain.WorkflowRevision, error)
	StartWorkflowExecution(context.Context, StartWorkflowExecutionRequest) (*domain.WorkflowExecution, error)
	ListWorkflowExecutions(context.Context, ListWorkflowExecutionsRequest) ([]*domain.WorkflowExecution, error)
	GetWorkflowExecution(context.Context, uuid.UUID) (*domain.WorkflowExecution, error)
	AdvanceWorkflowExecution(context.Context, uuid.UUID) (*domain.WorkflowExecution, error)
	WaitWorkflowExecution(context.Context, uuid.UUID, time.Duration) (*WaitWorkflowExecutionResult, error)
	GetWorkflowExecutionTrace(context.Context, uuid.UUID, int64, int) (*WorkflowExecutionTrace, error)
	ListWorkflowExecutionRuns(context.Context, uuid.UUID) ([]*domain.WorkflowNodeAttempt, error)
	SignalWorkflowExecution(context.Context, WorkflowExecutionSignalRequest) (*WorkflowExecutionOperationResult, error)
	CancelWorkflowExecution(context.Context, WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error)
	RetryWorkflowExecution(context.Context, WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error)
	ResumeWorkflowExecution(context.Context, WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error)
	RecoverWorkflowExecutions(context.Context) error
	SimulateWorkflow(context.Context, SimulateWorkflowRequest) (*WorkflowSimulation, error)
}

var _ WorkflowService = (*Orchestrator)(nil)
