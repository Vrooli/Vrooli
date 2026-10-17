package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/handlers/export"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/export/render"
	"github.com/vrooli/browser-automation-studio/services/export/source"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// PostExecutionExport handles POST /api/v1/executions/{id}/export
// For binary formats (mp4/gif) with output_dir: returns immediately with export ID, renders in background
// For json format: returns export spec synchronously
// For html format: streams a synchronous interactive bundle
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

	if format != "json" && format != "html" && format != "folder" && format != "mp4" && format != "gif" && format != "webm" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "unsupported export format"}))
		return
	}

	if body.Stylization != "" && body.Stylization != "raw" && body.Stylization != "stylized" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "stylization must be raw or stylized"}))
		return
	}
	if format == "webm" && body.AppliesStylesToRecording() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "styled recordings support MP4/GIF; select raw for WebM"}))
		return
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

	// JSON and HTML consume the same resolved presentation as binary replay.

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
	if preview == nil {
		h.log.WithField("execution_id", executionID).Error("Execution export service returned an empty preview")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "describe_export", "error": "empty export preview"}))
		return
	}

	generatedSpec, specErr := exportservices.BuildReplaySpec(preview.Package, body.MovieSpec, executionID)
	if specErr != nil {
		if errors.Is(specErr, exportservices.ErrMovieSpecUnavailable) {
			if format == "json" {
				pbPreview, conversionErr := protoconv.ExecutionExportPreviewToProto(preview)
				if conversionErr != nil {
					h.log.WithError(conversionErr).WithField("execution_id", executionID).Error("Failed to encode execution export preview")
					h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "encode_export_preview"}))
					return
				}
				h.respondProto(w, http.StatusOK, pbPreview)
			} else {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"error": "export package unavailable"}))
			}
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": specErr.Error()}))
		return
	}
	if err := h.resolveExportPresentation(previewCtx, generatedSpec, body); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "prepare_export", "error": err.Error()}))
		return
	}

	if format == "json" {
		setExportReceiptHeaders(w, generatedSpec, source.RenderSourceReplayFrames)
		preview.Package = generatedSpec
		if pbPreview, err := protoconv.ExecutionExportPreviewToProto(preview); err == nil {
			h.respondProto(w, http.StatusOK, pbPreview)
		} else {
			h.respondSuccess(w, http.StatusOK, preview)
		}
		return
	}

	if format == "html" {
		setExportReceiptHeaders(w, generatedSpec, source.RenderSourceReplayFrames)
		filename := normalizeExportFilename(body.FileName, "replay-export", ".zip")
		w.Header().Set("Content-Type", "application/zip")
		if strings.TrimSpace(filename) != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}

		baseURL := requestBaseURL(r)
		if err := exportservices.WriteHTMLBundle(previewCtx, w, generatedSpec, h.storage, h.log, baseURL); err != nil {
			h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to build HTML replay export")
		}
		return
	}

}

func setExportReceiptHeaders(w http.ResponseWriter, spec *exportsv1.ReplaySpec, selectedSource string) {
	w.Header().Set("X-BAS-Export-Selected-Source", selectedSource)
	w.Header().Set("X-BAS-Export-Edit-Map-Status", "not_represented_by_export_contract")
	if spec != nil && spec.GetCursor() != nil && spec.GetCursor().GetStyle() != "" {
		w.Header().Set("X-BAS-Export-Cursor-Style", spec.GetCursor().GetStyle())
	}
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
	if err := exportservices.ValidateFilename(exportName); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	// Create export record in "processing" state
	exportRecord := &database.ExportIndex{
		ID:          uuid.New(),
		ExecutionID: executionID,
		WorkflowID:  &execution.WorkflowID,
		Name:        exportName,
		Format:      format,
		Status:      "processing",
		Settings:    buildExportReceiptSettings(body, renderSource),
	}

	if err := h.exportJobs.Start(ctx, exportRecord.ID,
		func(ctx context.Context) error { return h.repo.CreateExport(ctx, exportRecord) },
		func(ctx context.Context) {
			h.renderExportInBackground(ctx, exportRecord, executionID, format, body, outputDir, renderSource)
		},
	); err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to create export record")
		if errors.Is(err, exportservices.ErrExportCapacity) {
			h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_export"}))
		return
	}

	exportID := exportRecord.ID.String()
	execIDStr := executionID.String()

	// Return immediately with export ID
	h.respondSuccess(w, http.StatusAccepted, map[string]any{
		"export_id":    exportID,
		"execution_id": execIDStr,
		"status":       "processing",
		"message":      "Export started. Subscribe to WebSocket for progress updates.",
	})

}

