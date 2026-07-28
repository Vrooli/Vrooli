package harness

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
	internalharness "vrooli-memory/internal/harness"
)

type connectHandler struct {
	importer *internalharness.Importer
	logger   *log.Logger
}

func NewConnectHandler(i *internalharness.Importer, l *log.Logger) *connectHandler {
	if l == nil {
		l = log.Default()
	}
	return &connectHandler{importer: i, logger: l}
}

func (h *connectHandler) RunImport(ctx context.Context, req *connect.Request[harnessv1.RunImportRequest]) (*connect.Response[harnessv1.RunImportResponse], error) {
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

func (h *connectHandler) RefreshProjection(context.Context, *connect.Request[harnessv1.RefreshProjectionRequest]) (*connect.Response[harnessv1.RefreshProjectionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnimplemented("projection"))
}

func (h *connectHandler) InstallPromptBlock(context.Context, *connect.Request[harnessv1.InstallPromptBlockRequest]) (*connect.Response[harnessv1.InstallPromptBlockResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnimplemented("prompt block"))
}

func (h *connectHandler) CaptureWrite(context.Context, *connect.Request[harnessv1.CaptureWriteRequest]) (*connect.Response[harnessv1.CaptureWriteResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errUnimplemented("capture"))
}

type errUnimplemented string

func (e errUnimplemented) Error() string { return string(e) + " is not implemented" }
