package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/ports"
)

type CreateIngestJobResponse struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	DocumentID string `json:"document_id"`
}

func (s *Server) handleCreateIngestJob(w http.ResponseWriter, r *http.Request) {
	var req IngestDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := req.normalize(); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if s == nil || s.jobStore == nil {
		s.respondError(w, http.StatusInternalServerError, "Job store unavailable")
		return
	}

	jobID, err := s.jobStore.EnqueueDocumentIngest(r.Context(), ports.DocumentIngestJobRequest{
		Namespace:    req.Namespace,
		Collection:   req.Collection,
		DocumentID:   req.DocumentID,
		Content:      req.Content,
		Metadata:     req.Metadata,
		Visibility:   req.Visibility,
		Source:       req.Source,
		SourceType:   req.SourceType,
		ChunkSize:    req.ChunkSize,
		ChunkOverlap: req.ChunkOverlap,
	})
	if err != nil {
		s.log("enqueue failed", map[string]interface{}{"error": err.Error()})
		s.respondError(w, http.StatusInternalServerError, "Failed to enqueue ingest job")
		return
	}

	resp := CreateIngestJobResponse{
		JobID:      jobID,
		Status:     "pending",
		DocumentID: req.DocumentID,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetIngestJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := strings.TrimSpace(vars["job_id"])
	if jobID == "" {
		s.respondError(w, http.StatusBadRequest, "job_id is required")
		return
	}

	if s == nil || s.jobStore == nil {
		s.respondError(w, http.StatusInternalServerError, "Job store unavailable")
		return
	}

	status, ok, err := s.jobStore.GetJob(r.Context(), jobID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to retrieve job")
		return
	}
	if !ok {
		s.respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	// Normalize empty timestamps for UI friendliness
	if status.CreatedAt == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		status.CreatedAt = &now
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

