package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/replay"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/proto"
)

// GetExecutionScreenshots handles GET /api/v1/executions/{id}/screenshots
func (h *Handler) GetExecutionScreenshots(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	screenshots, err := h.executionService.GetExecutionScreenshots(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_screenshots", "execution_id": executionID.String()}))
		return
	}

	h.respondProto(w, http.StatusOK, &basexecution.GetScreenshotsResponse{
		ExecutionId: executionID.String(),
		Screenshots: screenshots,
		Total:       int32(len(screenshots)),
	})
}

// GetExecutionTimeline handles GET /api/v1/executions/{id}/timeline
func (h *Handler) GetExecutionTimeline(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	// Preferred: proto timeline persisted to disk.
	if pbTimeline, err := h.executionService.GetExecutionTimelineProto(ctx, executionID); err == nil && pbTimeline != nil {
		h.respondProto(w, http.StatusOK, pbTimeline)
		return
	}

	// Fallback: legacy result.json timeline conversion.
	timeline, err := h.executionService.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution timeline")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_timeline", "execution_id": executionID.String()}))
		return
	}
	pbTimeline, err := protoconv.TimelineToProto(timeline)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to convert execution timeline to proto")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"operation": "timeline_to_proto",
			"error":     err.Error(),
		}))
		return
	}
	h.respondProto(w, http.StatusOK, pbTimeline)
}

