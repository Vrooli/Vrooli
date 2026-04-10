package main

// DOC: docs/reference/api-endpoints.md#documentation-viewer
import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"knowledge-observatory/internal/services/viewer"
)

type DocsContentResponse struct {
	Path        string           `json:"path"`
	Content     string           `json:"content"`
	Format      string           `json:"format"`
	DocType     string           `json:"doc_type,omitempty"`
	Size        int64            `json:"size"`
	ModifiedAt  time.Time        `json:"modified_at"`
	CanReset    bool             `json:"can_reset"`
	ResetConfig *DocsResetConfig `json:"reset_config,omitempty"`
}

type DocsResetConfig struct {
	MaxAgeDays     int `json:"max_age_days,omitempty"`
	KeepMinEntries int `json:"keep_min_entries,omitempty"`
}

type DocsResetRequest struct {
	Path           string `json:"path"`
	MaxAgeDays     int    `json:"max_age_days,omitempty"`
	KeepMinEntries int    `json:"keep_min_entries,omitempty"`
	PreviewOnly    bool   `json:"preview_only,omitempty"`
}

type DocsResetResponse struct {
	Path           string   `json:"path"`
	DocType        string   `json:"doc_type"`
	RemovedCount   int      `json:"removed_count"`
	KeptCount      int      `json:"kept_count"`
	RemovedEntries []string `json:"removed_entries,omitempty"`
	NewContent     string   `json:"new_content,omitempty"`
	PreviewOnly    bool     `json:"preview_only"`
}

func (s *Server) handleDocsContent(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docViewerService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation viewer service unavailable")
		return
	}
	path := r.URL.Query().Get("path")
	format := r.URL.Query().Get("format")
	result, err := s.docViewerService.GetContent(r.Context(), viewer.DocContentRequest{
		Path:   path,
		Format: format,
	})
	if err != nil {
		respondDocViewerError(w, err)
		return
	}
	if scenario := extractScenarioFromPath(result.Path); scenario != "" {
		s.logDocAccess(r.Context(), scenario, result.DocType, "read")
	}

	var resetConfig *DocsResetConfig
	if result.ResetConfig != nil {
		resetConfig = &DocsResetConfig{
			MaxAgeDays:     result.ResetConfig.MaxAgeDays,
			KeepMinEntries: result.ResetConfig.KeepMinEntries,
		}
	}
	response := DocsContentResponse{
		Path:        result.Path,
		Content:     result.Content,
		Format:      result.Format,
		DocType:     result.DocType,
		Size:        result.Size,
		ModifiedAt:  result.ModifiedAt,
		CanReset:    result.CanReset,
		ResetConfig: resetConfig,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleDocsViewerReset(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docViewerService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation viewer service unavailable")
		return
	}
	var req DocsResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	result, err := s.docViewerService.ResetDocument(r.Context(), viewer.DocResetRequest{
		Path:           req.Path,
		MaxAgeDays:     req.MaxAgeDays,
		KeepMinEntries: req.KeepMinEntries,
		PreviewOnly:    req.PreviewOnly,
	})
	if err != nil {
		respondDocViewerError(w, err)
		return
	}
	if scenario := extractScenarioFromPath(result.Path); scenario != "" {
		s.logDocAccess(r.Context(), scenario, result.DocType, "reset")
	}

	response := DocsResetResponse{
		Path:           result.Path,
		DocType:        result.DocType,
		RemovedCount:   result.RemovedCount,
		KeptCount:      result.KeptCount,
		RemovedEntries: result.RemovedEntries,
		NewContent:     result.NewContent,
		PreviewOnly:    result.PreviewOnly,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func respondDocViewerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, viewer.ErrPathRequired),
		errors.Is(err, viewer.ErrPathInvalid),
		errors.Is(err, viewer.ErrFormatInvalid),
		errors.Is(err, viewer.ErrResetUnsupported):
		respondWithError(w, http.StatusBadRequest, err)
	case errors.Is(err, viewer.ErrDocNotFound):
		respondWithError(w, http.StatusNotFound, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
