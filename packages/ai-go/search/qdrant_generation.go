package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrGenerationAliasConflict = errors.New("aisearch: generation alias conflicts with an existing physical collection")

// QdrantGenerationOptions configures an alias-backed GenerationStore. Alias is
// the stable collection name used by query traffic; immutable physical
// collections are created beneath it and switched with one Qdrant alias update.
type QdrantGenerationOptions struct {
	BaseURL string
	APIKey  string
	Alias   string
	Spec    CollectionSpec
	Client  interface {
		Do(*http.Request) (*http.Response, error)
	}
}

type qdrantGenerationStore struct {
	baseURL string
	apiKey  string
	alias   string
	spec    CollectionSpec
	client  interface {
		Do(*http.Request) (*http.Response, error)
	}
	mu         sync.Mutex
	candidates map[string]string
}

func NewQdrantGenerationStore(options QdrantGenerationOptions) (GenerationStore, error) {
	if strings.TrimSpace(options.Alias) == "" {
		return nil, errors.New("qdrant generation alias is required")
	}
	if strings.TrimSpace(options.BaseURL) == "" {
		options.BaseURL = DefaultQdrantURL
	}
	if options.Spec.DenseSize <= 0 {
		return nil, errors.New("qdrant generation collection spec is required")
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 30 * time.Second}
	}
	return &qdrantGenerationStore{baseURL: strings.TrimRight(options.BaseURL, "/"), apiKey: options.APIKey, alias: options.Alias, spec: options.Spec, client: options.Client, candidates: map[string]string{}}, nil
}

func (s *qdrantGenerationStore) BeginGeneration(ctx context.Context, metadata GenerationMetadata) error {
	if strings.TrimSpace(metadata.ID) == "" {
		return errors.New("generation id is required")
	}
	active, hasAlias, err := s.activeCollection(ctx)
	if err != nil {
		return err
	}
	if !hasAlias {
		found, inspectErr := s.physicalCollectionExists(ctx, s.alias)
		if inspectErr != nil {
			return inspectErr
		}
		if found {
			return fmt.Errorf("%w: %q; migrate it explicitly before enabling shadow generations", ErrGenerationAliasConflict, s.alias)
		}
	}
	candidate := generationCollectionName(s.alias, metadata.ID)
	spec := s.spec
	spec.Name = candidate
	store := NewVectorStoreWithClient(s.baseURL, s.apiKey, candidate, s.client)
	if err := store.EnsureCollection(ctx, spec); err != nil {
		return fmt.Errorf("create candidate collection: %w", err)
	}
	s.mu.Lock()
	s.candidates[metadata.ID] = candidate
	s.mu.Unlock()
	if !metadata.Full && active != "" {
		points, err := s.scrollPoints(ctx, active)
		if err != nil {
			return fmt.Errorf("copy active generation: %w", err)
		}
		if err := store.(BatchVectorStore).UpsertBatch(ctx, points, DefaultSourcePageSize); err != nil {
			return fmt.Errorf("seed candidate generation: %w", err)
		}
	}
	return nil
}

func (s *qdrantGenerationStore) LookupActiveSources(ctx context.Context, sourceIDs []string) (map[string]StoredSourceState, error) {
	active, ok, err := s.activeCollection(ctx)
	if err != nil || !ok {
		return map[string]StoredSourceState{}, err
	}
	return s.lookupCollectionSources(ctx, active, sourceIDs)
}

func (s *qdrantGenerationStore) LookupGenerationSources(ctx context.Context, generationID string, sourceIDs []string) (map[string]StoredSourceState, error) {
	candidate, err := s.candidate(generationID)
	if err != nil {
		return nil, err
	}
	return s.lookupCollectionSources(ctx, candidate, sourceIDs)
}

func (s *qdrantGenerationStore) lookupCollectionSources(ctx context.Context, collection string, sourceIDs []string) (map[string]StoredSourceState, error) {
	if len(sourceIDs) == 0 {
		return map[string]StoredSourceState{}, nil
	}
	store := NewVectorStoreWithClient(s.baseURL, s.apiKey, collection, s.client)
	qdrant, ok := store.(*qdrantVectorStore)
	if !ok {
		return nil, errors.New("generation source lookup requires qdrant vector store")
	}
	items, err := qdrant.scrollIDsForSources(ctx, sourceIDs)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]struct{}, len(sourceIDs))
	for _, id := range sourceIDs {
		wanted[id] = struct{}{}
	}
	out := make(map[string]StoredSourceState)
	for id, item := range items {
		if _, ok := wanted[item.SourceID]; !ok {
			continue
		}
		state := out[item.SourceID]
		state.SourceHash = item.SourceHash
		if state.Points == nil {
			state.Points = map[string]ScrollItem{}
		}
		state.Points[id] = item
		out[item.SourceID] = state
	}
	return out, nil
}

