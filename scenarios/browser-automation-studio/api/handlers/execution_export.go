package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers/export"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/export/render"
	"github.com/vrooli/browser-automation-studio/services/export/source"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// PostExecutionExport handles POST /api/v1/executions/{id}/export
// For binary formats (mp4/gif) with output_dir: returns immediately with export ID, renders in background
// For json format: returns export spec synchronously
// For html format with output_dir: saves to disk asynchronously
func (h *Handler) PostExecutionExport(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	var body export.Request
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

	renderSource, ok := source.NormalizeRenderSource(body.RenderSource)
	if !ok {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "unsupported render_source"}))
		return
	}
	if renderSource == source.RenderSourceReplayFrames && format == "webm" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "webm format is only available for recorded video exports"}))
		return
	}

	// Get output_dir from query or body
	outputDir := strings.TrimSpace(r.URL.Query().Get("output_dir"))
	if body.OutputDir != "" {
		outputDir = strings.TrimSpace(body.OutputDir)
	}

	// Binary formats (mp4/gif) require output_dir for server-side saving
	isBinaryFormat := format == "mp4" || format == "gif" || format == "webm"
	if isBinaryFormat && outputDir == "" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "output_dir is required for binary exports (mp4/gif/webm)"}))
		return
	}

	if format == "folder" {
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

	// For binary formats with output_dir, use async rendering with WebSocket progress
	if isBinaryFormat && outputDir != "" {
		h.handleAsyncBinaryExport(w, r, executionID, format, body, outputDir, renderSource)
		return
	}

	// From here on: json/html format (synchronous) or recorded video fallback

	if format != "json" && format != "html" && renderSource != source.RenderSourceReplayFrames {
		videoCtx, cancelVideo := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
		defer cancelVideo()

		recordedVideo, videoErr := h.loadRecordedVideo(videoCtx, executionID)
		if videoErr == nil && recordedVideo != nil {
			// For recorded video with output_dir, save to disk instead of streaming
			if outputDir != "" {
				h.handleAsyncRecordedVideoExport(w, executionID, recordedVideo, format, body.FileName, outputDir)
				return
			}
			// Legacy: stream directly to response (for backwards compatibility)
			if err := h.serveRecordedVideo(w, r, recordedVideo, format, body.FileName); err != nil {
				h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to serve recorded video export")
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "recorded_video_export", "error": err.Error()}))
			}
			return
		}
		if renderSource == source.RenderSourceAuto && format == "webm" {
			if errors.Is(videoErr, database.ErrNotFound) {
				h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
				return
			}
			if errors.Is(videoErr, source.ErrVideoNotFound) {
				h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "recorded video unavailable for webm export"}))
				return
			}
			if videoErr != nil {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "recorded_video_export", "error": videoErr.Error()}))
				return
			}
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "recorded video unavailable for webm export"}))
			return
		}
		if renderSource == source.RenderSourceRecordedVideo {
			if errors.Is(videoErr, database.ErrNotFound) {
				h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
				return
			}
			if errors.Is(videoErr, source.ErrVideoNotFound) {
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
		if videoErr != nil && !errors.Is(videoErr, source.ErrVideoNotFound) {
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
	spec, specErr := export.BuildSpec(preview.Package, body.MovieSpec, executionID)
	if specErr != nil {
		if errors.Is(specErr, export.ErrMovieSpecUnavailable) {
			if format == "json" {
				applyReplayConfigToSpec(preview.Package, replayConfig)
				export.Apply(preview.Package, replayOverrides)
				export.Apply(preview.Package, body.Overrides)
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

	// Enforce watermark requirements based on user's subscription tier.
	// This ensures free/solo tier users always have the Vrooli watermark enabled.
	if h.entitlementService != nil {
		ent := entitlement.FromContext(r.Context())
		requiresWatermark := false
		if ent != nil {
			requiresWatermark = h.entitlementService.TierRequiresWatermark(ent.Tier)
		} else {
			// No entitlement in context - check via service (uses default tier)
			userIdentity := entitlement.UserIdentityFromContext(r.Context())
			requiresWatermark = h.entitlementService.RequiresWatermark(previewCtx, userIdentity)
		}
		if result := exportservices.EnforceWatermarkRequirements(spec, requiresWatermark); result.WasEnforced {
			h.log.WithFields(logrus.Fields{
				"execution_id":     executionID,
				"original_enabled": result.OriginalEnabled,
				"original_asset":   result.OriginalAssetID,
			}).Debug("Watermark requirements enforced for export")
		}
	}

	if format == "json" {
		applyReplayConfigToSpec(spec, replayConfig)
		export.Apply(spec, replayOverrides)
		export.Apply(spec, body.Overrides)
		if pbPreview, err := protoconv.ExecutionExportPreviewToProto(preview); err == nil {
			h.respondProto(w, http.StatusOK, pbPreview)
		} else {
			h.respondSuccess(w, http.StatusOK, preview)
		}
		return
	}

	if format == "html" {
		applyReplayConfigToSpec(spec, replayConfig)
		export.Apply(spec, replayOverrides)
		export.Apply(spec, body.Overrides)

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

	// Legacy fallback: sync render and stream to response
	applyReplayConfigToSpec(spec, replayConfig)
	export.Apply(spec, replayOverrides)
	export.Apply(spec, body.Overrides)

	renderTimeout := render.EstimateReplayRenderTimeout(spec)
	renderCtx, cancelRender := context.WithTimeout(r.Context(), renderTimeout)
	defer cancelRender()

	media, renderErr := h.replayRenderer.Render(renderCtx, spec, render.RenderFormat(format), body.FileName)
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

// handleAsyncBinaryExport creates an export record and renders in background with WebSocket progress.
func (h *Handler) handleAsyncBinaryExport(w http.ResponseWriter, r *http.Request, executionID uuid.UUID, format string, body export.Request, outputDir string, renderSource string) {
	ctx := r.Context()

	// Verify execution exists and get workflow info
	execution, err := h.executionService.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_execution"}))
		return
	}

	// Generate export name and filename
	exportName := strings.TrimSpace(body.FileName)
	if exportName == "" {
		exportName = fmt.Sprintf("export-%s", executionID.String()[:8])
	}
	// Remove extension if present
	exportName = strings.TrimSuffix(exportName, "."+format)

	// Create export record in "processing" state
	exportRecord := &database.ExportIndex{
		ID:          uuid.New(),
		ExecutionID: executionID,
		WorkflowID:  &execution.WorkflowID,
		Name:        exportName,
		Format:      format,
		Status:      "processing",
	}

	if err := h.repo.CreateExport(ctx, exportRecord); err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to create export record")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_export"}))
		return
	}

	exportID := exportRecord.ID.String()
	execIDStr := executionID.String()

	// Broadcast initial progress
	h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
		ExportID:        exportID,
		ExecutionID:     execIDStr,
		Stage:           "preparing",
		ProgressPercent: 0,
		Status:          "processing",
	})

	// Return immediately with export ID
	h.respondSuccess(w, http.StatusAccepted, map[string]any{
		"export_id":    exportID,
		"execution_id": execIDStr,
		"status":       "processing",
		"message":      "Export started. Subscribe to WebSocket for progress updates.",
	})

	// Start background rendering
	go h.renderExportInBackground(exportRecord, executionID, format, body, outputDir, renderSource)
}

