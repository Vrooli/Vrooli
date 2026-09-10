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

const (
	// generationNameSeparator joins the alias and the generation ID in every
	// physical collection name, and is the prefix CleanupGenerations lists by.
	generationNameSeparator = "--generation--"
	// initFromCommitTimeout bounds one server-side candidate seed (Qdrant
	// init_from) and the client wait around it. A 65k-point hybrid collection
	// seeds in seconds; the bound exists so a wedged Qdrant fails the run
	// instead of hanging the indexer.
	initFromCommitTimeout = 5 * time.Minute
	// seedPollInterval is how often a seeded candidate is re-read while waiting
	// for its point count to reach the active generation's.
	seedPollInterval = 250 * time.Millisecond
)

// QdrantGenerationOptions configures an alias-backed GenerationStore. Alias is
// the stable collection name used by query traffic; immutable physical
// collections are created beneath it and switched with one Qdrant alias update.
type QdrantGenerationOptions struct {
	BaseURL   string
	APIKey    string
	Alias     string
	Spec      CollectionSpec
	Owner     string
	Namespace string
	Catalog   GenerationCatalog
	Client    interface {
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
	// promoted records generation IDs whose candidate has become the alias
	// target. CleanupGenerations must never delete an in-flight candidate
	// (begun, not yet promoted or rolled back), so it subtracts
	// candidates-minus-promoted from the retirement set.
	promoted map[string]struct{}
	// seedTimeout bounds awaitSeeded; tests shorten it.
	seedTimeout time.Duration
	owner       string
	namespace   string
	catalog     GenerationCatalog
	now         func() time.Time
	metricsMu   sync.Mutex
	metrics     GenerationLifecycleMetrics
	cleanupErrs uint64
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
		options.Client = &http.Client{Timeout: initFromCommitTimeout + 30*time.Second}
	}
	owner := strings.TrimSpace(options.Owner)
	if owner == "" {
		owner = "agent-manager"
	}
	namespace := strings.TrimSpace(options.Namespace)
	if namespace == "" {
		namespace = options.Alias
	}
	return &qdrantGenerationStore{baseURL: strings.TrimRight(options.BaseURL, "/"), apiKey: options.APIKey, alias: options.Alias, spec: options.Spec, client: options.Client, candidates: map[string]string{}, promoted: map[string]struct{}{}, seedTimeout: initFromCommitTimeout, owner: owner, namespace: namespace, catalog: options.Catalog, now: time.Now}, nil
}

func (s *qdrantGenerationStore) BeginGeneration(ctx context.Context, metadata GenerationMetadata) error {
	if strings.TrimSpace(metadata.ID) == "" {
		return errors.New("generation id is required")
	}
	metadata.Owner = lifecycleFirstNonEmpty(metadata.Owner, s.owner)
	metadata.Namespace = lifecycleFirstNonEmpty(metadata.Namespace, s.namespace)
	metadata.Alias = lifecycleFirstNonEmpty(metadata.Alias, s.alias)
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = s.clock()
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
	s.mu.Lock()
	s.candidates[metadata.ID] = candidate
	delete(s.promoted, metadata.ID)
	s.mu.Unlock()
	if err := s.saveRecord(ctx, GenerationRecord{Metadata: metadata, CollectionName: candidate, State: GenerationStateCandidate, UpdatedAt: s.clock()}); err != nil {
		return fmt.Errorf("record candidate generation: %w", err)
	}
	if metadata.Full || active == "" {
		if err := store.EnsureCollection(ctx, spec); err != nil {
			_ = s.markGenerationState(ctx, metadata.ID, GenerationStateFailed, "candidate creation failed")
			return fmt.Errorf("create candidate collection: %w", err)
		}
		return nil
	}
	// Incremental generations start as a copy of the active one. Prefer the
	// server-side seed: Qdrant copies dense, sparse and payload in place, so
	// the client never scrolls the whole corpus out and upserts it back in.
	if initializer, ok := store.(CollectionInitializer); ok {
		if err := initializer.EnsureCollectionFrom(ctx, spec, active); err != nil {
			return s.abandonCandidate(ctx, metadata.ID, fmt.Errorf("create candidate collection from %q: %w", active, err))
		}
		if err := s.awaitSeeded(ctx, candidate, active); err != nil {
			return s.abandonCandidate(ctx, metadata.ID, err)
		}
		return nil
	}
	if err := store.EnsureCollection(ctx, spec); err != nil {
		return fmt.Errorf("create candidate collection: %w", err)
	}
	points, err := s.scrollPoints(ctx, active)
	if err != nil {
		return s.abandonCandidate(ctx, metadata.ID, fmt.Errorf("copy active generation: %w", err))
	}
	if err := store.(BatchVectorStore).UpsertBatch(ctx, points, DefaultSourcePageSize); err != nil {
		return s.abandonCandidate(ctx, metadata.ID, fmt.Errorf("seed candidate generation: %w", err))
	}
	return nil
}

