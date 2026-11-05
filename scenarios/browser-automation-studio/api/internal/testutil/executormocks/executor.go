package executormocks

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

type Call struct {
	WorkflowID uuid.UUID
	Parameters map[string]any
}

// Executor is a shared fake for the common workflow execution seam. It records
// calls and lets tests configure the result, error, or complete behavior.
type Executor struct {
	Result  *database.ExecutionIndex
	Err     error
	RunFunc func(context.Context, uuid.UUID, map[string]any) (*database.ExecutionIndex, error)

	mu    sync.Mutex
	calls []Call
}

func (e *Executor) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	e.mu.Lock()
	e.calls = append(e.calls, Call{WorkflowID: workflowID, Parameters: parameters})
	e.mu.Unlock()
	if e.RunFunc != nil {
		return e.RunFunc(ctx, workflowID, parameters)
	}
	if e.Err != nil {
		return nil, e.Err
	}
	if e.Result != nil {
		return e.Result, nil
	}
	return &database.ExecutionIndex{ID: uuid.New(), WorkflowID: workflowID, Status: database.ExecutionStatusPending}, nil
}

func (e *Executor) Calls() []Call {
	e.mu.Lock()
	defer e.mu.Unlock()
	calls := make([]Call, len(e.calls))
	copy(calls, e.calls)
	return calls
}