// renderExportInBackground performs the actual export rendering and broadcasts progress via WebSocket.
func (h *Handler) renderExportInBackground(exportRecord *database.ExportIndex, executionID uuid.UUID, format string, body export.Request, outputDir string, renderSource string) {
	ctx := context.Background()
	exportID := exportRecord.ID.String()
	execIDStr := executionID.String()

	broadcastProgress := func(stage string, percent float64, status string) {
		h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
			ExportID:        exportID,
			ExecutionID:     execIDStr,
			Stage:           stage,
			ProgressPercent: percent,
			Status:          status,
		})
	}

	broadcastError := func(errMsg string) {
		_ = h.repo.UpdateExportStatus(ctx, exportRecord.ID, "failed", errMsg)
		h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
			ExportID:        exportID,
			ExecutionID:     execIDStr,
			Stage:           "failed",
			ProgressPercent: 0,
			Status:          "failed",
			Error:           errMsg,
		})
	}

	// Try to use recorded video first if not explicitly using replay frames
	if renderSource != source.RenderSourceReplayFrames {
		recordedVideo, videoErr := h.loadRecordedVideo(ctx, executionID)
		if videoErr == nil && recordedVideo != nil {
			h.renderRecordedVideoToFile(ctx, exportRecord, recordedVideo, format, outputDir, broadcastProgress, broadcastError)
			return
		}
	}

	// Fall back to replay rendering
	broadcastProgress("preparing", 10, "processing")

	preview, svcErr := h.executionService.DescribeExecutionExport(ctx, executionID)
	if svcErr != nil {
		broadcastError(fmt.Sprintf("Failed to describe export: %v", svcErr))
		return
	}

	replayConfig, _ := h.loadReplayConfig(ctx)
	replayOverrides := replayConfigToOverrides(replayConfig)
	spec, specErr := export.BuildSpec(preview.Package, body.MovieSpec, executionID)
	if specErr != nil {
		broadcastError(fmt.Sprintf("Failed to build export spec: %v", specErr))
		return
	}

	applyReplayConfigToSpec(spec, replayConfig)
	export.Apply(spec, replayOverrides)
	export.Apply(spec, body.Overrides)

	broadcastProgress("capturing", 30, "processing")

	// Render to temp file
	renderTimeout := render.EstimateReplayRenderTimeout(spec)
	renderCtx, cancelRender := context.WithTimeout(ctx, renderTimeout)
	defer cancelRender()

	media, renderErr := h.replayRenderer.Render(renderCtx, spec, render.RenderFormat(format), body.FileName)
	if renderErr != nil {
		broadcastError(fmt.Sprintf("Render failed: %v", renderErr))
		return
	}
	defer media.Cleanup()

	broadcastProgress("encoding", 70, "processing")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		broadcastError(fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	// Copy to output directory
	finalPath := filepath.Join(outputDir, media.Filename)
	if err := copyFile(media.Path, finalPath); err != nil {
		broadcastError(fmt.Sprintf("Failed to save file: %v", err))
		return
	}

	broadcastProgress("finalizing", 90, "processing")

	// Get file size
	info, err := os.Stat(finalPath)
	if err != nil {
		broadcastError(fmt.Sprintf("Failed to stat output file: %v", err))
		return
	}

	// Update export record
	if err := h.repo.UpdateExportComplete(ctx, exportRecord.ID, finalPath, info.Size()); err != nil {
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to update export record")
	}

	// Broadcast completion
	h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
		ExportID:        exportID,
		ExecutionID:     execIDStr,
		Stage:           "completed",
		ProgressPercent: 100,
		Status:          "completed",
		StorageURL:      finalPath,
		FileSizeBytes:   info.Size(),
	})

	h.log.WithFields(logrus.Fields{
		"export_id":    exportID,
		"execution_id": execIDStr,
		"output_path":  finalPath,
		"file_size":    info.Size(),
	}).Info("Export completed successfully")
}

