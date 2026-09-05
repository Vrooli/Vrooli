// Package mocks provides controllable collaborators for orchestration tests.
package mocks

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/adapters/sandbox"

	"github.com/google/uuid"
)

var _ sandbox.Provider = (*FakeSandboxProvider)(nil)

// FakeSandboxProvider is a reusable fake for the workspace-sandbox seam.
type FakeSandboxProvider struct {
	mu sync.Mutex

	CreateFunc         func(context.Context, sandbox.CreateRequest) (*sandbox.Sandbox, error)
	GetFunc            func(context.Context, uuid.UUID) (*sandbox.Sandbox, error)
	DeleteFunc         func(context.Context, uuid.UUID) error
	GetWorkspacePathFn func(context.Context, uuid.UUID) (string, error)
	GetDiffFunc        func(context.Context, uuid.UUID) (*sandbox.DiffResult, error)
	ApproveFunc        func(context.Context, sandbox.ApproveRequest) (*sandbox.ApproveResult, error)
	RejectFunc         func(context.Context, uuid.UUID, string) error
	PartialApproveFunc func(context.Context, sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error)
	StopFunc           func(context.Context, uuid.UUID) error
	StartFunc          func(context.Context, uuid.UUID) error
	ResumeFunc         func(context.Context, uuid.UUID) (*sandbox.Sandbox, error)
	IsAvailableFunc    func(context.Context) (bool, string)
	ValidatePathFunc   func(context.Context, string, string) (*sandbox.PathValidationResult, error)
	ExecProcessFunc    func(context.Context, sandbox.ExecProcessRequest) (*sandbox.ExecProcessResult, error)
	ApplyAtRunEndFunc  func(context.Context, sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error)
	TurnCheckpointFunc func(context.Context, sandbox.TurnCheckpointRequest) (*sandbox.TurnCheckpointResult, error)

	DeleteErr            error
	StopErr              error
	ApplyAtRunEndErr     error
	ApplyAtRunEndResult  *sandbox.ApplyAtRunEndResult
	TurnCheckpointErr    error
	TurnCheckpointResult *sandbox.TurnCheckpointResult

	createRequests         []sandbox.CreateRequest
	getWorkspacePathIDs    []uuid.UUID
	deleteIDs              []uuid.UUID
	deleteContextErrs      []error
	stopIDs                []uuid.UUID
	applyAtRunEndRequests  []sandbox.ApplyAtRunEndRequest
	turnCheckpointRequests []sandbox.TurnCheckpointRequest
}

func NewFakeSandboxProvider() *FakeSandboxProvider {
	sandboxID := uuid.New()
	return &FakeSandboxProvider{
		CreateFunc: func(_ context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			return &sandbox.Sandbox{
				ID:          sandboxID,
				ScopePath:   req.ScopePath,
				ProjectRoot: req.ProjectRoot,
				Status:      sandbox.SandboxStatusActive,
				WorkDir:     "/tmp/sandbox/" + sandboxID.String(),
				CreatedAt:   time.Now(),
			}, nil
		},
		GetWorkspacePathFn: func(_ context.Context, id uuid.UUID) (string, error) {
			return "/tmp/sandbox/" + id.String() + "/merged", nil
		},
	}
}

func (p *FakeSandboxProvider) Create(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	p.mu.Lock()
	p.createRequests = append(p.createRequests, req)
	p.mu.Unlock()
	if p.CreateFunc != nil {
		return p.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (p *FakeSandboxProvider) Get(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	if p.GetFunc != nil {
		return p.GetFunc(ctx, id)
	}
	return nil, nil
}

func (p *FakeSandboxProvider) Delete(ctx context.Context, id uuid.UUID) error {
	p.mu.Lock()
	p.deleteIDs = append(p.deleteIDs, id)
	p.deleteContextErrs = append(p.deleteContextErrs, ctx.Err())
	p.mu.Unlock()
	if p.DeleteFunc != nil {
		return p.DeleteFunc(ctx, id)
	}
	return p.DeleteErr
}

func (p *FakeSandboxProvider) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	p.mu.Lock()
	p.getWorkspacePathIDs = append(p.getWorkspacePathIDs, id)
	p.mu.Unlock()
	if p.GetWorkspacePathFn != nil {
		return p.GetWorkspacePathFn(ctx, id)
	}
	return "", nil
}

func (p *FakeSandboxProvider) GetDiff(ctx context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
	if p.GetDiffFunc != nil {
		return p.GetDiffFunc(ctx, id)
	}
	return &sandbox.DiffResult{}, nil
}

func (p *FakeSandboxProvider) Approve(ctx context.Context, req sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
	if p.ApproveFunc != nil {
		return p.ApproveFunc(ctx, req)
	}
	return &sandbox.ApproveResult{Success: true}, nil
}

func (p *FakeSandboxProvider) Reject(ctx context.Context, id uuid.UUID, actor string) error {
	if p.RejectFunc != nil {
		return p.RejectFunc(ctx, id, actor)
	}
	return nil
}

func (p *FakeSandboxProvider) PartialApprove(ctx context.Context, req sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
	if p.PartialApproveFunc != nil {
		return p.PartialApproveFunc(ctx, req)
	}
	return &sandbox.ApproveResult{Success: true}, nil
}

func (p *FakeSandboxProvider) ApplyAtRunEnd(ctx context.Context, req sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
	p.mu.Lock()
	p.applyAtRunEndRequests = append(p.applyAtRunEndRequests, req)
	p.mu.Unlock()
	if p.ApplyAtRunEndFunc != nil {
		return p.ApplyAtRunEndFunc(ctx, req)
	}
	if p.ApplyAtRunEndErr != nil {
		return nil, p.ApplyAtRunEndErr
	}
	if p.ApplyAtRunEndResult != nil {
		return p.ApplyAtRunEndResult, nil
	}
	return &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1, AppliedAt: time.Now()}, nil
}