// PostExecutionExport handles POST /api/v1/executions/{id}/export
func (h *Handler) PostExecutionExport(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	var body executionExportRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSONBodyAllowEmpty(w, r, &body); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid json payload"}))
			return
		}
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if body.Format != "" {
		format = strings.ToLower(strings.TrimSpace(body.Format))
	}
	if format == "" {
		format = "json"
	}

	renderSource, ok := normalizeRenderSource(body.RenderSource)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "unsupported render_source"}))
		return
	}
	if renderSource == renderSourceReplayFrames && format == "webm" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "webm format is only available for recorded video exports"}))
		return
	}

	if format == "folder" {
		outputDir := strings.TrimSpace(r.URL.Query().Get("output_dir"))
		if body.OutputDir != "" {
			outputDir = strings.TrimSpace(body.OutputDir)
		}

		if outputDir == "" {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "output_dir is required for folder format"}))
			return
		}

		exportCtx, cancelExport := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
		defer cancelExport()

		if err := h.executionService.ExportToFolder(exportCtx, executionID, outputDir, h.storage); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
				return
			}
			h.log.WithError(err).WithField("execution_id", executionID).WithField("output_dir", outputDir).Error("Failed to export to folder")
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "export_folder", "error": err.Error()}))
			return
		}

		h.respondSuccess(w, http.StatusOK, map[string]string{
			"message":    "Execution exported successfully",
			"output_dir": outputDir,
		})
		return
	}

	if format != "json" && format != "html" && renderSource != renderSourceReplayFrames {
		videoCtx, cancelVideo := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
		defer cancelVideo()

		recordedVideo, videoErr := h.loadRecordedVideo(videoCtx, executionID)
		if videoErr == nil && recordedVideo != nil {
			if err := h.serveRecordedVideo(w, r, recordedVideo, format, body.FileName); err != nil {
				h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to serve recorded video export")
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "recorded_video_export", "error": err.Error()}))
			}
			return
		}
		if renderSource == renderSourceRecordedVideo {
			if errors.Is(videoErr, database.ErrNotFound) {
				h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
				return
			}
			if errors.Is(videoErr, errRecordedVideoNotFound) {
				h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "recorded video unavailable for execution"}))
				return
			}
			if videoErr != nil {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "recorded_video_export", "error": videoErr.Error()}))
				return
			}
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "recorded video unavailable for execution"}))
			return
		}
		if videoErr != nil && !errors.Is(videoErr, errRecordedVideoNotFound) {
			h.log.WithError(videoErr).WithField("execution_id", executionID).Warn("Failed to load recorded video; falling back to replay render")
		}
	}

	previewCtx, cancelPreview := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancelPreview()

	preview, svcErr := h.executionService.DescribeExecutionExport(previewCtx, executionID)
	if svcErr != nil {
		if errors.Is(svcErr, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.log.WithError(svcErr).WithField("execution_id", executionID).Error("Failed to describe execution export")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "describe_export"}))
		return
	}

	replayConfig, configErr := h.loadReplayConfig(previewCtx)
	if configErr != nil && h.log != nil {
		h.log.WithError(configErr).Warn("Failed to load replay config for export")
	}
	replayOverrides := replayConfigToOverrides(replayConfig)
	spec, specErr := buildExportSpec(preview.Package, body.MovieSpec, executionID)
	if specErr != nil {
		if errors.Is(specErr, errMovieSpecUnavailable) {
			if format == "json" {
				applyReplayConfigToSpec(preview.Package, replayConfig)
				applyExportOverrides(preview.Package, replayOverrides)
				applyExportOverrides(preview.Package, body.Overrides)
				if pbPreview, err := protoconv.ExecutionExportPreviewToProto(preview); err == nil {
					h.respondProto(w, http.StatusOK, pbPreview)
				} else {
					h.respondSuccess(w, http.StatusOK, preview)
				}
			} else {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"error": "export package unavailable"}))
			}
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": specErr.Error()}))
		return
	}

	preview.Package = spec

	if format == "json" {
		applyReplayConfigToSpec(spec, replayConfig)
		applyExportOverrides(spec, replayOverrides)
		applyExportOverrides(spec, body.Overrides)
		if pbPreview, err := protoconv.ExecutionExportPreviewToProto(preview); err == nil {
			h.respondProto(w, http.StatusOK, pbPreview)
		} else {
			h.respondSuccess(w, http.StatusOK, preview)
		}
		return
	}

	if format == "html" {
		applyReplayConfigToSpec(spec, replayConfig)
		applyExportOverrides(spec, replayOverrides)
		applyExportOverrides(spec, body.Overrides)

		filename := normalizeExportFilename(body.FileName, "replay-export", ".zip")
		w.Header().Set("Content-Type", "application/zip")
		if strings.TrimSpace(filename) != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}

		baseURL := requestBaseURL(r)
		if err := exportservices.WriteHTMLBundle(previewCtx, w, spec, h.storage, h.log, baseURL); err != nil {
			h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to build HTML replay export")
		}
		return
	}

	applyReplayConfigToSpec(spec, replayConfig)
	applyExportOverrides(spec, replayOverrides)
	applyExportOverrides(spec, body.Overrides)

	renderTimeout := replay.EstimateReplayRenderTimeout(spec)
	renderCtx, cancelRender := context.WithTimeout(r.Context(), renderTimeout)
	defer cancelRender()

	media, renderErr := h.replayRenderer.Render(renderCtx, spec, replay.RenderFormat(format), body.FileName)
	if renderErr != nil {
		errMsg := strings.TrimSpace(renderErr.Error())
		if len(errMsg) > 0 && len(errMsg) > 512 {
			errMsg = errMsg[:512]
		}
		fields := logrus.Fields{"execution_id": executionID}
		if errMsg != "" {
			fields["renderer_error"] = errMsg
		}
		h.log.WithError(renderErr).WithFields(fields).Error("Failed to render replay export")
		details := map[string]string{"operation": "render_export"}
		if errMsg != "" {
			details["error"] = errMsg
		}
		h.respondError(w, ErrInternalServer.WithDetails(details))
		return
	}
	defer media.Cleanup()

	file, err := os.Open(media.Path)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "open_export"}))
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stat_export"}))
		return
	}

	w.Header().Set("Content-Type", media.ContentType)
	if info.Size() > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	}
	if strings.TrimSpace(media.Filename) != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", media.Filename))
	}

	http.ServeContent(w, r, media.Filename, info.ModTime(), file)
}

func (h *Handler) loadRecordedVideo(ctx context.Context, executionID uuid.UUID) (*recordedVideoSource, error) {
	if h.repo == nil {
		return nil, errRecordedVideoNotFound
	}
	execution, err := h.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(execution.ResultPath) == "" {
		return nil, errRecordedVideoNotFound
	}
	raw, err := os.ReadFile(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errRecordedVideoNotFound
		}
		return nil, err
	}
	var result executionwriter.ExecutionResultData
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return resolveRecordedVideoSource(result.Artifacts)
}

