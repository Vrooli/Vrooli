package sandboxiface

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// FakeService implements sandbox.ServiceAPI for handler tests using a
// function-pointer pattern: every interface method dispatches to a
// `*Fn` field. Unset fields return a NotImplemented error so a test
// that forgets to wire a method gets a clear failure instead of a
// silent zero-value response.
//
// This pattern (versus the larger state-machine pattern used by
// FakeRepository / FakeDriver) suits the handler-test surface where
// each test cares about exactly one or two service calls.
type FakeService struct {
	CreateFn             func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error)
	GetFn                func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	ListFn               func(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error)
	StopFn               func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	StartFn              func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
	DeleteFn             func(ctx context.Context, id uuid.UUID) error
	GetDiffFn            func(ctx context.Context, id uuid.UUID) (*types.DiffResult, error)
	ApproveFn            func(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error)
	ApplyAtRunEndFn      func(ctx context.Context, req *types.ApplyAtRunEndRequest) (*types.ApprovalResult, error)
	RejectFn             func(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error)
	DiscardFn            func(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error)
	GetWorkspacePathFn   func(ctx context.Context, id uuid.UUID) (string, error)
	CheckConflictsFn     func(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error)
	RebaseFn             func(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error)
	ValidatePathFn       func(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error)
	GetPendingChangesFn  func(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error)
	GetFileProvenanceFn  func(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error)
	GetCommitPreviewFn   func(ctx context.Context, req *types.CommitPreviewRequest) (*types.CommitPreviewResult, error)
	CommitPendingFn      func(ctx context.Context, req *types.CommitPendingRequest) (*types.CommitPendingResult, error)
	MarkCommittedFn      func(ctx context.Context, req *types.MarkCommittedRequest) (*types.MarkCommittedResult, error)
	GetProvenanceByRunFn func(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error)
}

// NewFakeService returns a fresh FakeService with all function pointers
// nil. Tests set only the methods they need.
func NewFakeService() *FakeService {
	return &FakeService{}
}

func notImpl(method string) error {
	return fmt.Errorf("FakeService.%s not implemented (set the corresponding *Fn field on the FakeService)", method)
}

func (m *FakeService) Create(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req)
	}
	return nil, notImpl("Create")
}

func (m *FakeService) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return nil, notImpl("Get")
}

func (m *FakeService) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, notImpl("List")
}

func (m *FakeService) Stop(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	if m.StopFn != nil {
		return m.StopFn(ctx, id)
	}
	return nil, notImpl("Stop")
}

func (m *FakeService) Start(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	if m.StartFn != nil {
		return m.StartFn(ctx, id)
	}
	return nil, notImpl("Start")
}

func (m *FakeService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return notImpl("Delete")
}

func (m *FakeService) GetDiff(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
	if m.GetDiffFn != nil {
		return m.GetDiffFn(ctx, id)
	}
	return nil, notImpl("GetDiff")
}

func (m *FakeService) Approve(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
	if m.ApproveFn != nil {
		return m.ApproveFn(ctx, req)
	}
	return nil, notImpl("Approve")
}

func (m *FakeService) ApplyAtRunEnd(ctx context.Context, req *types.ApplyAtRunEndRequest) (*types.ApprovalResult, error) {
	if m.ApplyAtRunEndFn != nil {
		return m.ApplyAtRunEndFn(ctx, req)
	}
	return nil, notImpl("ApplyAtRunEnd")
}

func (m *FakeService) Reject(ctx context.Context, id uuid.UUID, actor string) (*types.Sandbox, error) {
	if m.RejectFn != nil {
		return m.RejectFn(ctx, id, actor)
	}
	return nil, notImpl("Reject")
}

func (m *FakeService) Discard(ctx context.Context, req *types.DiscardRequest) (*types.DiscardResult, error) {
	if m.DiscardFn != nil {
		return m.DiscardFn(ctx, req)
	}
	return nil, notImpl("Discard")
}

func (m *FakeService) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	if m.GetWorkspacePathFn != nil {
		return m.GetWorkspacePathFn(ctx, id)
	}
	return "", notImpl("GetWorkspacePath")
}

func (m *FakeService) CheckConflicts(ctx context.Context, id uuid.UUID) (*types.ConflictCheckResponse, error) {
	if m.CheckConflictsFn != nil {
		return m.CheckConflictsFn(ctx, id)
	}
	return nil, notImpl("CheckConflicts")
}

func (m *FakeService) Rebase(ctx context.Context, req *types.RebaseRequest) (*types.RebaseResult, error) {
	if m.RebaseFn != nil {
		return m.RebaseFn(ctx, req)
	}
	return nil, notImpl("Rebase")
}

func (m *FakeService) ValidatePath(ctx context.Context, path, projectRoot string) (*types.PathValidationResult, error) {
	if m.ValidatePathFn != nil {
		return m.ValidatePathFn(ctx, path, projectRoot)
	}
	return nil, notImpl("ValidatePath")
}

func (m *FakeService) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	if m.GetPendingChangesFn != nil {
		return m.GetPendingChangesFn(ctx, projectRoot, limit, offset)
	}
	return nil, notImpl("GetPendingChanges")
}

func (m *FakeService) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	if m.GetFileProvenanceFn != nil {
		return m.GetFileProvenanceFn(ctx, filePath, projectRoot, limit)
	}
	return nil, notImpl("GetFileProvenance")
}

func (m *FakeService) GetCommitPreview(ctx context.Context, req *types.CommitPreviewRequest) (*types.CommitPreviewResult, error) {
	if m.GetCommitPreviewFn != nil {
		return m.GetCommitPreviewFn(ctx, req)
	}
	return nil, notImpl("GetCommitPreview")
}

func (m *FakeService) CommitPending(ctx context.Context, req *types.CommitPendingRequest) (*types.CommitPendingResult, error) {
	if m.CommitPendingFn != nil {
		return m.CommitPendingFn(ctx, req)
	}
	return nil, notImpl("CommitPending")
}

func (m *FakeService) MarkCommitted(ctx context.Context, req *types.MarkCommittedRequest) (*types.MarkCommittedResult, error) {
	if m.MarkCommittedFn != nil {
		return m.MarkCommittedFn(ctx, req)
	}
	// Default behavior: mark all paths in the request as committed.
	return &types.MarkCommittedResult{MarkedCount: len(req.FilePaths)}, nil
}

func (m *FakeService) GetProvenanceByRun(ctx context.Context, projectRoot string) ([]types.ProvenanceRunGroup, error) {
	if m.GetProvenanceByRunFn != nil {
		return m.GetProvenanceByRunFn(ctx, projectRoot)
	}
	return nil, nil
}

var _ sandbox.ServiceAPI = (*FakeService)(nil)
