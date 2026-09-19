package exports_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/ai"
	"github.com/vrooli/browser-automation-studio/services/credits"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

var (
	validFormats  = map[string]struct{}{"mp4": {}, "gif": {}, "json": {}, "html": {}}
	validStatuses = map[string]struct{}{"pending": {}, "processing": {}, "completed": {}, "failed": {}}
)

type service struct {
	deps Deps
}

// =============================================================================
// ListExports
// =============================================================================

func (s *service) ListExports(
	ctx context.Context,
	req *connect.Request[exportsv1.ListExportsRequest],
) (*connect.Response[exportsv1.ListExportsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	msg := req.Msg
	limit := int(msg.GetLimit())
	offset := int(msg.GetOffset())

	if execID := strings.TrimSpace(msg.GetExecutionId()); execID != "" {
		executionID, err := uuid.Parse(execID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExecutionID)
		}
		rows, err := s.deps.Repo.ListExportsByExecution(ctx, executionID)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("execution_id", executionID).Error("Failed to list exports by execution")
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&exportsv1.ListExportsResponse{
			Exports: exportsToProto(rows),
			Total:   int32(len(rows)),
		}), nil
	}

	if wfID := strings.TrimSpace(msg.GetWorkflowId()); wfID != "" {
		workflowID, err := uuid.Parse(wfID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
		}
		rows, err := s.deps.Repo.ListExportsByWorkflow(ctx, workflowID, limit, offset)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("workflow_id", workflowID).Error("Failed to list exports by workflow")
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&exportsv1.ListExportsResponse{
			Exports: exportsToProto(rows),
			Total:   int32(len(rows)),
		}), nil
	}

	rows, err := s.deps.Repo.ListExports(ctx, limit, offset)
	if err != nil {
		s.deps.Logger.WithError(err).Error("Failed to list exports")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&exportsv1.ListExportsResponse{
		Exports: exportsToProto(rows),
		Total:   int32(len(rows)),
	}), nil
}

// =============================================================================
// GetExport
// =============================================================================

func (s *service) GetExport(
	ctx context.Context,
	req *connect.Request[exportsv1.GetExportRequest],
) (*connect.Response[exportsv1.GetExportResponse], error) {
	exportID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&exportsv1.GetExportResponse{Export: exportToProto(row)}), nil
}

// =============================================================================
// CreateExport
// =============================================================================

func (s *service) CreateExport(
	ctx context.Context,
	req *connect.Request[exportsv1.CreateExportRequest],
) (*connect.Response[exportsv1.CreateExportResponse], error) {
	msg := req.Msg

	executionID, err := uuid.Parse(strings.TrimSpace(msg.GetExecutionId()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExecutionID)
	}

	name := strings.TrimSpace(msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNameRequired)
	}

	format := strings.ToLower(strings.TrimSpace(msg.GetFormat()))
	if format == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errFormatRequired)
	}
	if _, ok := validFormats[format]; !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidFormat)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	// Verify execution exists.
	if _, err := s.deps.Executor.GetExecution(ctx, executionID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExecutionNotFound)
		}
		s.deps.Logger.WithError(err).WithField("execution_id", executionID).Error("Failed to verify execution")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	row := &database.ExportIndex{
		ExecutionID:  executionID,
		Name:         name,
		Format:       format,
		Settings:     database.JSONMap(structToMap(msg.GetSettings())),
		StorageURL:   strings.TrimSpace(msg.GetStorageUrl()),
		ThumbnailURL: strings.TrimSpace(msg.GetThumbnailUrl()),
		Status:       "completed",
	}
	if msg.FileSizeBytes != nil {
		v := msg.GetFileSizeBytes()
		row.FileSizeBytes = &v
	}
	if msg.DurationMs != nil {
		v := int(msg.GetDurationMs())
		row.DurationMs = &v
	}
	if msg.FrameCount != nil {
		v := int(msg.GetFrameCount())
		row.FrameCount = &v
	}

	if wfStr := strings.TrimSpace(msg.GetWorkflowId()); wfStr != "" {
		workflowID, err := uuid.Parse(wfStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
		}
		row.WorkflowID = &workflowID
	}

	if statusStr := strings.TrimSpace(msg.GetStatus()); statusStr != "" {
		if _, ok := validStatuses[statusStr]; !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidStatus)
		}
		row.Status = statusStr
	}

	if err := s.deps.Repo.CreateExport(ctx, row); err != nil {
		s.deps.Logger.WithError(err).Error("Failed to create export")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&exportsv1.CreateExportResponse{
		ExportId: row.ID.String(),
		Status:   "created",
		Export:   exportToProto(row),
	}), nil
}