// renderRecordedVideoToFile converts and saves a recorded video to the output directory.
func (h *Handler) renderRecordedVideoToFile(ctx context.Context, exportRecord *database.ExportIndex, videoSource *source.VideoSource, format, outputDir string, broadcastProgress func(string, float64, string), broadcastError func(string)) {
	exportID := exportRecord.ID.String()
	execIDStr := exportRecord.ExecutionID.String()

	broadcastProgress("preparing", 20, "processing")

	inputPath := videoSource.Path
	inputExt := strings.ToLower(filepath.Ext(inputPath))
	outputExt := "." + format

	// Generate output filename
	filename := exportRecord.Name + outputExt

	broadcastProgress("encoding", 50, "processing")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		broadcastError(fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	finalPath := filepath.Join(outputDir, filename)

	// Convert or copy depending on format
	switch format {
	case "webm":
		if inputExt != ".webm" {
			broadcastError("Source video is not webm format")
			return
		}
		if err := copyFile(inputPath, finalPath); err != nil {
			broadcastError(fmt.Sprintf("Failed to copy video: %v", err))
			return
		}
	case "mp4":
		if inputExt == ".mp4" {
			if err := copyFile(inputPath, finalPath); err != nil {
				broadcastError(fmt.Sprintf("Failed to copy video: %v", err))
				return
			}
		} else {
			encoder := render.NewFFmpegEncoder(render.DetectFFmpegBinary())
			if err := encoder.ConvertToMP4(ctx, inputPath, finalPath); err != nil {
				broadcastError(fmt.Sprintf("Failed to convert to MP4: %v", err))
				return
			}
		}
	case "gif":
		encoder := render.NewFFmpegEncoder(render.DetectFFmpegBinary())
		if err := encoder.ConvertToGIF(ctx, inputPath, finalPath, 0, 0); err != nil {
			broadcastError(fmt.Sprintf("Failed to convert to GIF: %v", err))
			return
		}
	default:
		broadcastError(fmt.Sprintf("Unsupported format: %s", format))
		return
	}

	broadcastProgress("finalizing", 90, "processing")

	// Get file size
	info, err := os.Stat(finalPath)
	if err != nil {
		broadcastError(fmt.Sprintf("Failed to stat output file: %v", err))
		return
	}

	// Update export record
	if err := h.repo.UpdateExportComplete(ctx, exportRecord.ID, finalPath, info.Size()); err != nil {
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to update export record")
	}

	// Broadcast completion
	h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
		ExportID:        exportID,
		ExecutionID:     execIDStr,
		Stage:           "completed",
		ProgressPercent: 100,
		Status:          "completed",
		StorageURL:      finalPath,
		FileSizeBytes:   info.Size(),
	})

	h.log.WithFields(logrus.Fields{
		"export_id":    exportID,
		"execution_id": execIDStr,
		"output_path":  finalPath,
		"file_size":    info.Size(),
	}).Info("Recorded video export completed successfully")
}