// renderExportInBackground has one source-selection and finalization path.
func (h *Handler) renderExportInBackground(ctx context.Context, row *database.ExportIndex, executionID uuid.UUID, format string, body export.Request, outputDir, requestedSource string) {
	progress := func(stage string, percent float64) {
		h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{ExportID: row.ID.String(), ExecutionID: executionID.String(), Stage: stage, ProgressPercent: percent, Status: "processing"})
	}
	// Failure persistence must survive cancellation, while retaining routing values.
	fail := func(err error) {
		statusCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if persistErr := h.repo.UpdateExportStatus(statusCtx, row.ID, "failed", err.Error()); persistErr != nil {
			h.log.WithError(persistErr).WithField("export_id", row.ID).Error("Export failed; terminal state could not be persisted: " + err.Error())
			return
		}
		h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{ExportID: row.ID.String(), ExecutionID: executionID.String(), Stage: "failed", Status: "failed", Error: err.Error()})
	}
	// An unexpected worker panic must not strand an admitted row or the process.
	defer func() {
		if value := recover(); value != nil {
			fail(fmt.Errorf("export worker panic: %v", value))
		}
	}()
	progress("preparing", 10)
	media, selected, err := h.prepareExportMedia(ctx, executionID, format, body, requestedSource, row)
	if err != nil {
		fail(err)
		return
	}
	defer media.Cleanup()
	row.Settings["selected_source"] = selected
	row.Settings["rendered_content_type"] = media.ContentType
	if err = h.repo.UpdateExportSettings(ctx, row.ID, row.Settings); err != nil {
		fail(fmt.Errorf("persist export source receipt: %w", err))
		return
	}
	progress("finalizing", 90)
	published, err := exportservices.PublishFile(ctx, outputDir, normalizeExportFilename(row.Name, "export-"+row.ID.String(), "."+format), func(path string) error { return exportservices.CopyMedia(ctx, media.Path, path) })
	if err != nil {
		fail(err)
		return
	}
	if err = exportservices.CommitExport(ctx, h.repo, row, published); err != nil {
		fail(err)
		return
	}
	h.wsHub.BroadcastExportProgress(&wsHub.ExportProgress{ExportID: row.ID.String(), ExecutionID: executionID.String(), Stage: "completed", ProgressPercent: 100, Status: "completed", StorageURL: published.Path, FileSizeBytes: published.Size})
}