// =============================================================================
// UpdateExport
// =============================================================================

func (s *service) UpdateExport(
	ctx context.Context,
	req *connect.Request[exportsv1.UpdateExportRequest],
) (*connect.Response[exportsv1.UpdateExportResponse], error) {
	msg := req.Msg
	exportID, err := uuid.Parse(msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export for update")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if name := strings.TrimSpace(msg.GetName()); name != "" {
		row.Name = name
	}
	if cfg := msg.GetSettings(); cfg != nil {
		row.Settings = database.JSONMap(structToMap(cfg))
	}
	if storage := strings.TrimSpace(msg.GetStorageUrl()); storage != "" {
		row.StorageURL = storage
	}
	if thumb := strings.TrimSpace(msg.GetThumbnailUrl()); thumb != "" {
		row.ThumbnailURL = thumb
	}
	if msg.FileSizeBytes != nil {
		v := msg.GetFileSizeBytes()
		row.FileSizeBytes = &v
	}
	if msg.DurationMs != nil {
		v := int(msg.GetDurationMs())
		row.DurationMs = &v
	}
	if msg.FrameCount != nil {
		v := int(msg.GetFrameCount())
		row.FrameCount = &v
	}
	if caption := strings.TrimSpace(msg.GetAiCaption()); caption != "" {
		row.AICaption = caption
	}
	if statusStr := strings.TrimSpace(msg.GetStatus()); statusStr != "" {
		if _, ok := validStatuses[statusStr]; !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidStatus)
		}
		row.Status = statusStr
	}
	if errStr := msg.GetError(); errStr != "" {
		row.Error = errStr
	}

	if err := s.deps.Repo.UpdateExport(ctx, row); err != nil {
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to update export")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&exportsv1.UpdateExportResponse{
		ExportId: row.ID.String(),
		Status:   "updated",
		Export:   exportToProto(row),
	}), nil
}

// =============================================================================
// DeleteExport
// =============================================================================

func (s *service) DeleteExport(
	ctx context.Context,
	req *connect.Request[exportsv1.DeleteExportRequest],
) (*connect.Response[exportsv1.DeleteExportResponse], error) {
	exportID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export for deletion")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.deps.Repo.DeleteExport(ctx, exportID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to delete export")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Storage cleanup is intentionally deferred to TTL rules / periodic
	// jobs; see legacy comment.
	if row.StorageURL != "" {
		s.deps.Logger.WithFields(map[string]any{
			"export_id":   exportID,
			"storage_url": row.StorageURL,
		}).Debug("Export deleted; storage file cleanup delegated to external process")
	}

	return connect.NewResponse(&exportsv1.DeleteExportResponse{
		ExportId: exportID.String(),
		Status:   "deleted",
	}), nil
}

// =============================================================================
// GetExportStatus
// =============================================================================

func (s *service) GetExportStatus(
	ctx context.Context,
	req *connect.Request[exportsv1.GetExportStatusRequest],
) (*connect.Response[exportsv1.GetExportStatusResponse], error) {
	exportID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export status")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &exportsv1.GetExportStatusResponse{
		ExportId:    row.ID.String(),
		ExecutionId: row.ExecutionID.String(),
		Status:      row.Status,
		Format:      row.Format,
		Name:        row.Name,
		StorageUrl:  row.StorageURL,
		Error:       row.Error,
	}
	if row.FileSizeBytes != nil && *row.FileSizeBytes > 0 {
		v := *row.FileSizeBytes
		resp.FileSizeBytes = &v
	}
	return connect.NewResponse(resp), nil
}

// =============================================================================
// GenerateExportCaption
// =============================================================================