// abandonCandidate drops a candidate whose seeding failed and returns cause.
// The reconciler only registers its rollback after BeginGeneration succeeds,
// so a half-seeded candidate would otherwise survive as an orphan holding a
// full copy of the corpus in Qdrant's RAM.
func (s *qdrantGenerationStore) abandonCandidate(ctx context.Context, generationID string, cause error) error {
	if err := s.RollbackGeneration(context.WithoutCancel(ctx), generationID); err != nil {
		return errors.Join(cause, fmt.Errorf("abandon half-seeded candidate: %w", err))
	}
	return cause
}

// awaitSeeded blocks until the seeded candidate holds at least as many points
// as the active generation it was copied from, or the bound elapses. The
// active generation is immutable (writes only ever touch candidates), so its
// count is a stable target.
func (s *qdrantGenerationStore) awaitSeeded(ctx context.Context, candidate, active string) error {
	want, err := s.collectionPointCount(ctx, active)
	if err != nil {
		return fmt.Errorf("read active generation size: %w", err)
	}
	deadline := time.Now().Add(s.seedTimeout)
	for {
		got, err := s.collectionPointCount(ctx, candidate)
		if err != nil {
			return fmt.Errorf("read seeded candidate size: %w", err)
		}
		if got >= want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("seeded candidate %q holds %d of %d points after %s", candidate, got, want, s.seedTimeout)
		}
		timer := time.NewTimer(seedPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *qdrantGenerationStore) collectionPointCount(ctx context.Context, collection string) (int, error) {
	var response struct {
		Result struct {
			PointsCount *int `json:"points_count"`
		} `json:"result"`
	}
	if err := s.readJSON(ctx, s.baseURL+"/collections/"+url.PathEscape(collection), &response, http.StatusOK); err != nil {
		return 0, err
	}
	if response.Result.PointsCount == nil {
		return 0, nil
	}
	return *response.Result.PointsCount, nil
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
	previous, hasAlias, err := s.activeCollection(ctx)
	if err != nil {
		return err
	}
	actions := make([]map[string]any, 0, 2)
	if hasAlias {
		actions = append(actions, map[string]any{"delete_alias": map[string]any{"alias_name": s.alias}})
	}
	actions = append(actions, map[string]any{"create_alias": map[string]any{"collection_name": candidate, "alias_name": s.alias}})
	if err := s.writeJSON(ctx, http.MethodPost, s.baseURL+"/collections/aliases", map[string]any{"actions": actions}, http.StatusOK); err != nil {
		return err
	}
	s.mu.Lock()
	s.promoted[generationID] = struct{}{}
	s.mu.Unlock()
	if s.catalog != nil {
		if previous != "" && previous != candidate {
			_ = s.saveRecord(ctx, GenerationRecord{Metadata: GenerationMetadata{ID: generationIDFromCollection(previous), Owner: s.owner, Namespace: s.namespace, Alias: s.alias}, CollectionName: previous, State: GenerationStateRetired, UpdatedAt: s.clock()})
		}
		if record, found, loadErr := s.loadRecord(ctx, generationID); loadErr == nil && found {
			record.State = GenerationStateActive
			record.UpdatedAt = s.clock()
			if saveErr := s.saveRecord(ctx, record); saveErr != nil {
				return saveErr
			}
		}
	}
	return nil
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
		if record, found, loadErr := s.loadRecord(ctx, generationID); loadErr == nil && found {
			record.State = GenerationStateDeleted
			record.CleanupOutcome = "rollback"
			record.UpdatedAt = s.clock()
			_ = s.saveRecord(ctx, record)
		}
	}
	return err
}

