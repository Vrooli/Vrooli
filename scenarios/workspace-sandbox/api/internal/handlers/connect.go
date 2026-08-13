package handlers

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace/workspaceconnect"

	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// ConnectHandler exposes the existing sandbox service through the typed
// workspace-sandbox contract. Domain behavior remains in ServiceAPI; this
// layer only translates wire messages and stable transport errors.
type ConnectHandler struct {
	Service sandbox.ServiceAPI
}

func NewConnectHandler(service sandbox.ServiceAPI) *ConnectHandler {
	return &ConnectHandler{Service: service}
}

var _ workspaceconnect.WorkspaceSandboxServiceHandler = (*ConnectHandler)(nil)

func (h *ConnectHandler) ResolveWorkspace(ctx context.Context, req *connect.Request[workspacev1.ResolveWorkspaceRequest]) (*connect.Response[workspacev1.ResolveWorkspaceResponse], error) {
	id, err := parseSandboxID(req.Msg.GetSandboxId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	sandbox, err := h.Service.Get(ctx, id)
	if err != nil {
		return nil, domainConnectError(err)
	}
	root, err := h.Service.GetWorkspacePath(ctx, id)
	if err != nil {
		return nil, domainConnectError(err)
	}
	return connect.NewResponse(&workspacev1.ResolveWorkspaceResponse{
		Success:       true,
		SandboxId:     id.String(),
		WorkspaceRoot: root,
		IsolationMode: isolationMode(sandbox),
	}), nil
}

func (h *ConnectHandler) CreateSandbox(ctx context.Context, req *connect.Request[workspacev1.CreateSandboxRequest]) (*connect.Response[workspacev1.CreateSandboxResponse], error) {
	if req.Msg.GetScopePath() == "" && req.Msg.GetProjectRoot() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scope_path or project_root is required"))
	}
	create := &types.CreateRequest{
		Name:           req.Msg.GetName(),
		ScopePath:      req.Msg.GetScopePath(),
		ProjectRoot:    req.Msg.GetProjectRoot(),
		Owner:          req.Msg.GetOwner(),
		ReservedPaths:  append([]string(nil), req.Msg.GetReservedPaths()...),
		IdempotencyKey: req.Msg.GetIdempotencyKey(),
	}
	if len(create.ReservedPaths) > 0 {
		create.ReservedPath = create.ReservedPaths[0]
	}
	sandbox, err := h.Service.Create(ctx, create)
	if err != nil {
		return nil, domainConnectError(err)
	}
	return connect.NewResponse(&workspacev1.CreateSandboxResponse{
		Success: true,
		Sandbox: summary(sandbox),
	}), nil
}

func (h *ConnectHandler) GetSandboxDiff(ctx context.Context, req *connect.Request[workspacev1.GetSandboxDiffRequest]) (*connect.Response[workspacev1.GetSandboxDiffResponse], error) {
	id, err := parseSandboxID(req.Msg.GetSandboxId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	diff, err := h.Service.GetDiff(ctx, id)
	if err != nil {
		return nil, domainConnectError(err)
	}
	files := make([]*workspacev1.DiffFile, 0, len(diff.Files))
	for _, file := range diff.Files {
		if file == nil {
			continue
		}
		files = append(files, &workspacev1.DiffFile{
			Path:           file.FilePath,
			ChangeType:     string(file.ChangeType),
			Size:           file.FileSize,
			ApprovalStatus: string(file.ApprovalStatus),
		})
	}
	return connect.NewResponse(&workspacev1.GetSandboxDiffResponse{
		Success:      true,
		SandboxId:    id.String(),
		Files:        files,
		UnifiedDiff:  diff.UnifiedDiff,
		Stats:        diffStats(diff.Stats),
		ArchiveState: string(diff.ArchiveState),
	}), nil
}

func (h *ConnectHandler) PromoteSandbox(ctx context.Context, req *connect.Request[workspacev1.PromoteSandboxRequest]) (*connect.Response[workspacev1.PromoteSandboxResponse], error) {
	if !req.Msg.GetConfirm() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("confirm is required to promote sandbox changes"))
	}
	id, err := parseSandboxID(req.Msg.GetSandboxId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	mode := req.Msg.GetMode()
	if mode == "" {
		mode = "all"
	}
	result, err := h.Service.Approve(ctx, &types.ApprovalRequest{
		SandboxID:          id,
		Mode:               mode,
		Actor:              req.Msg.GetActor(),
		CommitMsg:          req.Msg.GetCommitMessage(),
		CreateCommit:       req.Msg.GetCreateCommit(),
		Force:              req.Msg.GetForce(),
		OverrideAcceptance: req.Msg.GetOverrideAcceptance(),
		Source:             types.SourceCLI,
	})
	if err != nil {
		return nil, domainConnectError(err)
	}
	return connect.NewResponse(&workspacev1.PromoteSandboxResponse{
		Success:    result.Success,
		SandboxId:  id.String(),
		Applied:    int32(result.Applied),
		Failed:     int32(result.Failed),
		Remaining:  int32(result.Remaining),
		IsPartial:  result.IsPartial,
		CommitHash: result.CommitHash,
		Error:      result.ErrorMsg,
		DiffPath:   result.DiffPath,
	}), nil
}

func parseSandboxID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("sandbox_id must be a UUID: %w", err)
	}
	return id, nil
}

func summary(sandbox *types.Sandbox) *workspacev1.SandboxSummary {
	if sandbox == nil {
		return nil
	}
	return &workspacev1.SandboxSummary{
		SandboxId:     sandbox.ID.String(),
		Status:        string(sandbox.Status),
		WorkspaceRoot: sandbox.MergedDir,
		IsolationMode: isolationMode(sandbox),
		ScopePath:     sandbox.ScopePath,
		ProjectRoot:   sandbox.ProjectRoot,
	}
}

func isolationMode(sandbox *types.Sandbox) string {
	if sandbox == nil {
		return ""
	}
	if sandbox.Containment != nil && sandbox.Containment.Level != "" {
		return sandbox.Containment.Level
	}
	return sandbox.DriverID
}

func diffStats(stats types.DiffStats) *workspacev1.DiffStats {
	return &workspacev1.DiffStats{
		FilesChanged:  int32(stats.FilesChanged),
		FilesAdded:    int32(stats.FilesAdded),
		FilesModified: int32(stats.FilesModified),
		FilesDeleted:  int32(stats.FilesDeleted),
		LinesAdded:    int32(stats.LinesAdded),
		LinesRemoved:  int32(stats.LinesRemoved),
		TotalBytes:    stats.TotalBytes,
	}
}

func domainConnectError(err error) error {
	var domainErr types.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.HTTPStatus() {
		case 400:
			return connect.NewError(connect.CodeInvalidArgument, err)
		case 404:
			return connect.NewError(connect.CodeNotFound, err)
		case 409:
			return connect.NewError(connect.CodeAlreadyExists, err)
		}
	}
	return connect.NewError(connect.CodeFailedPrecondition, err)
}