func (s *service) GenerateExportCaption(
	ctx context.Context,
	req *connect.Request[exportsv1.GenerateExportCaptionRequest],
) (*connect.Response[exportsv1.GenerateExportCaptionResponse], error) {
	exportID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export for caption generation")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var workflowName, workflowDescription string
	if row.WorkflowID != nil {
		if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, *row.WorkflowID); wfErr == nil && wf != nil {
			workflowName = wf.GetName()
			workflowDescription = wf.GetDescription()
		}
	}

	prompt := buildCaptionPrompt(row, workflowName, workflowDescription)

	caption, aiErr := s.executePrompt(ctx, prompt)
	formatLabel := captionFormatLabel(row.Format)
	if aiErr != nil {
		s.deps.Logger.WithError(aiErr).WithField("export_id", exportID).Error("Failed to generate AI caption")
		// Fallback caption rather than erroring.
		caption = fmt.Sprintf("Check out this %s replay: %s", formatLabel, row.Name)
	}

	caption = strings.TrimSpace(caption)
	caption = strings.Trim(caption, "\"'")

	now := time.Now()
	row.AICaption = caption
	row.AICaptionGeneratedAt = &now
	if updateErr := s.deps.Repo.UpdateExport(ctx, row); updateErr != nil {
		s.deps.Logger.WithError(updateErr).WithField("export_id", exportID).Error("Failed to save generated caption")
		// Still return the caption even if save fails.
	}

	return connect.NewResponse(&exportsv1.GenerateExportCaptionResponse{
		ExportId: exportID.String(),
		Caption:  caption,
		Export:   exportToProto(row),
	}), nil
}

// =============================================================================
// RevealExport
// =============================================================================

func (s *service) RevealExport(
	ctx context.Context,
	req *connect.Request[exportsv1.RevealExportRequest],
) (*connect.Response[exportsv1.RevealExportResponse], error) {
	exportID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export for reveal")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if row.StorageURL == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoStoragePath)
	}

	if err := s.deps.Opener.Reveal(row.StorageURL); err != nil {
		s.deps.Logger.WithError(err).WithFields(map[string]any{
			"export_id": exportID,
			"path":      row.StorageURL,
		}).Error("Failed to reveal export in file manager")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to open file manager: %w", err))
	}

	return connect.NewResponse(&exportsv1.RevealExportResponse{
		ExportId: exportID.String(),
		Path:     row.StorageURL,
		Status:   "revealed",
	}), nil
}

// =============================================================================
// OpenExportFolder
// =============================================================================

func (s *service) OpenExportFolder(
	ctx context.Context,
	req *connect.Request[exportsv1.OpenExportFolderRequest],
) (*connect.Response[exportsv1.OpenExportFolderResponse], error) {
	exportID, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidExportID)
	}

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	row, err := s.deps.Repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExportNotFound)
		}
		s.deps.Logger.WithError(err).WithField("export_id", exportID).Error("Failed to get export for open-folder")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if row.StorageURL == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoStoragePath)
	}

	folder := filepath.Dir(row.StorageURL)
	if err := s.deps.Opener.OpenFolder(folder); err != nil {
		s.deps.Logger.WithError(err).WithFields(map[string]any{
			"export_id": exportID,
			"path":      folder,
		}).Error("Failed to open folder in file manager")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to open folder: %w", err))
	}

	return connect.NewResponse(&exportsv1.OpenExportFolderResponse{
		ExportId: exportID.String(),
		Folder:   folder,
		Status:   "opened",
	}), nil
}

// =============================================================================
// Helpers
// =============================================================================

func (s *service) executePrompt(ctx context.Context, prompt string) (string, error) {
	if s.deps.AIClientFactory != nil {
		userIdentity := entitlement.UserIdentityFromContext(ctx)
		byokKey := middleware.BYOKKeyFromContext(ctx)
		client := s.deps.AIClientFactory.CreateClient(ai.ClientOptions{
			UserIdentity:  userIdentity,
			BYOKKey:       byokKey,
			OperationType: credits.OpAICaptionGenerate,
		})
		return client.ExecutePrompt(ctx, prompt)
	}
	// Legacy direct OpenRouter path (no credit tracking).
	client := ai.NewOpenRouterClient(s.deps.Logger)
	return client.ExecutePrompt(ctx, prompt)
}

