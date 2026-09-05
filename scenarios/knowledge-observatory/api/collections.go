package main

// DOC: docs/reference/api-endpoints.md#collection-inventory
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type collectionInventoryResponse struct {
	Collections []collectionInventoryItem `json:"collections"`
	Timestamp   string                    `json:"timestamp"`
}

type collectionInventoryItem struct {
	Name               string `json:"name"`
	TotalPoints        *int   `json:"total_points,omitempty"`
	Ownership          string `json:"ownership"`
	OwnershipLabel     string `json:"ownership_label"`
	IngestAttempts     int    `json:"ingest_attempts"`
	MetadataRows       int    `json:"metadata_rows"`
	DistinctNamespaces int    `json:"distinct_namespaces"`
	LastIngestAt       string `json:"last_ingest_at,omitempty"`
}

type collectionProvenance struct {
	IngestAttempts     int
	MetadataRows       int
	DistinctNamespaces int
	LastIngestAt       string
}

type collectionRecordsResponse struct {
	Collection string                    `json:"collection"`
	TotalCount int                       `json:"total_count"`
	Offset     int                       `json:"offset"`
	Limit      int                       `json:"limit"`
	NextOffset *int                      `json:"next_offset,omitempty"`
	Records    []collectionRecordPreview `json:"records"`
}

type collectionRecordPreview struct {
	ID             string                 `json:"id"`
	Namespace      string                 `json:"namespace,omitempty"`
	DocumentID     string                 `json:"document_id,omitempty"`
	ChunkIndex     *int                   `json:"chunk_index,omitempty"`
	ExternalID     string                 `json:"external_id,omitempty"`
	Visibility     string                 `json:"visibility,omitempty"`
	ContentHash    string                 `json:"content_hash,omitempty"`
	IngestedAt     string                 `json:"ingested_at,omitempty"`
	Source         string                 `json:"source,omitempty"`
	SourceType     string                 `json:"source_type,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	ContentPreview string                 `json:"content_preview,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type deleteCollectionResponse struct {
	Collection               string `json:"collection"`
	Deleted                  bool   `json:"deleted"`
	MetadataRowsDeleted      int64  `json:"metadata_rows_deleted"`
	IngestHistoryRowsDeleted int64  `json:"ingest_history_rows_deleted"`
	Warning                  string `json:"warning,omitempty"`
	Timestamp                string `json:"timestamp"`
}

