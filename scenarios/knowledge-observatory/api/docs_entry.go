package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/ports"
	"knowledge-observatory/internal/services/viewer"
)

type DocsAppendEntryRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body,omitempty"`
	Author string `json:"author,omitempty"`
	Status string `json:"status,omitempty"`
}

type DocsAppendEntryResponse struct {
	ScenarioName string `json:"scenario_name"`
	DocType      string `json:"doc_type"`
	EntryAdded   string `json:"entry_added"`
}

type DocsAccessStatsResponse struct {
	Stats []DocsAccessStatEntry `json:"stats"`
}

type DocsAccessStatEntry struct {
	ScenarioName string `json:"scenario_name"`
	DocType      string `json:"doc_type"`
	ReadCount    int    `json:"read_count"`
	WriteCount   int    `json:"write_count"`
	ResetCount   int    `json:"reset_count"`
}

func (s *Server) handleDocsReadByType(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docViewerService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation viewer service unavailable")
		return
	}
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	docType := vars["doc_type"]
	format := r.URL.Query().Get("format")

	result, err := s.docViewerService.GetContentByType(r.Context(), viewer.DocReadByTypeRequest{
		ScenarioName: scenarioName,
		DocType:      docType,
		Format:       format,
	})
	if err != nil {
		respondDocEntryError(w, err)
		return
	}

	s.logDocAccess(r.Context(), scenarioName, docType, "read")

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

func (s *Server) handleDocsAppendEntry(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docViewerService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation viewer service unavailable")
		return
	}
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	docType := vars["doc_type"]

	var req DocsAppendEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := s.docViewerService.AppendEntry(r.Context(), viewer.DocAppendRequest{
		ScenarioName: scenarioName,
		DocType:      docType,
		Title:        req.Title,
		Body:         req.Body,
		Author:       req.Author,
		Status:       req.Status,
	})
	if err != nil {
		respondDocEntryError(w, err)
		return
	}

	s.logDocAccess(r.Context(), scenarioName, docType, "write")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(DocsAppendEntryResponse{
		ScenarioName: result.ScenarioName,
		DocType:      result.DocType,
		EntryAdded:   result.EntryAdded,
	})
}

func (s *Server) handleDocsAccessStats(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docAccessLogger == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Access tracking unavailable")
		return
	}

	scenario := r.URL.Query().Get("scenario")
	docType := r.URL.Query().Get("doc_type")

	stats, err := s.docAccessLogger.QueryStats(r.Context(), ports.DocAccessFilter{
		ScenarioName: scenario,
		DocType:      docType,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	entries := make([]DocsAccessStatEntry, len(stats))
	for i, s := range stats {
		entries[i] = DocsAccessStatEntry{
			ScenarioName: s.ScenarioName,
			DocType:      s.DocType,
			ReadCount:    s.ReadCount,
			WriteCount:   s.WriteCount,
			ResetCount:   s.ResetCount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(DocsAccessStatsResponse{Stats: entries})
}

func (s *Server) logDocAccess(ctx context.Context, scenario, docType, operation string) {
	if s == nil || s.docAccessLogger == nil {
		return
	}
	detached := context.WithoutCancel(ctx)
	go func() {
		if err := s.docAccessLogger.LogAccess(detached, ports.DocAccessRow{
			ScenarioName: scenario,
			DocType:      docType,
			Operation:    operation,
		}); err != nil {
			log.Printf("doc access log error: %v", err)
		}
	}()
}

func extractScenarioFromPath(relPath string) string {
	// Paths look like: scenarios/<name>/docs/internal/PROBLEMS.md
	parts := strings.Split(strings.ReplaceAll(relPath, "\\", "/"), "/")
	for i, part := range parts {
		if part == "scenarios" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func respondDocEntryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, viewer.ErrScenarioRequired),
		errors.Is(err, viewer.ErrScenarioInvalid),
		errors.Is(err, viewer.ErrDocTypeInvalid),
		errors.Is(err, viewer.ErrTitleRequired),
		errors.Is(err, viewer.ErrAppendUnsupported),
		errors.Is(err, viewer.ErrFormatInvalid):
		respondWithError(w, http.StatusBadRequest, err)
	case errors.Is(err, viewer.ErrDocNotFound):
		respondWithError(w, http.StatusNotFound, err)
	case errors.Is(err, viewer.ErrServiceUnavailable):
		respondWithError(w, http.StatusServiceUnavailable, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