// CleanupGenerations deletes retired generations beyond the newest keep,
// one collection per request. A generation is retired when its physical
// collection carries this alias's generation prefix, is not the alias target,
// and is not an in-flight candidate of this store. Retired generations are
// ordered by collection name, newest first, so generation IDs must sort
// chronologically (Agent Manager's conversation-<unixnano>-<seq> does).
// Untracked generations left by an earlier process count as retired: nothing
// but the alias target is load-bearing after promotion, and every retired
// copy holds its whole HNSW graph in Qdrant's RAM (the 2026-09-09 incident
// accumulated 1,301 of them: 445 GB on disk and 56 GB resident).
// Deletion failures are collected and returned together after every
// retirable collection has been attempted.
func (s *qdrantGenerationStore) CleanupGenerations(ctx context.Context, keep int) error {
	if keep < 0 {
		keep = 0
	}
	retired, err := s.retiredGenerations(ctx)
	if err != nil {
		return err
	}
	if len(retired) <= keep {
		return nil
	}
	var failures []error
	for _, name := range retired[keep:] {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		endpoint := s.baseURL + "/collections/" + url.PathEscape(name)
		if err := s.writeJSON(ctx, http.MethodDelete, endpoint, nil, http.StatusOK); err != nil {
			failures = append(failures, fmt.Errorf("delete retired generation %q: %w", name, err))
		}
	}
	return errors.Join(failures...)
}

// retiredGenerations lists this alias's retired physical collections, newest
// first by name.
func (s *qdrantGenerationStore) retiredGenerations(ctx context.Context) ([]string, error) {
	names, err := s.listCollections(ctx)
	if err != nil {
		return nil, err
	}
	active, _, err := s.activeCollection(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	inflight := make(map[string]struct{}, len(s.candidates))
	for id, candidate := range s.candidates {
		if _, done := s.promoted[id]; !done {
			inflight[candidate] = struct{}{}
		}
	}
	s.mu.Unlock()
	prefix := s.alias + generationNameSeparator
	retired := make([]string, 0, len(names))
	for _, name := range names {
		if !strings.HasPrefix(name, prefix) || name == active {
			continue
		}
		if _, ok := inflight[name]; ok {
			continue
		}
		retired = append(retired, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(retired)))
	return retired, nil
}

func (s *qdrantGenerationStore) listCollections(ctx context.Context) ([]string, error) {
	var response struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	if err := s.readJSON(ctx, s.baseURL+"/collections", &response, http.StatusOK); err != nil {
		return nil, fmt.Errorf("list qdrant collections: %w", err)
	}
	names := make([]string, 0, len(response.Result.Collections))
	for _, collection := range response.Result.Collections {
		names = append(names, collection.Name)
	}
	return names, nil
}

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

func (s *qdrantGenerationStore) clock() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *qdrantGenerationStore) saveRecord(ctx context.Context, record GenerationRecord) error {
	if s.catalog == nil {
		return nil
	}
	return s.catalog.SaveGenerationRecord(ctx, record)
}

func (s *qdrantGenerationStore) loadRecord(ctx context.Context, generationID string) (GenerationRecord, bool, error) {
	if s.catalog == nil {
		return GenerationRecord{}, false, nil
	}
	records, err := s.catalog.ListGenerationRecords(ctx, s.namespace, s.alias)
	if err != nil {
		return GenerationRecord{}, false, err
	}
	for _, record := range records {
		if record.Metadata.ID == generationID || generationIDFromCollection(record.CollectionName) == generationID {
			return record, true, nil
		}
	}
	return GenerationRecord{}, false, nil
}

// Metrics returns a bounded point-in-time lifecycle metric projection.
func (s *qdrantGenerationStore) Metrics() GenerationLifecycleMetrics {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	return s.metrics
}

