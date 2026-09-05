package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type collectionDiagnosticsResponse struct {
	Collection           string                         `json:"collection"`
	Mode                 string                         `json:"mode"`
	TotalPoints          *int                           `json:"total_points,omitempty"`
	AnalyzedPoints       int                            `json:"analyzed_points"`
	VectorDimensions     []dimensionCount               `json:"vector_dimensions"`
	Namespaces           []namedCount                   `json:"namespaces"`
	ChunkLength          chunkLengthStats               `json:"chunk_length"`
	MissingPayloadFields map[string]int                 `json:"missing_payload_fields"`
	Redundancy           redundancyDiagnostics          `json:"redundancy"`
	StaleChunks          staleChunkDiagnostics          `json:"stale_chunks"`
	IngestHistory        *collectionIngestHistoryHealth `json:"ingest_history,omitempty"`
	Recommendations      []string                       `json:"recommendations"`
	Timestamp            string                         `json:"timestamp"`
}

type dimensionCount struct {
	Dimension int `json:"dimension"`
	Count     int `json:"count"`
}

type namedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type chunkLengthStats struct {
	MinCharacters int     `json:"min_characters"`
	MaxCharacters int     `json:"max_characters"`
	AvgCharacters float64 `json:"avg_characters"`
}

type redundancyDiagnostics struct {
	DuplicateContentHashes int     `json:"duplicate_content_hashes"`
	DuplicatePointCount    int     `json:"duplicate_point_count"`
	DuplicateRatio         float64 `json:"duplicate_ratio"`
}

type staleChunkDiagnostics struct {
	GroupsDetected      int          `json:"groups_detected"`
	CandidateDeleteRows int          `json:"candidate_delete_rows"`
	TopDocuments        []namedCount `json:"top_documents"`
}

type collectionIngestHistoryHealth struct {
	TotalAttempts       int     `json:"total_attempts"`
	SuccessCount        int     `json:"success_count"`
	FailureCount        int     `json:"failure_count"`
	FailureCountLast24H int     `json:"failure_count_last_24h"`
	FailureRate         float64 `json:"failure_rate"`
	LastFailureAt       string  `json:"last_failure_at,omitempty"`
}

type collectionMaintenanceRequest struct {
	DryRun     bool `json:"dry_run"`
	MaxDeletes int  `json:"max_deletes,omitempty"`
}

type collectionMaintenanceResponse struct {
	Collection           string `json:"collection"`
	Action               string `json:"action"`
	DryRun               bool   `json:"dry_run"`
	AnalyzedPoints       int    `json:"analyzed_points"`
	CandidateDeleteCount int    `json:"candidate_delete_count"`
	DeletedCount         int    `json:"deleted_count"`
	TookMS               int64  `json:"took_ms"`
}

type qdrantScrollRequest struct {
	Limit       int         `json:"limit"`
	WithPayload bool        `json:"with_payload"`
	WithVectors bool        `json:"with_vectors"`
	Filter      interface{} `json:"filter,omitempty"`
	Offset      interface{} `json:"offset,omitempty"`
}

type qdrantScrollResponse struct {
	Result struct {
		Points []struct {
			ID      interface{}            `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"points"`
		NextPageOffset interface{} `json:"next_page_offset"`
	} `json:"result"`
}

type qdrantPoint struct {
	ID      string
	Vector  []float64
	Payload map[string]interface{}
}

func (s *Server) handleCollectionDiagnostics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collection := strings.TrimSpace(vars["collection"])
	if collection == "" {
		s.respondError(w, http.StatusBadRequest, "collection is required")
		return
	}

	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = "sample"
	}
	if mode != "sample" && mode != "full" {
		s.respondError(w, http.StatusBadRequest, "mode must be one of: sample, full")
		return
	}

	limit, err := parsePositiveIntQuery(r, "limit")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if mode == "sample" {
		if limit <= 0 {
			limit = 500
		}
		if limit > 5000 {
			limit = 5000
		}
	} else {
		if limit <= 0 {
			limit = 20000
		}
		if limit > 100000 {
			limit = 100000
		}
	}

	start := time.Now()
	points, err := s.scrollCollectionPoints(r.Context(), collection, limit, true, nil)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to inspect collection points")
		return
	}

	var totalPoints *int
	if count, err := s.getCollectionPointCount(r.Context(), strings.TrimRight(s.qdrantURL(), "/"), collection); err == nil {
		totalPoints = &count
	}

	report := buildCollectionDiagnostics(points)
	report.Collection = collection
	report.Mode = mode
	report.TotalPoints = totalPoints
	report.Timestamp = time.Now().UTC().Format(time.RFC3339)

	if ingestStats, err := s.queryCollectionIngestHistory(r.Context(), collection); err == nil {
		report.IngestHistory = ingestStats
	}

	report.Recommendations = diagnosticsRecommendations(report)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Processing-Time-Ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
	_ = json.NewEncoder(w).Encode(report)
}