func (p *FakeSandboxProvider) TurnCheckpoint(ctx context.Context, req sandbox.TurnCheckpointRequest) (*sandbox.TurnCheckpointResult, error) {
	p.mu.Lock()
	p.turnCheckpointRequests = append(p.turnCheckpointRequests, req)
	p.mu.Unlock()
	if p.TurnCheckpointFunc != nil {
		return p.TurnCheckpointFunc(ctx, req)
	}
	if p.TurnCheckpointErr != nil {
		return nil, p.TurnCheckpointErr
	}
	if p.TurnCheckpointResult != nil {
		return p.TurnCheckpointResult, nil
	}
	return &sandbox.TurnCheckpointResult{
		SandboxID: req.SandboxID,
		Status:    sandbox.SandboxStatusCheckpointed,
		Success:   true,
		Applied:   1,
		AppliedAt: time.Now(),
	}, nil
}

func (p *FakeSandboxProvider) Stop(ctx context.Context, id uuid.UUID) error {
	p.mu.Lock()
	p.stopIDs = append(p.stopIDs, id)
	p.mu.Unlock()
	if p.StopFunc != nil {
		return p.StopFunc(ctx, id)
	}
	return p.StopErr
}

func (p *FakeSandboxProvider) Start(ctx context.Context, id uuid.UUID) error {
	if p.StartFunc != nil {
		return p.StartFunc(ctx, id)
	}
	return nil
}

func (p *FakeSandboxProvider) Resume(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	if p.ResumeFunc != nil {
		return p.ResumeFunc(ctx, id)
	}
	return &sandbox.Sandbox{
		ID:        id,
		Status:    sandbox.SandboxStatusActive,
		WorkDir:   "/tmp/sandbox/" + id.String() + "/merged",
		CreatedAt: time.Now(),
	}, nil
}

func (p *FakeSandboxProvider) IsAvailable(ctx context.Context) (bool, string) {
	if p.IsAvailableFunc != nil {
		return p.IsAvailableFunc(ctx)
	}
	return true, ""
}

func (p *FakeSandboxProvider) ValidatePath(ctx context.Context, path string, projectRoot string) (*sandbox.PathValidationResult, error) {
	if p.ValidatePathFunc != nil {
		return p.ValidatePathFunc(ctx, path, projectRoot)
	}
	return &sandbox.PathValidationResult{Path: path, Valid: true}, nil
}

func (p *FakeSandboxProvider) ExecProcess(ctx context.Context, req sandbox.ExecProcessRequest) (*sandbox.ExecProcessResult, error) {
	if p.ExecProcessFunc != nil {
		return p.ExecProcessFunc(ctx, req)
	}
	return &sandbox.ExecProcessResult{ExitCode: 0}, nil
}

func (p *FakeSandboxProvider) CreateRequests() []sandbox.CreateRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]sandbox.CreateRequest(nil), p.createRequests...)
}

func (p *FakeSandboxProvider) GetWorkspacePathCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.getWorkspacePathIDs)
}

func (p *FakeSandboxProvider) GetWorkspacePathIDs() []uuid.UUID {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]uuid.UUID(nil), p.getWorkspacePathIDs...)
}

func (p *FakeSandboxProvider) DeleteCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.deleteIDs)
}

func (p *FakeSandboxProvider) DeleteContextErrs() []error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]error(nil), p.deleteContextErrs...)
}

func (p *FakeSandboxProvider) StopCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.stopIDs)
}

func (p *FakeSandboxProvider) ApplyAtRunEndCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.applyAtRunEndRequests)
}

func (p *FakeSandboxProvider) ApplyAtRunEndRequests() []sandbox.ApplyAtRunEndRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]sandbox.ApplyAtRunEndRequest(nil), p.applyAtRunEndRequests...)
}

func (p *FakeSandboxProvider) TurnCheckpointCallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.turnCheckpointRequests)
}

func (p *FakeSandboxProvider) TurnCheckpointRequests() []sandbox.TurnCheckpointRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]sandbox.TurnCheckpointRequest(nil), p.turnCheckpointRequests...)
}