func (s *qdrantGenerationStore) InspectGenerationLifecycle(ctx context.Context, policy GenerationRetentionPolicy) (GenerationInspection, error) {
	if err := validateCleanupPolicy(policy); err != nil {
		return GenerationInspection{}, err
	}
	now := s.clock()
	active, _, err := s.activeCollection(ctx)
	if err != nil {
		return GenerationInspection{}, err
	}
	names, err := s.listCollections(ctx)
	if err != nil {
		return GenerationInspection{}, err
	}
	var records []GenerationRecord
	if s.catalog != nil {
		records, err = s.catalog.ListGenerationRecords(ctx, s.namespace, s.alias)
		if err != nil {
			return GenerationInspection{}, fmt.Errorf("list generation catalog: %w", err)
		}
	}
	byCollection := make(map[string]GenerationRecord, len(records))
	for _, record := range records {
		byCollection[record.CollectionName] = record
	}
	s.mu.Lock()
	inflight := make(map[string]struct{}, len(s.candidates))
	for id, collection := range s.candidates {
		if _, promoted := s.promoted[id]; !promoted {
			inflight[collection] = struct{}{}
		}
		if _, exists := byCollection[collection]; !exists {
			byCollection[collection] = GenerationRecord{Metadata: GenerationMetadata{ID: id, Owner: s.owner, Namespace: s.namespace, Alias: s.alias}, CollectionName: collection, State: GenerationStateCandidate, UpdatedAt: now}
		}
	}
	s.mu.Unlock()

	prefix := s.alias + generationNameSeparator
	inspection := GenerationInspection{Owner: s.owner, Namespace: s.namespace, Alias: s.alias, ActiveCollection: active, ObservedAt: now}
	for _, collection := range names {
		if !strings.HasPrefix(collection, prefix) {
			continue
		}
		record, known := byCollection[collection]
		if !known {
			record = GenerationRecord{Metadata: GenerationMetadata{ID: generationIDFromCollection(collection), Owner: s.owner, Namespace: s.namespace, Alias: s.alias}, CollectionName: collection, State: GenerationStateQuarantined}
		}
		record.Metadata.ID = lifecycleFirstNonEmpty(record.Metadata.ID, generationIDFromCollection(collection))
		record.Metadata.Owner = lifecycleFirstNonEmpty(record.Metadata.Owner, s.owner)
		record.Metadata.Namespace = lifecycleFirstNonEmpty(record.Metadata.Namespace, s.namespace)
		record.Metadata.Alias = lifecycleFirstNonEmpty(record.Metadata.Alias, s.alias)
		record.CollectionName = collection
		if record.Metadata.CreatedAt.IsZero() && record.State != GenerationStateQuarantined {
			record.Metadata.CreatedAt = record.UpdatedAt
		}
		if record.UpdatedAt.IsZero() && record.State != GenerationStateQuarantined {
			record.UpdatedAt = record.Metadata.CreatedAt
		}
		if collection == active {
			record.State = GenerationStateActive
		} else if record.State == GenerationStateActive {
			// The alias is the source of truth for the serving generation. A
			// restart or an older catalog writer can leave more than one record
			// marked active even though Qdrant serves exactly one collection.
			// Normalize that stale state to retired so rollback and retention
			// policy can evaluate the physical collection safely.
			record.State = GenerationStateRetired
		} else if record.State == "" {
			record.State = GenerationStateQuarantined
		}
		if record.State == GenerationStateRetired && now.Sub(record.Metadata.CreatedAt) >= policy.RetiredMaxAge {
			record.State = GenerationStateExpired
		}
		if record.State == GenerationStateFailed && now.Sub(record.Metadata.CreatedAt) >= policy.FailedMaxAge {
			record.State = GenerationStateExpired
		}
		if record.State == GenerationStateCandidate && !record.Lease.Active(now) && now.Sub(record.Metadata.CreatedAt) >= policy.FailedMaxAge {
			record.State = GenerationStateFailed
		}
		if count, countErr := s.collectionPointCount(ctx, collection); countErr == nil {
			record.Points = count
		}
		inspection.Generations = append(inspection.Generations, record)
	}
	sort.Slice(inspection.Generations, func(i, j int) bool {
		return inspection.Generations[i].Metadata.CreatedAt.After(inspection.Generations[j].Metadata.CreatedAt)
	})
	protectedRollback := 0
	for _, record := range inspection.Generations {
		_, locallyInflight := inflight[record.CollectionName]
		protected := record.CollectionName == active || (record.State == GenerationStateCandidate && locallyInflight) || record.Lease.Active(now)
		catalogState := byCollection[record.CollectionName].State
		rollbackEligible := record.State == GenerationStateRetired || (record.State == GenerationStateExpired && catalogState == GenerationStateRetired)
		if !protected && protectedRollback < policy.KeepRollback && rollbackEligible {
			protected = true
			protectedRollback++
		}
		if protected {
			inspection.Protected = append(inspection.Protected, record.CollectionName)
			continue
		}
		if record.State == GenerationStateQuarantined {
			inspection.Quarantined = append(inspection.Quarantined, record.CollectionName)
			continue
		}
		if record.State == GenerationStateExpired || record.State == GenerationStateFailed {
			inspection.Eligible = append(inspection.Eligible, record.CollectionName)
		}
		inspection.LogicalBytes += record.Bytes
	}
	if len(inspection.Generations) > policy.MaxGenerationCount {
		for _, record := range inspection.Generations[policy.MaxGenerationCount:] {
			if !containsString(inspection.Protected, record.CollectionName) && !containsString(inspection.Quarantined, record.CollectionName) && !containsString(inspection.Eligible, record.CollectionName) {
				inspection.Eligible = append(inspection.Eligible, record.CollectionName)
			}
		}
	}
	inspection.SnapshotIdentity = cleanupSnapshotIdentity(s.namespace, s.alias, policy.PolicyRevision, inspection.Generations)
	s.metricsMu.Lock()
	s.metrics.GenerationCount = len(inspection.Generations)
	s.metrics.RetiredCount = 0
	s.metrics.RetiredBytes = 0
	s.metrics.QuarantineCount = len(inspection.Quarantined)
	s.metrics.UpdatedAt = now
	if len(inspection.Generations) > 0 {
		s.metrics.OldestGeneration = inspection.Generations[len(inspection.Generations)-1].Metadata.CreatedAt
	}
	for _, record := range inspection.Generations {
		if record.State == GenerationStateRetired || record.State == GenerationStateExpired {
			s.metrics.RetiredCount++
			s.metrics.RetiredBytes += record.Bytes
		}
	}
	s.metricsMu.Unlock()
	return inspection, nil
}