// prepareExportMedia preserves source cleanup until the resulting media is consumed.
func (h *Handler) prepareExportMedia(ctx context.Context, executionID uuid.UUID, format string, body export.Request, requested string, row *database.ExportIndex) (*render.RenderedMedia, string, error) {
	if requested != source.RenderSourceReplayFrames {
		video, err := h.loadRecordedVideo(ctx, executionID)
		if err == nil {
			if video.Cleanup != nil {
				defer video.Cleanup()
			}
			video, err = source.FreezeVideoSource(ctx, video, filepath.Join(h.recordingsRoot, executionID.String(), "artifacts", "export-sources"))
			if err != nil {
				return nil, "", err
			}
			defer video.Cleanup()

			if err = setExportSelectedSource(ctx, h.repo, row, source.RenderSourceRecordedVideo, video); err != nil {
				return nil, "", err
			}
			if body.AppliesStylesToRecording() {
				spec, cleanup, err := h.prepareStyledRecordedVideo(ctx, executionID, video, body)
				if err != nil {
					return nil, "", err
				}
				defer cleanup()
				for key, value := range buildRenderedSpecReceipt(spec) {
					row.Settings[key] = value
				}
				row.Settings["cursor_application"] = "source-pixels; replay-overlay-disabled"
				media, err := h.replayRenderer.Render(ctx, spec, render.RenderFormat(format), body.FileName)
				if err != nil {
					return nil, "", err
				}
				if err = render.FinishRecordedVideo(ctx, media, video.Path, spec.GetSummary().GetTotalDurationMs(), format); err != nil {
					media.Cleanup()
					return nil, "", err
				}
				return media, source.RenderSourceRecordedVideo, nil
			}
			media, err := render.ConvertRecordedMedia(ctx, video.Path, format)
			return media, source.RenderSourceRecordedVideo, err
		}
		if requested == source.RenderSourceRecordedVideo || format == "webm" || !errors.Is(err, source.ErrVideoNotFound) {
			return nil, "", fmt.Errorf("recorded source unavailable: %w", err)
		}
	}
	preview, err := h.executionService.DescribeExecutionExport(ctx, executionID)
	if err != nil {
		return nil, "", err
	}
	if preview == nil {
		return nil, "", errors.New("empty export preview")
	}
	spec, err := exportservices.BuildReplaySpec(preview.Package, body.MovieSpec, executionID)
	if err != nil {
		return nil, "", err
	}
	if err = h.resolveExportPresentation(ctx, spec, body); err != nil {
		return nil, "", err
	}
	for key, value := range buildRenderedSpecReceipt(spec) {
		row.Settings[key] = value
	}
	if spec.GetCursor() != nil {
		row.Settings["cursor_style"] = spec.GetCursor().GetStyle()
	}
	media, err := h.replayRenderer.Render(ctx, spec, render.RenderFormat(format), body.FileName)
	return media, source.RenderSourceReplayFrames, err
}

func buildExportReceiptSettings(body export.Request, requestedSource string) database.JSONMap {
	receipt := database.JSONMap{
		"requested_source": requestedSource,
		"selected_source":  "pending",
		"stylization":      body.Stylization,
		"edit_map_status":  "not_represented_by_export_contract",
	}
	if body.MovieSpec != nil {
		receipt["frame_count"] = len(body.MovieSpec.GetFrames())
		receipt["cursor_style"] = body.MovieSpec.GetCursor().GetStyle()
		if specJSON, err := protojson.Marshal(body.MovieSpec); err == nil {
			hash := sha256.Sum256(specJSON)
			receipt["movie_spec_sha256"] = hex.EncodeToString(hash[:])
		}
		receipt["edit_map_status"] = "not_represented_by_export_contract"
	}
	if body.Overrides != nil {
		if body.Overrides.Cursor != nil {
			receipt["cursor_style"] = body.Overrides.Cursor.GetStyle()
			receipt["cursor_initial_position"] = body.Overrides.Cursor.GetInitialPosition()
			receipt["cursor_scale"] = body.Overrides.Cursor.GetScale()
			receipt["click_animation"] = body.Overrides.Cursor.GetClickAnimation()
		}
		if body.Overrides.CursorPreset != nil {
			preset := body.Overrides.CursorPreset
			receipt["cursor_style"] = preset.Theme
			receipt["cursor_initial_position"] = preset.InitialPosition
			receipt["cursor_scale"] = preset.Scale
			receipt["click_animation"] = preset.ClickAnimation
		}
		if body.Overrides.ThemePreset != nil {
			receipt["chrome_theme"] = body.Overrides.ThemePreset.ChromeTheme
			receipt["background_theme"] = body.Overrides.ThemePreset.BackgroundTheme
		}
		if body.Overrides.Theme != nil {
			receipt["chrome_theme"] = body.Overrides.Theme.GetBrowserChrome().GetVariant()
			receipt["background_theme"] = body.Overrides.Theme.GetBackgroundPattern()
		}
	}
	return receipt
}

func buildRenderedSpecReceipt(spec *exportsv1.ReplaySpec) database.JSONMap {
	if spec == nil {
		return nil
	}
	receipt := database.JSONMap{"rendered_frame_count": len(spec.GetFrames())}
	if specJSON, err := protojson.Marshal(spec); err == nil {
		hash := sha256.Sum256(specJSON)
		receipt["rendered_spec_sha256"] = hex.EncodeToString(hash[:])
	}
	return receipt
}

