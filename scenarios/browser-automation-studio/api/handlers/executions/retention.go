package executions

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/services/retention"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

var errRetentionUnavailable = errors.New("execution artifact retention is not available")

// PreviewExecutionArtifactRetention reports (dry-run) which terminal executions
// and artifact directories a retention sweep would remove. It performs no
// mutation.
func (s *service) PreviewExecutionArtifactRetention(
	ctx context.Context,
	req *connect.Request[basapi.ExecutionArtifactRetentionRequest],
) (*connect.Response[basapi.ExecutionArtifactRetentionResponse], error) {
	return s.runRetention(ctx, req.Msg, false)
}

// RunExecutionArtifactRetention deletes matched terminal execution rows and
// their artifact directories together. It requires confirm=true.
func (s *service) RunExecutionArtifactRetention(
	ctx context.Context,
	req *connect.Request[basapi.ExecutionArtifactRetentionRequest],
) (*connect.Response[basapi.ExecutionArtifactRetentionResponse], error) {
	if !req.Msg.GetConfirm() {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("confirm must be true to run execution artifact retention; use the preview RPC for a dry-run"))
	}
	return s.runRetention(ctx, req.Msg, true)
}

func (s *service) runRetention(
	ctx context.Context,
	msg *basapi.ExecutionArtifactRetentionRequest,
	apply bool,
) (*connect.Response[basapi.ExecutionArtifactRetentionResponse], error) {
	if s.deps.Retention == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errRetentionUnavailable)
	}

	opts := retention.Options{
		MaxAgeDays: int(msg.GetMaxAgeDays()),
		KeepLatest: int(msg.GetKeepLatest()),
		Apply:      apply,
	}

	if raw := strings.TrimSpace(msg.GetWorkflowId()); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workflow_id"))
		}
		opts.WorkflowID = &id
	}
	if raw := strings.TrimSpace(msg.GetProjectId()); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid project_id"))
		}
		opts.ProjectID = &id
	}
	if msg.Status != nil {
		status := enums.ExecutionStatusToString(msg.GetStatus())
		if !database.IsTerminalStatus(status) {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("status filter must be a terminal status (completed or failed)"))
		}
		opts.Status = status
	}

	report, err := s.deps.Retention.Sweep(ctx, opts)
	if err != nil {
		if errors.Is(err, retention.ErrRecordingsRootNotConfigured) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		s.log().WithError(err).Error("execution artifact retention sweep failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(retentionReportToProto(report)), nil
}

func retentionReportToProto(report *retention.Report) *basapi.ExecutionArtifactRetentionResponse {
	out := &basapi.ExecutionArtifactRetentionResponse{
		DryRun:          report.DryRun,
		Removed:         make([]*basapi.ExecutionRetentionItem, 0, len(report.Removed)),
		Skipped:         make([]*basapi.ExecutionRetentionItem, 0, len(report.Skipped)),
		EstimatedBytes:  report.EstimatedBytes,
		RemovedCount:    int32(report.RemovedCount),
		SkippedCount:    int32(report.SkippedCount),
		ErrorCount:      int32(report.ErrorCount),
		RemovedByStatus: make(map[string]int32, len(report.RemovedByStatus)),
	}
	for _, item := range report.Removed {
		out.Removed = append(out.Removed, retentionItemToProto(item))
	}
	for _, item := range report.Skipped {
		out.Skipped = append(out.Skipped, retentionItemToProto(item))
	}
	for status, count := range report.RemovedByStatus {
		out.RemovedByStatus[status] = int32(count)
	}
	return out
}

func retentionItemToProto(item retention.Item) *basapi.ExecutionRetentionItem {
	out := &basapi.ExecutionRetentionItem{
		ExecutionId:    item.ExecutionID.String(),
		Status:         enums.StringToExecutionStatus(item.Status),
		WorkflowId:     item.WorkflowID.String(),
		ResultPath:     item.ResultPath,
		ArtifactDir:    item.ArtifactDir,
		EstimatedBytes: item.EstimatedBytes,
		Reason:         item.Reason,
	}
	if !item.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(item.StartedAt)
	}
	if item.CompletedAt != nil && !item.CompletedAt.IsZero() {
		out.CompletedAt = timestamppb.New(*item.CompletedAt)
	}
	return out
}
