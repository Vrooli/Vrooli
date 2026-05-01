package toolexecution

import (
	"context"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
)

// FakeOrchestrator implements the orchestration surface used by tool execution
// tests. This fake lives in a subpackage so the root mocks package does not
// import orchestration and create cycles for orchestration's own tests.
type FakeOrchestrator struct {
	CreateTaskFunc func(ctx context.Context, task *domain.Task) (*domain.Task, error)
	CreateRunFunc  func(ctx context.Context, req orchestration.CreateRunRequest) (*domain.Run, error)
	GetRunFunc     func(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	ListRunsFunc   func(ctx context.Context, opts orchestration.RunListOptions) ([]*domain.Run, error)
	StopRunFunc    func(ctx context.Context, id uuid.UUID) error
	GetRunDiffFunc func(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error)
	ApproveRunFunc func(ctx context.Context, req orchestration.ApproveRequest) (*orchestration.ApproveResult, error)
}

func NewFakeOrchestrator() *FakeOrchestrator {
	return &FakeOrchestrator{}
}

func (f *FakeOrchestrator) CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if f.CreateTaskFunc != nil {
		return f.CreateTaskFunc(ctx, task)
	}
	task.ID = uuid.New()
	task.CreatedAt = time.Now()
	return task, nil
}

func (f *FakeOrchestrator) CreateRun(ctx context.Context, req orchestration.CreateRunRequest) (*domain.Run, error) {
	if f.CreateRunFunc != nil {
		return f.CreateRunFunc(ctx, req)
	}
	return &domain.Run{
		ID:        uuid.New(),
		TaskID:    req.TaskID,
		Status:    domain.RunStatusPending,
		Phase:     domain.RunPhaseQueued,
		CreatedAt: time.Now(),
	}, nil
}

func (f *FakeOrchestrator) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	if f.GetRunFunc != nil {
		return f.GetRunFunc(ctx, id)
	}
	return nil, &domain.NotFoundError{EntityType: "run", ID: id.String()}
}

func (f *FakeOrchestrator) ListRuns(ctx context.Context, opts orchestration.RunListOptions) ([]*domain.Run, error) {
	if f.ListRunsFunc != nil {
		return f.ListRunsFunc(ctx, opts)
	}
	return []*domain.Run{}, nil
}

func (f *FakeOrchestrator) StopRun(ctx context.Context, id uuid.UUID) error {
	if f.StopRunFunc != nil {
		return f.StopRunFunc(ctx, id)
	}
	return nil
}

func (f *FakeOrchestrator) GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error) {
	if f.GetRunDiffFunc != nil {
		return f.GetRunDiffFunc(ctx, runID)
	}
	return nil, &domain.NotFoundError{EntityType: "run", ID: runID.String()}
}

func (f *FakeOrchestrator) ApproveRun(ctx context.Context, req orchestration.ApproveRequest) (*orchestration.ApproveResult, error) {
	if f.ApproveRunFunc != nil {
		return f.ApproveRunFunc(ctx, req)
	}
	return &orchestration.ApproveResult{Success: true}, nil
}