func (s *Server) handleCollectionInventory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	collections, err := s.getCollections(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to list collections")
		return
	}

	items := make([]collectionInventoryItem, 0, len(collections))
	for _, name := range collections {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		item := collectionInventoryItem{Name: name}
		if count, err := s.getCollectionPointCount(ctx, strings.TrimRight(s.qdrantURL(), "/"), name); err == nil {
			item.TotalPoints = &count
		}
		provenance, err := s.queryCollectionProvenance(ctx, name)
		if err == nil {
			item.IngestAttempts = provenance.IngestAttempts
			item.MetadataRows = provenance.MetadataRows
			item.DistinctNamespaces = provenance.DistinctNamespaces
			item.LastIngestAt = provenance.LastIngestAt
		}
		item.Ownership, item.OwnershipLabel = classifyCollectionOwnership(name, item.IngestAttempts, item.MetadataRows)
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		left, right := 0, 0
		if items[i].TotalPoints != nil {
			left = *items[i].TotalPoints
		}
		if items[j].TotalPoints != nil {
			right = *items[j].TotalPoints
		}
		if left == right {
			return items[i].Name < items[j].Name
		}
		return left > right
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(collectionInventoryResponse{
		Collections: items,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleCollectionRecords(w http.ResponseWriter, r *http.Request) {
	collection := strings.TrimSpace(mux.Vars(r)["collection"])
	if collection == "" {
		s.respondError(w, http.StatusBadRequest, "collection is required")
		return
	}

	limit := parseCollectionPreviewLimit(r.URL.Query().Get("limit"))
	offset := parseCollectionPreviewOffset(r.URL.Query().Get("offset"))
	namespace := strings.TrimSpace(r.URL.Query().Get("namespace"))
	documentID := strings.TrimSpace(r.URL.Query().Get("document_id"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	filter := map[string]interface{}{}
	must := make([]map[string]interface{}, 0, 2)
	if namespace != "" {
		must = append(must, map[string]interface{}{"key": "namespace", "match": map[string]interface{}{"value": namespace}})
	}
	if documentID != "" {
		must = append(must, map[string]interface{}{"key": "document_id", "match": map[string]interface{}{"value": documentID}})
	}
	if len(must) > 0 {
		filter["must"] = must
	}
	var appliedFilter interface{}
	if len(filter) > 0 {
		appliedFilter = filter
	}

	points, err := s.scrollCollectionPoints(r.Context(), collection, 100000, false, appliedFilter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to inspect collection points")
		return
	}

	sort.SliceStable(points, func(i, j int) bool {
		left := payloadUnixMS(points[i].Payload)
		right := payloadUnixMS(points[j].Payload)
		if left == right {
			return points[i].ID < points[j].ID
		}
		return left > right
	})

	records := make([]collectionRecordPreview, 0, len(points))
	for _, point := range points {
		record := mapCollectionRecordPreview(point)
		if search != "" {
			searchText := strings.ToLower(search)
			content := strings.ToLower(record.ContentPreview)
			hash := strings.ToLower(record.ContentHash)
			doc := strings.ToLower(record.DocumentID)
			if !strings.Contains(content, searchText) && !strings.Contains(hash, searchText) && !strings.Contains(doc, searchText) {
				continue
			}
		}
		records = append(records, record)
	}

	total := len(records)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	nextOffset := 0
	var nextOffsetPtr *int
	if end < total {
		nextOffset = end
		nextOffsetPtr = &nextOffset
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(collectionRecordsResponse{
		Collection: collection,
		TotalCount: total,
		Offset:     offset,
		Limit:      limit,
		NextOffset: nextOffsetPtr,
		Records:    records[offset:end],
	})
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	collection := strings.TrimSpace(mux.Vars(r)["collection"])
	if collection == "" {
		s.respondError(w, http.StatusBadRequest, "collection is required")
		return
	}

	if err := s.deleteQdrantCollectionHTTP(r.Context(), collection); err != nil {
		if errors.Is(err, errCollectionNotFound) {
			s.respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		s.respondError(w, http.StatusInternalServerError, "failed to delete collection")
		return
	}

	metadataRowsDeleted, ingestRowsDeleted, cleanupErr := s.deleteCollectionProvenance(r.Context(), collection)

	resp := deleteCollectionResponse{
		Collection:               collection,
		Deleted:                  true,
		MetadataRowsDeleted:      metadataRowsDeleted,
		IngestHistoryRowsDeleted: ingestRowsDeleted,
		Timestamp:                time.Now().UTC().Format(time.RFC3339),
	}
	if cleanupErr != nil {
		resp.Warning = "collection deleted from vector store, but metadata cleanup failed"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) queryCollectionProvenance(ctx context.Context, collection string) (collectionProvenance, error) {
	if s == nil || s.stores == nil {
		return collectionProvenance{}, nil
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return collectionProvenance{}, nil
	}

	provenance, err := s.stores.Ingest.ProvenanceForCollection(ctx, collection)
	if err != nil {
		return collectionProvenance{}, err
	}
	metadataRows, err := s.stores.Metadata.CountByCollection(ctx, collection)
	if err != nil {
		return collectionProvenance{}, err
	}

	out := collectionProvenance{
		IngestAttempts:     provenance.IngestAttempts,
		DistinctNamespaces: provenance.DistinctNamespaces,
		MetadataRows:       metadataRows,
	}
	if provenance.LastIngestAt != nil {
		out.LastIngestAt = provenance.LastIngestAt.Format(time.RFC3339)
	}
	return out, nil
}

func classifyCollectionOwnership(collection string, ingestAttempts, metadataRows int) (key string, label string) {
	collection = strings.TrimSpace(collection)
	switch {
	case collection == defaultKnowledgeCollection:
		return "knowledge_observatory", "Likely KO-managed"
	case ingestAttempts > 0 && metadataRows > 0:
		return "knowledge_observatory", "Likely KO-managed"
	case ingestAttempts > 0 || metadataRows > 0:
		return "mixed", "Likely external/mixed"
	default:
		return "external_or_unknown", "Unknown ownership"
	}
}

func (s *Server) deleteCollectionProvenance(ctx context.Context, collection string) (metadataRowsDeleted int64, ingestRowsDeleted int64, err error) {
	if s == nil || s.stores == nil {
		return 0, 0, nil
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return 0, 0, nil
	}

	metadataRowsDeleted, err = s.stores.Metadata.DeleteByCollection(ctx, collection)
	if err != nil {
		return 0, 0, err
	}

	ingestRowsDeleted, err = s.stores.Ingest.DeleteHistoryByCollection(ctx, collection)
	if err != nil {
		return metadataRowsDeleted, 0, err
	}
	return metadataRowsDeleted, ingestRowsDeleted, nil
}

func mapCollectionRecordPreview(point qdrantPoint) collectionRecordPreview {
	out := collectionRecordPreview{
		ID:          strings.TrimSpace(point.ID),
		Namespace:   payloadString(point.Payload, "namespace"),
		DocumentID:  payloadString(point.Payload, "document_id"),
		ExternalID:  payloadString(point.Payload, "external_id"),
		Visibility:  payloadString(point.Payload, "visibility"),
		ContentHash: payloadString(point.Payload, "content_hash"),
		IngestedAt:  payloadString(point.Payload, "ingested_at"),
		Source:      payloadString(point.Payload, "source"),
		SourceType:  payloadString(point.Payload, "source_type"),
		Tags:        payloadStringList(point.Payload, "tags"),
		Metadata:    payloadObject(point.Payload, "metadata"),
	}
	if idx, ok := payloadInt(point.Payload, "chunk_index"); ok {
		out.ChunkIndex = &idx
	}
	content := payloadString(point.Payload, "content")
	out.ContentPreview = trimRunes(content, 220)
	return out
}

func payloadObject(payload map[string]interface{}, key string) map[string]interface{} {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	typed, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	return typed
}

func payloadStringList(payload map[string]interface{}, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			value = strings.TrimSpace(value)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func payloadInt(payload map[string]interface{}, key string) (int, bool) {
	value := payloadString(payload, key)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func trimRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if value == "" || max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}

func parseCollectionPreviewLimit(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 50
	}
	if value > 200 {
		return 200
	}
	return value
}

func parseCollectionPreviewOffset(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}
