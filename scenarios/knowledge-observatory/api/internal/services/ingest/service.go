package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"knowledge-observatory/internal/ports"
)

type Service struct {
	VectorStore ports.VectorStore
	Embedder    ports.Embedder
	Metadata    ports.MetadataStore
	Clock       func() time.Time
}

type UpsertRecordRequest struct {
	Namespace  string
	Collection string
	RecordID   string
	DocumentID string
	ChunkIndex *int
	Content    string
	Metadata   map[string]interface{}
	Visibility string
	Source     string
	SourceType string
}

type UpsertRecordResponse struct {
	RecordID    string
	Collection  string
	Namespace   string
	ContentHash string
	TookMS      int64
}

func (r *UpsertRecordRequest) Validate() error {
	r.Namespace = strings.TrimSpace(r.Namespace)
	r.Collection = strings.TrimSpace(r.Collection)
	r.RecordID = strings.TrimSpace(r.RecordID)
	r.Content = strings.TrimSpace(r.Content)
	r.Visibility = strings.TrimSpace(r.Visibility)
	r.Source = strings.TrimSpace(r.Source)
	r.SourceType = strings.TrimSpace(r.SourceType)

	if r.Namespace == "" {
		return errors.New("namespace is required")
	}
	if r.Content == "" {
		return errors.New("content is required")
	}
	if r.Collection == "" {
		return errors.New("collection is required")
	}
	if r.RecordID == "" {
		return errors.New("record_id is required")
	}
	if r.Visibility == "" {
		return errors.New("visibility is required")
	}
	return nil
}

func contentHash(namespace, content string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
	sum := sha256.Sum256([]byte(namespace + "\n" + normalized))
	return hex.EncodeToString(sum[:])
}

func (s *Service) now() time.Time {
	if s != nil && s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

func (s *Service) UpsertRecord(ctx context.Context, req UpsertRecordRequest) (UpsertRecordResponse, error) {
	start := s.now()

	if err := req.Validate(); err != nil {
		return UpsertRecordResponse{}, err
	}
	if s == nil || s.VectorStore == nil || s.Embedder == nil {
		return UpsertRecordResponse{}, errors.New("ingest service dependencies are not configured")
	}

	hash := contentHash(req.Namespace, req.Content)

	embedding, err := s.Embedder.Embed(ctx, req.Content)
	if err != nil {
		_ = s.writeHistory(ctx, req, hash, "failure", err.Error(), time.Since(start).Milliseconds())
		return UpsertRecordResponse{}, fmt.Errorf("embedding generation failed: %w", err)
	}

	if err := s.VectorStore.EnsureCollection(ctx, req.Collection, len(embedding)); err != nil {
		_ = s.writeHistory(ctx, req, hash, "failure", err.Error(), time.Since(start).Milliseconds())
		return UpsertRecordResponse{}, fmt.Errorf("ensure collection failed: %w", err)
	}

	payload := map[string]interface{}{
		"schema_version": "ko.knowledge.v1",
		"namespace":      req.Namespace,
		"visibility":     req.Visibility,
		"document_id":    strings.TrimSpace(req.DocumentID),
		"record_id":      req.RecordID,
		"chunk_index":    req.ChunkIndex,
		"content":        req.Content,
		"content_hash":   hash,
		"ingested_at":    s.now().UTC().Format(time.RFC3339),
		"metadata":       req.Metadata,
	}
	if strings.TrimSpace(req.DocumentID) == "" {
		delete(payload, "document_id")
	}
	if req.ChunkIndex == nil {
		delete(payload, "chunk_index")
	}
	if req.Source != "" {
		payload["source"] = req.Source
	}
	if req.SourceType != "" {
		payload["source_type"] = req.SourceType
	}

	if err := s.VectorStore.UpsertPoint(ctx, req.Collection, req.RecordID, embedding, payload); err != nil {
		_ = s.writeHistory(ctx, req, hash, "failure", err.Error(), time.Since(start).Milliseconds())
		return UpsertRecordResponse{}, fmt.Errorf("upsert point failed: %w", err)
	}

	if s.Metadata != nil {
		sourceType := req.SourceType
		if strings.TrimSpace(sourceType) == "" {
			sourceType = "unknown"
		}
		if err := s.Metadata.UpsertKnowledgeMetadata(ctx, req.RecordID, req.Collection, hash, req.Namespace, sourceType); err != nil {
			_ = s.writeHistory(ctx, req, hash, "failure", err.Error(), time.Since(start).Milliseconds())
			return UpsertRecordResponse{}, fmt.Errorf("metadata upsert failed: %w", err)
		}
	}

	if err := s.writeHistory(ctx, req, hash, "success", "", time.Since(start).Milliseconds()); err != nil {
		return UpsertRecordResponse{}, fmt.Errorf("ingest history insert failed: %w", err)
	}

	return UpsertRecordResponse{
		RecordID:    req.RecordID,
		Collection:  req.Collection,
		Namespace:   req.Namespace,
		ContentHash: hash,
		TookMS:      time.Since(start).Milliseconds(),
	}, nil
}

func (s *Service) writeHistory(ctx context.Context, req UpsertRecordRequest, hash, status, errorMessage string, tookMS int64) error {
	if s == nil || s.Metadata == nil {
		return nil
	}
	return s.Metadata.InsertIngestHistory(ctx, ports.IngestHistoryRow{
		RecordID:     req.RecordID,
		Namespace:    req.Namespace,
		Collection:   req.Collection,
		ContentHash:  hash,
		Visibility:   req.Visibility,
		Source:       req.Source,
		SourceType:   req.SourceType,
		Status:       status,
		ErrorMessage: errorMessage,
		TookMS:       tookMS,
	})
}

func (s *Service) DeleteRecord(ctx context.Context, recordID string) (collection string, err error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return "", errors.New("record_id is required")
	}
	if s == nil || s.VectorStore == nil {
		return "", errors.New("ingest service dependencies are not configured")
	}

	if s.Metadata == nil {
		return "", errors.New("metadata store is required for delete (collection lookup)")
	}

	collection, ok, err := s.Metadata.LookupCollectionForVectorID(ctx, recordID)
	if err != nil {
		return "", err
	}
	if !ok || strings.TrimSpace(collection) == "" {
		return "", errors.New("record not found")
	}

	if err := s.VectorStore.DeletePoint(ctx, collection, recordID); err != nil {
		return collection, err
	}
	return collection, nil
}