func (s *qdrantGenerationStore) StageSource(ctx context.Context, generationID string, write GenerationSourceWrite) error {
	return s.StageSources(ctx, generationID, []GenerationSourceWrite{write})
}

func (s *qdrantGenerationStore) StageSources(ctx context.Context, generationID string, writes []GenerationSourceWrite) error {
	if len(writes) == 0 {
		return nil
	}
	candidate, err := s.candidate(generationID)
	if err != nil {
		return err
	}
	store := NewVectorStoreWithClient(s.baseURL, s.apiKey, candidate, s.client)
	sourceIDs := make([]string, 0, len(writes))
	var reuseIDs []string
	points := make([]Point, 0, len(writes))
	for _, write := range writes {
		if strings.TrimSpace(write.SourceID) == "" {
			return errors.New("candidate source id is required")
		}
		sourceIDs = append(sourceIDs, write.SourceID)
		reuseIDs = append(reuseIDs, write.ReusePointIDs...)
		points = append(points, write.ChangedPoints...)
	}
	if err := retryGenerationWrite(ctx, func() error { return s.deleteCandidateSources(ctx, candidate, sourceIDs) }); err != nil {
		return err
	}
	if len(reuseIDs) > 0 {
		active, ok, activeErr := s.activeCollection(ctx)
		if activeErr != nil {
			return activeErr
		}
		if !ok {
			return errors.New("cannot reuse points without an active generation")
		}
		reused, fetchErr := s.fetchPoints(ctx, active, reuseIDs)
		if fetchErr != nil {
			return fetchErr
		}
		if len(reused) != len(reuseIDs) {
			return fmt.Errorf("requested %d reusable points, found %d", len(reuseIDs), len(reused))
		}
		points = append(points, reused...)
	}
	if len(points) == 0 {
		return nil
	}
	return retryGenerationWrite(ctx, func() error {
		return store.(BatchVectorStore).UpsertBatch(ctx, points, DefaultSourcePageSize)
	})
}

func retryGenerationWrite(ctx context.Context, operation func() error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = operation(); err == nil {
			return nil
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ctx.Err(), err)
		case <-timer.C:
		}
	}
	return err
}

func (s *qdrantGenerationStore) deleteCandidateSources(ctx context.Context, collection string, sourceIDs []string) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	body := map[string]any{"filter": map[string]any{"must": []any{
		map[string]any{"key": sourceIDKey, "match": map[string]any{"any": sourceIDs}},
	}}}
	endpoint := s.baseURL + "/collections/" + url.PathEscape(collection) + "/points/delete?wait=true"
	return s.writeJSON(ctx, http.MethodPost, endpoint, body, http.StatusOK)
}

func (s *qdrantGenerationStore) StageDelete(ctx context.Context, generationID, sourceID string) error {
	return s.StageSource(ctx, generationID, GenerationSourceWrite{SourceID: sourceID})
}

func (s *qdrantGenerationStore) ValidateGeneration(ctx context.Context, generationID string) (GenerationValidation, error) {
	candidate, err := s.candidate(generationID)
	if err != nil {
		return GenerationValidation{}, err
	}
	items, err := NewVectorStoreWithClient(s.baseURL, s.apiKey, candidate, s.client).ScrollIDs(ctx)
	if err != nil {
		return GenerationValidation{}, err
	}
	sources := map[string]struct{}{}
	for _, item := range items {
		if strings.TrimSpace(item.SourceID) == "" {
			return GenerationValidation{PointCount: len(items), Valid: false, Detail: "point missing source identity"}, nil
		}
		sources[item.SourceID] = struct{}{}
	}
	return GenerationValidation{SourceCount: len(sources), PointCount: len(items), Valid: true}, nil
}

func (s *qdrantGenerationStore) PromoteGeneration(ctx context.Context, generationID string) error {
	candidate, err := s.candidate(generationID)
	if err != nil {
		return err
	}
	_, hasAlias, err := s.activeCollection(ctx)
	if err != nil {
		return err
	}
	actions := make([]map[string]any, 0, 2)
	if hasAlias {
		actions = append(actions, map[string]any{"delete_alias": map[string]any{"alias_name": s.alias}})
	}
	actions = append(actions, map[string]any{"create_alias": map[string]any{"collection_name": candidate, "alias_name": s.alias}})
	return s.writeJSON(ctx, http.MethodPost, s.baseURL+"/collections/aliases", map[string]any{"actions": actions}, http.StatusOK)
}

func (s *qdrantGenerationStore) RollbackGeneration(ctx context.Context, generationID string) error {
	candidate, err := s.candidate(generationID)
	if err != nil {
		return nil
	}
	active, _, activeErr := s.activeCollection(ctx)
	if activeErr != nil {
		return activeErr
	}
	if candidate == active {
		return nil
	}
	err = s.writeJSON(ctx, http.MethodDelete, s.baseURL+"/collections/"+url.PathEscape(candidate), nil, http.StatusOK)
	if err == nil {
		s.mu.Lock()
		delete(s.candidates, generationID)
		s.mu.Unlock()
	}
	return err
}

