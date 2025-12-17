package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"knowledge-observatory/internal/services/ingest"
)

const (
	defaultChunkSize    = 1200
	defaultChunkOverlap = 150
	maxChunkSize        = 8000
	maxChunkOverlap     = 2000
	maxChunksPerDoc     = 5000
)

type IngestDocumentRequest struct {
	Namespace    string                 `json:"namespace"`
	Collection   string                 `json:"collection,omitempty"`
	DocumentID   string                 `json:"document_id,omitempty"`
	Content      string                 `json:"content"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Visibility   string                 `json:"visibility,omitempty"`
	Source       string                 `json:"source,omitempty"`
	SourceType   string                 `json:"source_type,omitempty"`
	ChunkSize    int                    `json:"chunk_size,omitempty"`
	ChunkOverlap int                    `json:"chunk_overlap,omitempty"`
}

type IngestDocumentResponse struct {
	DocumentID  string   `json:"document_id"`
	Collection  string   `json:"collection"`
	Namespace   string   `json:"namespace"`
	ChunkCount  int      `json:"chunk_count"`
	RecordIDs   []string `json:"record_ids"`
	ContentHash string   `json:"content_hash"`
	TookMS      int64    `json:"took_ms"`
}

func (r *IngestDocumentRequest) normalize() error {
	r.Namespace = strings.TrimSpace(r.Namespace)
	r.Collection = normalizeKnowledgeCollection(r.Collection)
	r.DocumentID = strings.TrimSpace(r.DocumentID)
	r.Content = strings.TrimSpace(r.Content)
	r.Source = strings.TrimSpace(r.Source)
	r.SourceType = strings.TrimSpace(r.SourceType)

	visibility, err := normalizeVisibility(r.Visibility)
	if err != nil {
		return err
	}
	r.Visibility = visibility

	if r.Namespace == "" {
		return errors.New("namespace is required")
	}
	if r.Content == "" {
		return errors.New("content is required")
	}
	if r.DocumentID == "" {
		r.DocumentID = newUUIDv4()
	}

	if r.ChunkSize <= 0 {
		r.ChunkSize = defaultChunkSize
	}
	if r.ChunkOverlap < 0 {
		r.ChunkOverlap = 0
	}
	if r.ChunkSize > maxChunkSize {
		r.ChunkSize = maxChunkSize
	}
	if r.ChunkOverlap > maxChunkOverlap {
		r.ChunkOverlap = maxChunkOverlap
	}
	if r.ChunkOverlap >= r.ChunkSize {
		r.ChunkOverlap = r.ChunkSize / 4
	}
	return nil
}

func (s *Server) handleIngestDocument(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req IngestDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := req.normalize(); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if s == nil || s.ingestService == nil {
		s.respondError(w, http.StatusInternalServerError, "Ingest service unavailable")
		return
	}

	chunks := ingest.ChunkText(req.Content, req.ChunkSize, req.ChunkOverlap, maxChunksPerDoc)
	recordIDs := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		idx := i
		recordID := ingest.RecordIDForChunk(req.Namespace, req.DocumentID, i, chunk)
		_, err := s.ingestService.UpsertRecord(r.Context(), ingest.UpsertRecordRequest{
			Namespace:  req.Namespace,
			Collection: req.Collection,
			RecordID:   recordID,
			DocumentID: req.DocumentID,
			ChunkIndex: &idx,
			Content:    chunk,
			Metadata:   req.Metadata,
			Visibility: req.Visibility,
			Source:     req.Source,
			SourceType: req.SourceType,
		})
		if err != nil {
			s.log("document ingest chunk failed", map[string]interface{}{"error": err.Error(), "chunk_index": i})
			s.respondError(w, http.StatusInternalServerError, "Failed to ingest document")
			return
		}
		recordIDs = append(recordIDs, recordID)
	}

	resp := IngestDocumentResponse{
		DocumentID:  req.DocumentID,
		Collection:  req.Collection,
		Namespace:   req.Namespace,
		ChunkCount:  len(recordIDs),
		RecordIDs:   recordIDs,
		ContentHash: ingest.HashDocument(req.Namespace, req.Content),
		TookMS:      time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