func (s *qdrantGenerationStore) PreviewGenerationCleanup(ctx context.Context, policy GenerationRetentionPolicy, planIdentity string) (GenerationCleanupPlan, error) {
	if err := validateCleanupPolicy(policy); err != nil {
		return GenerationCleanupPlan{}, err
	}
	if strings.TrimSpace(planIdentity) == "" {
		return GenerationCleanupPlan{}, errors.New("cleanup plan identity is required")
	}
	inspection, err := s.InspectGenerationLifecycle(ctx, policy)
	if err != nil {
		return GenerationCleanupPlan{}, err
	}
	if inspection.ActiveCollection == "" {
		return GenerationCleanupPlan{}, errors.New("cannot preview cleanup while the active generation alias is unavailable")
	}
	byCollection := make(map[string]GenerationRecord, len(inspection.Generations))
	for _, record := range inspection.Generations {
		byCollection[record.CollectionName] = record
	}
	plan := GenerationCleanupPlan{PlanIdentity: planIdentity, IdempotencyKey: planIdentity, PolicyRevision: policy.PolicyRevision, Policy: policy, SnapshotIdentity: inspection.SnapshotIdentity, Namespace: s.namespace, Alias: s.alias, Protected: append([]string(nil), inspection.Protected...), Quarantined: append([]string(nil), inspection.Quarantined...), Approved: false, CreatedAt: s.clock()}
	for _, collection := range inspection.Eligible {
		record := byCollection[collection]
		plan.Candidates = append(plan.Candidates, GenerationCleanupCandidate{GenerationID: record.Metadata.ID, CollectionName: collection, State: record.State, Bytes: record.Bytes, Reason: "policy eligible and not protected"})
	}
	return plan, nil
}

