package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
)

// PostExecutionExport handles POST /api/v1/executions/{id}/export
func (h *Handler) PostExecutionExport(w http.ResponseWriter, r *http.Request) {
	executionID, err := parseExecutionID(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	var body executionExportRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSONBodyAllowEmpty(w, r, &body); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid json payload"}))
			return
		}
	}

	format := resolveExportFormat(r, body)

	if format == "folder" {
		h.handleFolderExport(w, r, executionID, body)
		return
	}

	h.handleStreamedExport(w, r, executionID, format, body)
}

func parseExecutionID(raw string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(raw))
}

func resolveExportFormat(r *http.Request, body executionExportRequest) string {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if body.Format != "" {
		format = strings.ToLower(strings.TrimSpace(body.Format))
	}
	if format == "" {
		format = "json"
	}
	return format
}

func (h *Handler) handleFolderExport(w http.ResponseWriter, r *http.Request, executionID uuid.UUID, body executionExportRequest) {
	outputDir := strings.TrimSpace(r.URL.Query().Get("output_dir"))
	if body.OutputDir != "" {
		outputDir = strings.TrimSpace(body.OutputDir)
	}

	if outputDir == "" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "output_dir is required for folder format"}))
		return
	}

	exportCtx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	if err := h.workflowService.ExportToFolder(exportCtx, executionID, outputDir, h.storage); err != nil {
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
}

func (h *Handler) handleStreamedExport(w http.ResponseWriter, r *http.Request, executionID uuid.UUID, format string, body executionExportRequest) {
	previewCtx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	preview, svcErr := h.workflowService.DescribeExecutionExport(previewCtx, executionID)
	if svcErr != nil {
		if errors.Is(svcErr, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.log.WithError(svcErr).WithField("execution_id", executionID).Error("Failed to describe execution export")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "describe_export"}))
		return
	}

	spec, specErr := buildExportSpec(preview.Package, body.MovieSpec, executionID)
	if specErr != nil {
		if errors.Is(specErr, errMovieSpecUnavailable) {
			if format == "json" {
				applyExportOverrides(preview.Package, body.Overrides)
				h.respondSuccess(w, http.StatusOK, preview)
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
		applyExportOverrides(spec, body.Overrides)
		h.respondSuccess(w, http.StatusOK, preview)
		return
	}

	h.streamExport(w, r, executionID, format, body, spec)
}

func (h *Handler) streamExport(w http.ResponseWriter, r *http.Request, executionID uuid.UUID, format string, body executionExportRequest, spec *services.ReplayMovieSpec) {
	response, err := h.workflowService.StreamExecutionExport(r.Context(), executionID, format, spec, body.FileName, body.Overrides, h.storage)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.log.WithError(err).WithFields(map[string]any{
			"execution_id": executionID,
			"format":       format,
		}).Error("Failed to stream execution export")

		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrExportFormatNotSupported) || errors.Is(err, services.ErrExportStillProcessing) {
			status = http.StatusBadRequest
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stream_export", "error": err.Error()})).WithCode(status)
		return
	}

	if response.Headers != nil {
		for k, v := range response.Headers {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(response.Status)
	if len(response.Body) > 0 {
		_, _ = w.Write(response.Body)
	}
}
