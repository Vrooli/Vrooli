package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"knowledge-observatory/internal/services/ingest"
)

type UpsertRecordRequest struct {
	Namespace  string                 `json:"namespace"`
	Collection string                 `json:"collection,omitempty"`
	RecordID   string                 `json:"record_id,omitempty"`
	ExternalID string                 `json:"external_id,omitempty"`
	Content    string                 `json:"content"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Visibility string                 `json:"visibility,omitempty"`
	Source     string                 `json:"source,omitempty"`
	SourceType string                 `json:"source_type,omitempty"`
}

type UpsertRecordResponse struct {
	RecordID    string `json:"record_id"`
	Collection  string `json:"collection"`
	Namespace   string `json:"namespace"`
	ContentHash string `json:"content_hash"`
	Upserted    bool   `json:"upserted"`
	TookMS      int64  `json:"took_ms"`
}

func (r *UpsertRecordRequest) normalize() error {
	r.Namespace = strings.TrimSpace(r.Namespace)
	r.Collection = normalizeKnowledgeCollection(r.Collection)
	r.RecordID = strings.TrimSpace(r.RecordID)
	r.ExternalID = strings.TrimSpace(r.ExternalID)
	r.Content = strings.TrimSpace(r.Content)
	visibility, err := normalizeVisibility(r.Visibility)
	if err != nil {
		return err
	}
	r.Visibility = visibility
	r.Source = strings.TrimSpace(r.Source)
	r.SourceType = strings.TrimSpace(r.SourceType)

	if r.Namespace == "" {
		return errors.New("namespace is required")
	}
	if r.Content == "" {
		return errors.New("content is required")
	}
	if r.RecordID == "" {
		if r.ExternalID != "" {
			r.RecordID = ingest.HashExternalID(r.Namespace, r.ExternalID)
		} else {
			r.RecordID = ingest.HashDocument(r.Namespace, r.Content)
		}
	}
	return nil
}

func (s *Server) handleUpsertRecord(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req UpsertRecordRequest
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

	out, err := s.ingestService.UpsertRecord(r.Context(), ingest.UpsertRecordRequest{
		Namespace:  req.Namespace,
		Collection: req.Collection,
		RecordID:   req.RecordID,
		ExternalID: req.ExternalID,
		Content:    req.Content,
		Tags:       normalizeStringList(req.Tags),
		Metadata:   req.Metadata,
		Visibility: req.Visibility,
		Source:     req.Source,
		SourceType: req.SourceType,
	})
	if err != nil {
		s.log("ingest upsert failed", map[string]interface{}{"error": err.Error(), "collection": req.Collection})
		s.respondError(w, http.StatusInternalServerError, "Failed to upsert record")
		return
	}

	resp := UpsertRecordResponse{
		RecordID:    out.RecordID,
		Collection:  out.Collection,
		Namespace:   out.Namespace,
		ContentHash: out.ContentHash,
		Upserted:    true,
		TookMS:      time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