func captionFormatLabel(format string) string {
	labels := map[string]string{
		"mp4":  "video",
		"gif":  "animated GIF",
		"json": "JSON export package",
		"html": "interactive HTML replay",
	}
	if v, ok := labels[format]; ok {
		return v
	}
	return format
}

func buildCaptionPrompt(row *database.ExportIndex, workflowName, workflowDescription string) string {
	formatLabel := captionFormatLabel(row.Format)

	durationStr := ""
	if row.DurationMs != nil && *row.DurationMs > 0 {
		seconds := *row.DurationMs / 1000
		if seconds < 60 {
			durationStr = fmt.Sprintf("%d seconds", seconds)
		} else {
			minutes := seconds / 60
			remaining := seconds % 60
			durationStr = fmt.Sprintf("%d:%02d", minutes, remaining)
		}
	}

	descLine := ""
	if workflowDescription != "" {
		descLine = fmt.Sprintf("- Description: %s\n", workflowDescription)
	}
	durLine := ""
	if durationStr != "" {
		durLine = fmt.Sprintf("- Duration: %s\n", durationStr)
	}

	return fmt.Sprintf(`Generate a short, engaging caption for sharing this browser automation export on social media or in a presentation.

Export Details:
- Name: %s
- Format: %s
- Workflow: %s
%s%s

Requirements:
- Keep the caption concise (1-2 sentences, max 280 characters for Twitter compatibility)
- Make it engaging and professional
- Highlight what the automation does if clear from the name
- Don't use hashtags unless specifically requested
- Don't include emoji unless it adds value

Return ONLY the caption text, nothing else.`,
		row.Name, formatLabel, workflowName, descLine, durLine,
	)
}

// =============================================================================
// Proto conversions
// =============================================================================

func exportsToProto(in []*database.ExportIndex) []*exportsv1.Export {
	out := make([]*exportsv1.Export, 0, len(in))
	for _, e := range in {
		out = append(out, exportToProto(e))
	}
	return out
}

func exportToProto(e *database.ExportIndex) *exportsv1.Export {
	if e == nil {
		return nil
	}
	pb := &exportsv1.Export{
		Id:           e.ID.String(),
		ExecutionId:  e.ExecutionID.String(),
		Name:         e.Name,
		Format:       e.Format,
		Settings:     jsonMapToStruct(e.Settings),
		StorageUrl:   e.StorageURL,
		ThumbnailUrl: e.ThumbnailURL,
		AiCaption:    e.AICaption,
		Status:       e.Status,
		Error:        e.Error,
		CreatedAt:    timestamppb.New(e.CreatedAt),
		UpdatedAt:    timestamppb.New(e.UpdatedAt),
		WorkflowName: e.WorkflowName,
	}
	if e.WorkflowID != nil {
		pb.WorkflowId = e.WorkflowID.String()
	}
	if e.FileSizeBytes != nil {
		v := *e.FileSizeBytes
		pb.FileSizeBytes = &v
	}
	if e.DurationMs != nil {
		v := int32(*e.DurationMs)
		pb.DurationMs = &v
	}
	if e.FrameCount != nil {
		v := int32(*e.FrameCount)
		pb.FrameCount = &v
	}
	if e.AICaptionGeneratedAt != nil {
		pb.AiCaptionGeneratedAt = timestamppb.New(*e.AICaptionGeneratedAt)
	}
	if e.ExecutionDate != nil {
		pb.ExecutionDate = timestamppb.New(*e.ExecutionDate)
	}
	return pb
}

// jsonMapToStruct converts a JSONMap to *structpb.Struct, tolerating
// non-string-keyed nested maps by re-marshaling through JSON.
func jsonMapToStruct(m database.JSONMap) *structpb.Struct {
	if len(m) == 0 {
		return nil
	}
	raw, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return nil
	}
	pb, err := structpb.NewStruct(asMap)
	if err != nil {
		return nil
	}
	return pb
}

func structToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}