// CleanupGenerations is intentionally conservative. Candidate rollback removes
// failed shadows, while retired successful generations remain available for an
// explicit operator rollback; this method never turns retention into an
// automatic data-loss policy.
func (s *qdrantGenerationStore) CleanupGenerations(context.Context, int) error { return nil }

func (s *qdrantGenerationStore) candidate(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	candidate := s.candidates[id]
	if candidate == "" {
		candidate = generationCollectionName(s.alias, id)
		s.candidates[id] = candidate
	}
	return candidate, nil
}

func (s *qdrantGenerationStore) activeCollection(ctx context.Context) (string, bool, error) {
	var response struct {
		Result struct {
			Aliases []struct {
				AliasName      string `json:"alias_name"`
				CollectionName string `json:"collection_name"`
			} `json:"aliases"`
		} `json:"result"`
	}
	if err := s.readJSON(ctx, s.baseURL+"/aliases", &response, http.StatusOK); err != nil {
		return "", false, err
	}
	for _, alias := range response.Result.Aliases {
		if alias.AliasName == s.alias {
			return alias.CollectionName, true, nil
		}
	}
	return "", false, nil
}

func (s *qdrantGenerationStore) physicalCollectionExists(ctx context.Context, name string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/collections/"+url.PathEscape(name), nil)
	if err != nil {
		return false, err
	}
	resp, err := s.do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("inspect qdrant collection: status %d", resp.StatusCode)
	}
	return true, nil
}

type generationPoint struct {
	ID      any                        `json:"id"`
	Vector  map[string]json.RawMessage `json:"vector"`
	Payload map[string]any             `json:"payload"`
}

func (s *qdrantGenerationStore) fetchPoints(ctx context.Context, collection string, ids []string) ([]Point, error) {
	var response struct {
		Result []generationPoint `json:"result"`
	}
	body := map[string]any{"ids": ids, "with_payload": true, "with_vector": true}
	if err := s.exchangeJSON(ctx, http.MethodPost, s.baseURL+"/collections/"+url.PathEscape(collection)+"/points", body, &response); err != nil {
		return nil, err
	}
	return decodeGenerationPoints(response.Result)
}

func (s *qdrantGenerationStore) scrollPoints(ctx context.Context, collection string) ([]Point, error) {
	var out []Point
	var offset any
	for {
		var response struct {
			Result struct {
				Points []generationPoint `json:"points"`
				Next   any               `json:"next_page_offset"`
			} `json:"result"`
		}
		body := map[string]any{"limit": MaxSourcePageSize, "with_payload": true, "with_vector": true}
		if offset != nil {
			body["offset"] = offset
		}
		if err := s.exchangeJSON(ctx, http.MethodPost, s.baseURL+"/collections/"+url.PathEscape(collection)+"/points/scroll", body, &response); err != nil {
			return nil, err
		}
		points, err := decodeGenerationPoints(response.Result.Points)
		if err != nil {
			return nil, err
		}
		out = append(out, points...)
		if response.Result.Next == nil {
			break
		}
		offset = response.Result.Next
	}
	return out, nil
}

func decodeGenerationPoints(input []generationPoint) ([]Point, error) {
	out := make([]Point, 0, len(input))
	for _, raw := range input {
		if _, meta := raw.Payload[metaMarkerKey]; meta {
			continue
		}
		point := Point{ID: stringifyID(raw.ID), Payload: raw.Payload}
		if err := json.Unmarshal(raw.Vector[denseVectorName], &point.Dense); err != nil {
			return nil, fmt.Errorf("decode dense vector: %w", err)
		}
		if sparseRaw, ok := raw.Vector[sparseVectorName]; ok {
			var sparse sparseVectorJSON
			if err := json.Unmarshal(sparseRaw, &sparse); err != nil {
				return nil, err
			}
			point.Sparse = &SparseVector{Indices: sparse.Indices, Values: sparse.Values}
		}
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *qdrantGenerationStore) exchangeJSON(ctx context.Context, method, endpoint string, body any, output any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func (s *qdrantGenerationStore) readJSON(ctx context.Context, endpoint string, output any, want int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := s.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		return fmt.Errorf("qdrant request returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func (s *qdrantGenerationStore) writeJSON(ctx context.Context, method, endpoint string, body any, want int) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (s *qdrantGenerationStore) do(req *http.Request) (*http.Response, error) {
	if strings.TrimSpace(s.apiKey) != "" {
		req.Header.Set("api-key", s.apiKey)
	}
	return s.client.Do(req)
}

func generationCollectionName(alias, id string) string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, id)
	name := alias + "--generation--" + clean
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

var _ GenerationStore = (*qdrantGenerationStore)(nil)