func (h *Handler) serveRecordedVideo(w http.ResponseWriter, r *http.Request, source *recordedVideoSource, format, fileName string) error {
	if source == nil {
		return errors.New("recorded video source missing")
	}
	if cleanup := source.Cleanup; cleanup != nil {
		defer cleanup()
	}

	inputPath := source.Path
	inputExt := strings.ToLower(filepath.Ext(inputPath))
	outputPath := inputPath
	outputType := source.ContentType
	outputExt := inputExt
	var outputCleanup func()

	defer func() {
		if outputCleanup != nil {
			outputCleanup()
		}
	}()

	switch format {
	case "webm":
		if inputExt != ".webm" {
			return fmt.Errorf("recorded video is not webm")
		}
		outputType = "video/webm"
		outputExt = ".webm"
	case "mp4":
		outputType = "video/mp4"
		outputExt = ".mp4"
		if inputExt != ".mp4" {
			tmp, err := os.CreateTemp("", "bas-recorded-video-*.mp4")
			if err != nil {
				return err
			}
			_ = tmp.Close()
			outputPath = tmp.Name()
			outputCleanup = func() { _ = os.Remove(outputPath) }
			encoder := replay.NewFFmpegEncoder(replay.DetectFFmpegBinary())
			ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
			defer cancel()
			if err := encoder.ConvertToMP4(ctx, inputPath, outputPath); err != nil {
				return err
			}
		}
	case "gif":
		outputType = "image/gif"
		outputExt = ".gif"
		tmp, err := os.CreateTemp("", "bas-recorded-video-*.gif")
		if err != nil {
			return err
		}
		_ = tmp.Close()
		outputPath = tmp.Name()
		outputCleanup = func() { _ = os.Remove(outputPath) }
		encoder := replay.NewFFmpegEncoder(replay.DetectFFmpegBinary())
		ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
		defer cancel()
		if err := encoder.ConvertToGIF(ctx, inputPath, outputPath, 0, 0); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported export format %q", format)
	}

	if strings.TrimSpace(outputType) == "" {
		outputType = detectVideoContentType(outputPath)
	}
	filename := normalizeExportFilename(fileName, "recorded-video", outputExt)

	file, err := os.Open(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", outputType)
	if info.Size() > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	}
	if strings.TrimSpace(filename) != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	http.ServeContent(w, r, filename, info.ModTime(), file)
	return nil
}

// GetExecution handles GET /api/v1/executions/{id}
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	execution, err := h.executionService.GetExecution(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": id.String()}))
		return
	}

	pbExecution, err := h.executionService.HydrateExecutionProto(ctx, execution)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).WithField("execution_id", id).Error("Failed to hydrate execution proto")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_execution"}))
		return
	}

	h.respondProto(w, http.StatusOK, pbExecution)
}

// ListExecutions handles GET /api/v1/executions
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	workflowIDStr := r.URL.Query().Get("workflow_id")
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		if id, err := uuid.Parse(workflowIDStr); err == nil {
			workflowID = &id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 0, 0)

	executions, err := h.executionService.ListExecutions(ctx, workflowID, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list executions")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_executions"}))
		return
	}

	protoExecutions := make([]proto.Message, 0, len(executions))
	for idx := range executions {
		if executions[idx] == nil {
			continue
		}
		pbExec, convErr := h.executionService.HydrateExecutionProto(ctx, executions[idx])
		if convErr != nil {
			if h.log != nil {
				h.log.WithError(convErr).WithField("execution_id", executions[idx].ID).Error("Failed to hydrate execution proto")
			}
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"operation": fmt.Sprintf("hydrate_execution[%d]", idx),
				"error":     convErr.Error(),
			}))
			return
		}
		protoExecutions = append(protoExecutions, pbExec)
	}

	h.respondProtoList(w, http.StatusOK, "executions", protoExecutions)
}

// StopExecution handles POST /api/v1/executions/{id}/stop
func (h *Handler) StopExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.executionService.StopExecution(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to stop execution")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stop_execution", "execution_id": id.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "stopped",
	})
}

// resumeExecutionRequest represents the request body for resuming an execution.
type resumeExecutionRequest struct {
	Parameters map[string]any `json:"parameters,omitempty"`
}

// ResumeExecution handles POST /api/v1/executions/{id}/resume
// Resumes an interrupted execution from its last checkpoint.
func (h *Handler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	var body resumeExecutionRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSONBodyAllowEmpty(w, r, &body); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid json payload"}))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	newExecution, err := h.executionService.ResumeExecution(ctx, id, body.Parameters)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to resume execution")
		// Check if it's a "cannot be resumed" error vs a server error
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot be resumed") || strings.Contains(errMsg, "not resumable") {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"operation":    "resume_execution",
				"execution_id": id.String(),
				"error":        errMsg,
			}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"operation":    "resume_execution",
			"execution_id": id.String(),
		}))
		return
	}

	pbExecution, err := protoconv.ExecutionToProto(newExecution)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).WithField("execution_id", newExecution.ID).Error("Failed to convert resumed execution to proto")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"operation": "execution_to_proto",
			"error":     err.Error(),
		}))
		return
	}

	h.respondProto(w, http.StatusOK, pbExecution)
}