func (s *qdrantGenerationStore) ApplyGenerationCleanup(ctx context.Context, plan GenerationCleanupPlan) (GenerationCleanupReceipt, error) {
	if !plan.Approved {
		return GenerationCleanupReceipt{}, errors.New("cleanup plan requires explicit approval")
	}
	if strings.TrimSpace(plan.IdempotencyKey) == "" || strings.TrimSpace(plan.PolicyRevision) == "" || strings.TrimSpace(plan.SnapshotIdentity) == "" {
		return GenerationCleanupReceipt{}, errors.New("cleanup plan is missing idempotency key, policy revision, or snapshot identity")
	}
	if stored, ok, err := s.loadReceipt(ctx, plan.IdempotencyKey); err != nil {
		return GenerationCleanupReceipt{}, err
	} else if ok {
		return stored, nil
	}
	policy := plan.Policy
	if policy.PolicyRevision == "" {
		policy = DefaultGenerationRetentionPolicy()
		policy.PolicyRevision = plan.PolicyRevision
	}
	inspection, err := s.InspectGenerationLifecycle(ctx, policy)
	if err != nil {
		return GenerationCleanupReceipt{}, err
	}
	if inspection.SnapshotIdentity != plan.SnapshotIdentity || inspection.ActiveCollection == "" {
		return GenerationCleanupReceipt{}, errors.New("cleanup plan is stale or alias target is unavailable")
	}
	receipt := GenerationCleanupReceipt{PlanIdentity: plan.PlanIdentity, IdempotencyKey: plan.IdempotencyKey, PolicyRevision: plan.PolicyRevision, ActiveCollection: inspection.ActiveCollection, QuarantinedCollections: append([]string(nil), inspection.Quarantined...), CompletedAt: s.clock()}
	for offset := 0; offset < len(plan.Candidates); offset += policy.DeletionBatchSize {
		end := offset + policy.DeletionBatchSize
		if end > len(plan.Candidates) {
			end = len(plan.Candidates)
		}
		fresh, freshErr := s.InspectGenerationLifecycle(ctx, policy)
		if freshErr != nil {
			return GenerationCleanupReceipt{}, freshErr
		}
		if fresh.ActiveCollection != inspection.ActiveCollection {
			return GenerationCleanupReceipt{}, errors.New("active generation changed during cleanup")
		}
		for _, candidate := range plan.Candidates[offset:end] {
			if containsString(fresh.Protected, candidate.CollectionName) || containsString(fresh.Quarantined, candidate.CollectionName) {
				receipt.SkippedCollections = append(receipt.SkippedCollections, candidate.CollectionName)
				continue
			}
			if err := s.deleteGenerationCollection(ctx, candidate.CollectionName); err != nil {
				s.metricsMu.Lock()
				s.cleanupErrs++
				s.metrics.CleanupErrors = s.cleanupErrs
				s.metricsMu.Unlock()
				return GenerationCleanupReceipt{}, err
			}
			receipt.DeletedCollections = append(receipt.DeletedCollections, candidate.CollectionName)
			receipt.LogicalReclaimedBytes += candidate.Bytes
			if record, found, loadErr := s.loadRecord(ctx, candidate.GenerationID); loadErr == nil && found {
				record.State = GenerationStateDeleted
				record.CleanupOutcome = "owner-approved cleanup"
				record.UpdatedAt = s.clock()
				_ = s.saveRecord(ctx, record)
			}
		}
	}
	if err := s.saveReceipt(ctx, receipt); err != nil {
		return GenerationCleanupReceipt{}, err
	}
	return receipt, nil
}

func (s *qdrantGenerationStore) deleteGenerationCollection(ctx context.Context, collection string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.baseURL+"/collections/"+url.PathEscape(collection), nil)
	if err != nil {
		return err
	}
	resp, err := s.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("delete qdrant collection %q returned status %d: %s", collection, resp.StatusCode, strings.TrimSpace(string(raw)))
}

func (s *qdrantGenerationStore) markGenerationState(ctx context.Context, generationID string, state GenerationLifecycleState, outcome string) error {
	record, found, err := s.loadRecord(ctx, generationID)
	if err != nil || !found {
		return err
	}
	record.State = state
	record.CleanupOutcome = outcome
	record.UpdatedAt = s.clock()
	return s.saveRecord(ctx, record)
}

func (s *qdrantGenerationStore) loadReceipt(ctx context.Context, key string) (GenerationCleanupReceipt, bool, error) {
	if store, ok := s.catalog.(GenerationCleanupReceiptStore); ok {
		return store.LoadGenerationCleanupReceipt(ctx, key)
	}
	return GenerationCleanupReceipt{}, false, nil
}

func (s *qdrantGenerationStore) saveReceipt(ctx context.Context, receipt GenerationCleanupReceipt) error {
	if store, ok := s.catalog.(GenerationCleanupReceiptStore); ok {
		return store.SaveGenerationCleanupReceipt(ctx, receipt)
	}
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func generationCollectionName(alias, id string) string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, id)
	name := alias + generationNameSeparator + clean
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

var _ GenerationStore = (*qdrantGenerationStore)(nil)