// handleAsyncRecordedVideoExport handles recorded video exports with output_dir.
func (h *Handler) handleAsyncRecordedVideoExport(w http.ResponseWriter, executionID uuid.UUID, videoSource *source.VideoSource, format, fileName, outputDir string) {
	ctx := context.Background()

	// Get execution to get workflow ID
	execution, err := h.executionService.GetExecution(ctx, executionID)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_execution"}))
		return
	}

	// Generate export name
	exportName := strings.TrimSpace(fileName)
	if exportName == "" {
		exportName = fmt.Sprintf("recording-%s", executionID.String()[:8])
	}
	exportName = strings.TrimSuffix(exportName, "."+format)

	// Create export record
	exportRecord := &database.ExportIndex{
		ID:          uuid.New(),
		ExecutionID: executionID,
		WorkflowID:  &execution.WorkflowID,
		Name:        exportName,
		Format:      format,
		Status:      "processing",
	}

	if err := h.repo.CreateExport(ctx, exportRecord); err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to create export record")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_export"}))
		return
	}

	exportID := exportRecord.ID.String()
	execIDStr := executionID.String()

	// Broadcast initial progress
	h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
		ExportID:        exportID,
		ExecutionID:     execIDStr,
		Stage:           "preparing",
		ProgressPercent: 0,
		Status:          "processing",
	})

	// Return immediately
	h.respondSuccess(w, http.StatusAccepted, map[string]any{
		"export_id":    exportID,
		"execution_id": execIDStr,
		"status":       "processing",
		"message":      "Export started. Subscribe to WebSocket for progress updates.",
	})

	// Start background conversion
	go func() {
		broadcastProgress := func(stage string, percent float64, status string) {
			h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
				ExportID:        exportID,
				ExecutionID:     execIDStr,
				Stage:           stage,
				ProgressPercent: percent,
				Status:          status,
			})
		}
		broadcastError := func(errMsg string) {
			_ = h.repo.UpdateExportStatus(context.Background(), exportRecord.ID, "failed", errMsg)
			h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{
				ExportID:        exportID,
				ExecutionID:     execIDStr,
				Stage:           "failed",
				ProgressPercent: 0,
				Status:          "failed",
				Error:           errMsg,
			})
		}
		h.renderRecordedVideoToFile(context.Background(), exportRecord, videoSource, format, outputDir, broadcastProgress, broadcastError)
	}()
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (h *Handler) loadRecordedVideo(ctx context.Context, executionID uuid.UUID) (*source.VideoSource, error) {
	if h.repo == nil {
		return nil, source.ErrVideoNotFound
	}
	execution, err := h.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(h.recordingsRoot) == "" || strings.TrimSpace(execution.ResultPath) == "" {
		return nil, source.ErrVideoNotFound
	}
	videoDir := filepath.Join(h.recordingsRoot, executionID.String(), "artifacts", "videos")
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, source.ErrVideoNotFound
		}
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	if len(files) == 0 {
		return nil, source.ErrVideoNotFound
	}
	sort.Strings(files)
	path := filepath.Join(videoDir, files[0])
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		if data, readErr := os.ReadFile(path); readErr == nil {
			contentType = http.DetectContentType(data)
		}
	}
	if contentType == "" {
		contentType = "video/webm"
	}
	return &source.VideoSource{
		Path:        path,
		ContentType: contentType,
	}, nil
}

func (h *Handler) serveRecordedVideo(w http.ResponseWriter, r *http.Request, videoSource *source.VideoSource, format, fileName string) error {
	if videoSource == nil {
		return errors.New("recorded video source missing")
	}
	if cleanup := videoSource.Cleanup; cleanup != nil {
		defer cleanup()
	}

	inputPath := videoSource.Path
	inputExt := strings.ToLower(filepath.Ext(inputPath))
	outputPath := inputPath
	outputType := videoSource.ContentType
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
			encoder := render.NewFFmpegEncoder(render.DetectFFmpegBinary())
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
		encoder := render.NewFFmpegEncoder(render.DetectFFmpegBinary())
		ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
		defer cancel()
		if err := encoder.ConvertToGIF(ctx, inputPath, outputPath, 0, 0); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported export format %q", format)
	}

	if strings.TrimSpace(outputType) == "" {
		outputType = source.DetectVideoContentType(outputPath)
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
