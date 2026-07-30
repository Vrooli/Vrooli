package harness

import (
	"context"
	"log"
	"os"
	"time"

	"connectrpc.com/connect"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
	internalharness "vrooli-memory/internal/harness"
)

type connectHandler struct {
	importer  *internalharness.Importer
	projector *internalharness.Projector
	logger    *log.Logger
}

func NewConnectHandler(i *internalharness.Importer, p *internalharness.Projector, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{importer: i, projector: p, logger: l}
}

func (h *connectHandler) RunImport(ctx context.Context, req *connect.Request[harnessv1.RunImportRequest]) (*connect.Response[harnessv1.RunImportResponse], error) {
	if req.Msg.GetDryRun() {
		result, err := h.importer.Import(ctx, req.Msg.GetRuntime(), true)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return connect.NewResponse(&harnessv1.RunImportResponse{ImportedCount: int32(result.Seen), DryRun: true}), nil
	}
	run, joined, err := h.importer.Start(ctx, req.Msg.GetRuntime())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&harnessv1.RunImportResponse{Run: importRunMessage(run), JoinedExistingRun: joined}), nil
}

func (h *connectHandler) GetImportStatus(ctx context.Context, req *connect.Request[harnessv1.GetImportStatusRequest]) (*connect.Response[harnessv1.GetImportStatusResponse], error) {
	run, err := h.importer.Status(ctx, req.Msg.GetRunId(), req.Msg.GetRuntime())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&harnessv1.GetImportStatusResponse{Run: importRunMessage(run)}), nil
}

func importRunMessage(run internalharness.ImportRun) *harnessv1.ImportRun {
	format := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339Nano)
	}
	return &harnessv1.ImportRun{Id: run.ID, Runtime: run.Runtime, SourceRoot: run.SourceRoot, Status: string(run.Status), TotalSources: int32(run.TotalSources), ProcessedSources: int32(run.ProcessedSources), ImportedCount: int32(run.ImportedCount), ExistingCount: int32(run.ExistingCount), FailedCount: int32(run.FailedCount), CurrentPath: run.CurrentPath, ErrorMessage: run.ErrorMessage, StartedAt: format(run.StartedAt), CompletedAt: format(run.CompletedAt), UpdatedAt: format(run.UpdatedAt)}
}

func (h *connectHandler) RefreshProjection(ctx context.Context, req *connect.Request[harnessv1.RefreshProjectionRequest]) (*connect.Response[harnessv1.RefreshProjectionResponse], error) {
	result, err := h.projector.Project(ctx, req.Msg.GetRuntime(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	content := ""
	if result.DryRun {
		content = result.Content
	}
	return connect.NewResponse(&harnessv1.RefreshProjectionResponse{Path: result.Path, SizeBytes: result.SizeBytes, Overflow: result.Overflow, DryRun: result.DryRun, RenderedContent: content}), nil
}

func (h *connectHandler) InstallPromptBlock(_ context.Context, req *connect.Request[harnessv1.InstallPromptBlockRequest]) (*connect.Response[harnessv1.InstallPromptBlockResponse], error) {
	path, err := internalharness.PromptTarget(req.Msg.GetRuntime(), os.Getenv("VROOLI_MEMORY_WORKSPACE_ROOT"))
	if err == nil {
		err = internalharness.InstallPromptBlock(path)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&harnessv1.InstallPromptBlockResponse{Installed: true}), nil
}

func (h *connectHandler) CaptureWrite(ctx context.Context, req *connect.Request[harnessv1.CaptureWriteRequest]) (*connect.Response[harnessv1.CaptureWriteResponse], error) {
	entry, err := h.importer.Capture(ctx, req.Msg.GetRuntime(), req.Msg.GetSourcePath(), req.Msg.GetContent())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&harnessv1.CaptureWriteResponse{EntryId: entry.ID}), nil
}