func (s *Server) handlePruneStaleChunks(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	collection := strings.TrimSpace(mux.Vars(r)["collection"])
	if collection == "" {
		s.respondError(w, http.StatusBadRequest, "collection is required")
		return
	}

	var req collectionMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	points, err := s.scrollCollectionPoints(r.Context(), collection, 100000, false, nil)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to inspect collection points")
		return
	}
	candidateIDs := staleChunkDeleteCandidates(points)
	if req.MaxDeletes > 0 && len(candidateIDs) > req.MaxDeletes {
		candidateIDs = candidateIDs[:req.MaxDeletes]
	}

	deleted := 0
	if !req.DryRun && len(candidateIDs) > 0 {
		if err := s.deletePointsByIDBatch(r.Context(), collection, candidateIDs); err != nil {
			s.respondError(w, http.StatusInternalServerError, "failed to delete stale chunks")
			return
		}
		deleted = len(candidateIDs)
	}

	resp := collectionMaintenanceResponse{
		Collection:           collection,
		Action:               "prune_stale_chunks",
		DryRun:               req.DryRun,
		AnalyzedPoints:       len(points),
		CandidateDeleteCount: len(candidateIDs),
		DeletedCount:         deleted,
		TookMS:               time.Since(start).Milliseconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDedupeContent(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	collection := strings.TrimSpace(mux.Vars(r)["collection"])
	if collection == "" {
		s.respondError(w, http.StatusBadRequest, "collection is required")
		return
	}

	var req collectionMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	points, err := s.scrollCollectionPoints(r.Context(), collection, 100000, false, nil)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to inspect collection points")
		return
	}
	candidateIDs := duplicateContentDeleteCandidates(points)
	if req.MaxDeletes > 0 && len(candidateIDs) > req.MaxDeletes {
		candidateIDs = candidateIDs[:req.MaxDeletes]
	}

	deleted := 0
	if !req.DryRun && len(candidateIDs) > 0 {
		if err := s.deletePointsByIDBatch(r.Context(), collection, candidateIDs); err != nil {
			s.respondError(w, http.StatusInternalServerError, "failed to delete duplicate content chunks")
			return
		}
		deleted = len(candidateIDs)
	}

	resp := collectionMaintenanceResponse{
		Collection:           collection,
		Action:               "dedupe_content_hash",
		DryRun:               req.DryRun,
		AnalyzedPoints:       len(points),
		CandidateDeleteCount: len(candidateIDs),
		DeletedCount:         deleted,
		TookMS:               time.Since(start).Milliseconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func parsePositiveIntQuery(r *http.Request, key string) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return parsed, nil
}

func buildCollectionDiagnostics(points []qdrantPoint) collectionDiagnosticsResponse {
	out := collectionDiagnosticsResponse{
		AnalyzedPoints:       len(points),
		VectorDimensions:     []dimensionCount{},
		Namespaces:           []namedCount{},
		MissingPayloadFields: map[string]int{},
		Recommendations:      []string{},
	}
	if len(points) == 0 {
		return out
	}

	dimensionCounts := map[int]int{}
	namespaceCounts := map[string]int{}
	contentHashCounts := map[string]int{}
	chunkKeyCounts := map[string]int{}
	staleDocCounts := map[string]int{}
	totalChunkChars := 0
	minChunkChars := -1
	maxChunkChars := 0

	for _, point := range points {
		dimensionCounts[len(point.Vector)]++
		payload := point.Payload
		if payload == nil {
			out.MissingPayloadFields["payload"]++
			continue
		}

		namespace := payloadString(payload, "namespace")
		documentID := payloadString(payload, "document_id")
		chunkIndex := payloadString(payload, "chunk_index")
		contentHash := payloadString(payload, "content_hash")
		content := payloadString(payload, "content")

		if namespace == "" {
			out.MissingPayloadFields["namespace"]++
		} else {
			namespaceCounts[namespace]++
		}
		if documentID == "" {
			out.MissingPayloadFields["document_id"]++
		}
		if chunkIndex == "" {
			out.MissingPayloadFields["chunk_index"]++
		}
		if contentHash == "" {
			out.MissingPayloadFields["content_hash"]++
		} else {
			contentHashCounts[contentHash]++
		}
		if payloadString(payload, "ingested_at") == "" && payloadString(payload, "ingested_at_unix_ms") == "" {
			out.MissingPayloadFields["ingested_at"]++
		}
		if content == "" {
			out.MissingPayloadFields["content"]++
		} else {
			size := len([]rune(content))
			totalChunkChars += size
			if minChunkChars < 0 || size < minChunkChars {
				minChunkChars = size
			}
			if size > maxChunkChars {
				maxChunkChars = size
			}
		}

		if namespace != "" && documentID != "" && chunkIndex != "" {
			chunkKey := namespace + "|" + documentID + "|" + chunkIndex
			chunkKeyCounts[chunkKey]++
		}
	}

	out.ChunkLength = chunkLengthStats{
		MinCharacters: maxInt(minChunkChars, 0),
		MaxCharacters: maxChunkChars,
	}
	if len(points) > 0 {
		out.ChunkLength.AvgCharacters = float64(totalChunkChars) / float64(len(points))
	}

	duplicateContentHashes := 0
	duplicatePoints := 0
	for _, count := range contentHashCounts {
		if count > 1 {
			duplicateContentHashes++
			duplicatePoints += count - 1
		}
	}
	out.Redundancy = redundancyDiagnostics{
		DuplicateContentHashes: duplicateContentHashes,
		DuplicatePointCount:    duplicatePoints,
		DuplicateRatio:         ratio(duplicatePoints, len(points)),
	}

	groupsDetected := 0
	candidateDeletes := 0
	for chunkKey, count := range chunkKeyCounts {
		if count <= 1 {
			continue
		}
		groupsDetected++
		extras := count - 1
		candidateDeletes += extras
		parts := strings.SplitN(chunkKey, "|", 3)
		if len(parts) >= 2 {
			docKey := parts[0] + "/" + parts[1]
			staleDocCounts[docKey] += extras
		}
	}
	out.StaleChunks = staleChunkDiagnostics{
		GroupsDetected:      groupsDetected,
		CandidateDeleteRows: candidateDeletes,
		TopDocuments:        topCounts(staleDocCounts, 10),
	}

	out.VectorDimensions = dimensionCountsToSlice(dimensionCounts)
	out.Namespaces = topCounts(namespaceCounts, 20)
	return out
}

func diagnosticsRecommendations(report collectionDiagnosticsResponse) []string {
	out := make([]string, 0, 6)
	if report.Redundancy.DuplicateRatio > 0.15 {
		out = append(out, "Run dedupe_content_hash to reduce duplicate chunk embeddings.")
	}
	if report.StaleChunks.CandidateDeleteRows > 0 {
		out = append(out, "Run prune_stale_chunks to remove superseded chunks per document chunk index.")
	}
	if report.IngestHistory != nil && report.IngestHistory.FailureRate > 0.1 {
		out = append(out, "Investigate ingest failures (Ollama/Qdrant reachability) before reingest.")
	}
	if len(report.VectorDimensions) > 1 {
		out = append(out, "Collection has mixed vector dimensions; verify embedding model and reingest for consistency.")
	}
	if report.MissingPayloadFields["namespace"] > 0 || report.MissingPayloadFields["document_id"] > 0 {
		out = append(out, "Payload metadata is incomplete for some points; ensure ingest writes namespace/document_id consistently.")
	}
	if len(out) == 0 {
		out = append(out, "No immediate issues detected from sampled points. Continue monitoring freshness and redundancy.")
	}
	return out
}

func staleChunkDeleteCandidates(points []qdrantPoint) []string {
	type pointMeta struct {
		id    string
		score int64
	}
	grouped := map[string][]pointMeta{}
	for _, point := range points {
		namespace := payloadString(point.Payload, "namespace")
		documentID := payloadString(point.Payload, "document_id")
		chunkIndex := payloadString(point.Payload, "chunk_index")
		if namespace == "" || documentID == "" || chunkIndex == "" {
			continue
		}
		key := namespace + "|" + documentID + "|" + chunkIndex
		grouped[key] = append(grouped[key], pointMeta{id: point.ID, score: payloadUnixMS(point.Payload)})
	}

	candidates := make([]string, 0, 128)
	for _, group := range grouped {
		if len(group) <= 1 {
			continue
		}
		sort.SliceStable(group, func(i, j int) bool { return group[i].score > group[j].score })
		for i := 1; i < len(group); i++ {
			candidates = append(candidates, group[i].id)
		}
	}
	return candidates
}

func duplicateContentDeleteCandidates(points []qdrantPoint) []string {
	type pointMeta struct {
		id    string
		score int64
	}
	grouped := map[string][]pointMeta{}
	for _, point := range points {
		namespace := payloadString(point.Payload, "namespace")
		contentHash := payloadString(point.Payload, "content_hash")
		if namespace == "" || contentHash == "" {
			continue
		}
		key := namespace + "|" + contentHash
		grouped[key] = append(grouped[key], pointMeta{id: point.ID, score: payloadUnixMS(point.Payload)})
	}

	candidates := make([]string, 0, 128)
	for _, group := range grouped {
		if len(group) <= 1 {
			continue
		}
		sort.SliceStable(group, func(i, j int) bool { return group[i].score > group[j].score })
		for i := 1; i < len(group); i++ {
			candidates = append(candidates, group[i].id)
		}
	}
	return candidates
}

func (s *Server) scrollCollectionPoints(ctx context.Context, collection string, maxPoints int, withVectors bool, filter interface{}) ([]qdrantPoint, error) {
	if maxPoints <= 0 {
		maxPoints = 500
	}
	if maxPoints > 100000 {
		maxPoints = 100000
	}

	baseURL, err := url.Parse(strings.TrimRight(s.qdrantURL(), "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	baseURL.Path = fmt.Sprintf("%s/collections/%s/points/scroll", strings.TrimRight(baseURL.Path, "/"), collection)

	pageSize := 512
	if maxPoints < pageSize {
		pageSize = maxPoints
	}

	points := make([]qdrantPoint, 0, maxPoints)
	var offset interface{}
	for len(points) < maxPoints {
		reqBody := qdrantScrollRequest{
			Limit:       pageSize,
			WithPayload: true,
			WithVectors: withVectors,
			Filter:      filter,
			Offset:      offset,
		}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scroll request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", baseURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create scroll request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.qdrantDo(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("qdrant scroll returned status %d", resp.StatusCode)
		}

		var decoded qdrantScrollResponse
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			return nil, fmt.Errorf("failed to decode scroll response: %w", err)
		}

		if len(decoded.Result.Points) == 0 {
			break
		}
		for _, point := range decoded.Result.Points {
			points = append(points, qdrantPoint{
				ID:      stringifyPointID(point.ID),
				Vector:  point.Vector,
				Payload: point.Payload,
			})
			if len(points) >= maxPoints {
				break
			}
		}

		if decoded.Result.NextPageOffset == nil || len(points) >= maxPoints {
			break
		}
		offset = decoded.Result.NextPageOffset
	}

	return points, nil
}

func (s *Server) deletePointsByIDBatch(ctx context.Context, collection string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	baseURL, err := url.Parse(strings.TrimRight(s.qdrantURL(), "/"))
	if err != nil {
		return fmt.Errorf("invalid qdrant url: %w", err)
	}
	baseURL.Path = fmt.Sprintf("%s/collections/%s/points/delete", strings.TrimRight(baseURL.Path, "/"), collection)
	query := baseURL.Query()
	query.Set("wait", "true")
	baseURL.RawQuery = query.Encode()

	const batchSize = 256
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		body, err := json.Marshal(map[string]interface{}{"points": batch})
		if err != nil {
			return fmt.Errorf("failed to marshal delete request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", baseURL.String(), bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create delete request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.qdrantDo(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("qdrant delete returned status %d", resp.StatusCode)
		}
	}
	return nil
}

func (s *Server) queryCollectionIngestHistory(ctx context.Context, collection string) (*collectionIngestHistoryHealth, error) {
	if s == nil || s.stores == nil {
		return nil, nil
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return nil, nil
	}

	health, err := s.stores.Ingest.HealthForCollection(ctx, collection)
	if err != nil {
		return nil, err
	}

	row := &collectionIngestHistoryHealth{
		TotalAttempts:       health.TotalAttempts,
		SuccessCount:        health.SuccessCount,
		FailureCount:        health.FailureCount,
		FailureCountLast24H: health.FailureCountLast24H,
	}
	row.FailureRate = ratio(row.FailureCount, row.TotalAttempts)
	if health.LastFailureAt != nil {
		row.LastFailureAt = health.LastFailureAt.Format(time.RFC3339)
	}
	return row, nil
}

func payloadString(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func payloadUnixMS(payload map[string]interface{}) int64 {
	if payload == nil {
		return 0
	}
	if raw, ok := payload["ingested_at_unix_ms"]; ok {
		switch value := raw.(type) {
		case float64:
			return int64(value)
		case int64:
			return value
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
				return parsed
			}
		case json.Number:
			if parsed, err := value.Int64(); err == nil {
				return parsed
			}
		}
	}
	if raw, ok := payload["ingested_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw)); err == nil {
			return parsed.UnixMilli()
		}
	}
	return 0
}

func stringifyPointID(id interface{}) string {
	switch v := id.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func dimensionCountsToSlice(counts map[int]int) []dimensionCount {
	out := make([]dimensionCount, 0, len(counts))
	for dimension, count := range counts {
		out = append(out, dimensionCount{Dimension: dimension, Count: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Dimension < out[j].Dimension
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func topCounts(values map[string]int, limit int) []namedCount {
	out := make([]namedCount, 0, len(values))
	for name, count := range values {
		out = append(out, namedCount{Name: name, Count: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func ratio(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