func setExportSelectedSource(ctx context.Context, repo database.Repository, record *database.ExportIndex, selected string, video *source.VideoSource) error {
	if record == nil || repo == nil {
		return fmt.Errorf("export repository and record are required")
	}
	if record.Settings == nil {
		record.Settings = database.JSONMap{}
	}
	record.Settings["selected_source"] = selected
	if video != nil {
		if video.ArtifactID != "" {
			record.Settings["source_artifact_id"] = video.ArtifactID
		}
		if video.SHA256 != "" {
			record.Settings["source_sha256"] = video.SHA256
		}
		if video.ContentType != "" {
			record.Settings["source_content_type"] = video.ContentType
		}
		if video.SizeBytes != nil {
			record.Settings["source_size_bytes"] = *video.SizeBytes
		}
	}
	return repo.UpdateExportSettings(ctx, record.ID, record.Settings)
}

func (h *Handler) loadRecordedVideo(ctx context.Context, id uuid.UUID) (*source.VideoSource, error) {
	if h.repo == nil {
		return nil, source.ErrVideoNotFound
	}
	return exportservices.ResolveExecutionRecording(ctx, h.repo, h.recordingsRoot, id, h.storage)
}

// prepareStyledRecordedVideo binds a measured continuous source to the same
// canonical presentation renderer used by preview and replay export.
func (h *Handler) prepareStyledRecordedVideo(ctx context.Context, executionID uuid.UUID, video *source.VideoSource, body export.Request) (*exportsv1.ReplaySpec, func(), error) {
	base := strings.TrimRight(requestAPIBaseURL(), "/")
	sourcePath, cleanup, err := render.BrowserCompatibleRecording(ctx, video.Path, filepath.Join(h.recordingsRoot, executionID.String(), "artifacts", "export-sources"))
	if err != nil {
		return nil, func() {}, err
	}
	relative, err := filepath.Rel(filepath.Join(h.recordingsRoot, executionID.String()), sourcePath)
	if err != nil || strings.HasPrefix(relative, "..") {
		cleanup()
		return nil, func() {}, fmt.Errorf("recorded source outside execution storage")
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	sourceURL := base + "/api/v1/recordings/assets/" + executionID.String() + "/" + strings.Join(parts, "/")
	spec := &exportsv1.ReplaySpec{}
	if body.MovieSpec != nil {
		spec = proto.Clone(body.MovieSpec).(*exportsv1.ReplaySpec)
	}
	if err := h.resolveExportPresentation(ctx, spec, body); err != nil {
		cleanup()
		return nil, func() {}, err
	}

	prepared, err := render.PrepareRecordedVideoSpec(ctx, spec, sourcePath, sourceURL)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return prepared, cleanup, nil
}

// resolveExportPresentation adapts saved preferences once, then uses the service's
// canonical override/policy operations for JSON, HTML, replay and recorded video.
func (h *Handler) resolveExportPresentation(ctx context.Context, spec *exportsv1.ReplaySpec, body export.Request) error {
	config, err := h.loadReplayConfig(ctx)
	if err != nil {
		return fmt.Errorf("load export presentation: %w", err)
	}
	applyReplayConfigToSpec(spec, config)
	exportservices.Apply(spec, replayConfigToOverrides(config))
	if body.MovieSpec != nil {
		if body.MovieSpec.Decor != nil {
			spec.Decor = proto.Clone(body.MovieSpec.Decor).(*exportsv1.ReplayDecor)
		}
		if body.MovieSpec.Presentation != nil {
			spec.Presentation = proto.Clone(body.MovieSpec.Presentation).(*exportsv1.ReplayPresentation)
		}
	}
	exportservices.Apply(spec, body.Overrides)
	if h.entitlementService != nil {
		required := false
		if ent := entitlement.FromContext(ctx); ent != nil {
			required = !ent.HasFeature(entitlement.FeatureWatermarkFree)
		} else {
			required = h.entitlementService.RequiresWatermark(ctx, entitlement.UserIdentityFromContext(ctx))
		}
		exportservices.EnforceWatermarkRequirements(spec, required)
	}
	return nil
}

func requestAPIBaseURL() string {
	scheme := strings.TrimSpace(os.Getenv("API_SCHEME"))
	if scheme == "" {
		scheme = "http"
	}
	host := strings.TrimSpace(os.Getenv("API_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port != "" {
		host = net.JoinHostPort(host, port)
	}
	return scheme + "://" + host
}
