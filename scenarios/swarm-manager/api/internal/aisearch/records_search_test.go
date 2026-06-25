package aisearch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"swarm-manager/internal/records"
)

// recordSearchStore is a fake VectorStore whose Search returns canned hits.
// We can't reuse fakeVectorStore because its Search returns nil; a custom
// seeded type lets us drive Search/SearchRecords end-to-end without Qdrant.
type recordSearchStore struct {
	mu      sync.Mutex
	results []SearchResult
}

func (s *recordSearchStore) EnsureCollection(_ context.Context) error { return nil }
func (s *recordSearchStore) Upsert(_ context.Context, _ string, _ []float64, _ map[string]interface{}) error {
	return nil
}
func (s *recordSearchStore) Delete(_ context.Context, _ string) error        { return nil }
func (s *recordSearchStore) BatchDelete(_ context.Context, _ []string) error { return nil }
func (s *recordSearchStore) CountPoints(_ context.Context) (int, error)      { return len(s.results), nil }
func (s *recordSearchStore) ScrollIDs(_ context.Context) (map[string]ScrollItem, error) {
	return nil, nil
}
func (s *recordSearchStore) Available(_ context.Context) bool { return true }
func (s *recordSearchStore) Search(_ context.Context, _ []float64, limit int, _ float64) ([]SearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]SearchResult(nil), s.results...)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// inMemoryRecordStore satisfies records.Store for SearchRecords rehydrate tests.
type inMemoryRecordStore struct {
	items map[string]records.Record
}

func (s *inMemoryRecordStore) Create(r records.Record) error {
	if s.items == nil {
		s.items = make(map[string]records.Record)
	}
	s.items[r.ID] = r
	return nil
}

func (s *inMemoryRecordStore) Get(id string) (records.Record, error) {
	if r, ok := s.items[id]; ok {
		return r, nil
	}
	return records.Record{}, records.ErrNotFound
}

func (s *inMemoryRecordStore) List(_ records.ListFilter) ([]records.Record, error) {
	out := make([]records.Record, 0, len(s.items))
	for _, r := range s.items {
		out = append(out, r)
	}
	return out, nil
}

func (s *inMemoryRecordStore) UpdateNarrative(id string, _ records.Narrative, _ time.Time) (records.Record, error) {
	return records.Record{}, errors.New("not implemented")
}

func (s *inMemoryRecordStore) SetSupersededBy(id, _ string) (records.Record, error) {
	return records.Record{}, errors.New("not implemented")
}

func TestSearch_IncludesRecordsWhenEntityRecord(t *testing.T) {
	vs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Score: 0.9, Payload: map[string]interface{}{
			"entity_type": "record", "record_id": "rec-1", "kind": "fix", "scenario": "audio-tools",
		}},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	resp, err := svc.Search(context.Background(), AISearchRequest{
		Query:  "voice auto-stop",
		Entity: EntityRecord,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 result, got %d (results=%+v)", resp.Total, resp.Results)
	}
	if resp.Results[0].Entity != EntityRecord {
		t.Errorf("expected EntityRecord, got %s", resp.Results[0].Entity)
	}
}

func TestSearch_EntityBothIncludesRecords(t *testing.T) {
	recs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Score: 0.9, Payload: map[string]interface{}{
			"entity_type": "record", "kind": "fix", "scenario": "audio-tools",
		}},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(recs)

	resp, err := svc.Search(context.Background(), AISearchRequest{
		Query:  "anything",
		Entity: EntityBoth,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	// Backlog + initiative stores are nil, so EntityBoth should still pick up
	// the record store and return its results.
	if resp.Total != 1 || resp.Results[0].Entity != EntityRecord {
		t.Errorf("expected EntityBoth to surface records when only recordStore is wired; got %+v", resp)
	}
}

func TestApplyFilters_KindAppliesToRecords(t *testing.T) {
	in := []AISearchResult{
		{Entity: EntityRecord, ID: "r1", Payload: map[string]interface{}{"kind": "fix"}},
		{Entity: EntityRecord, ID: "r2", Payload: map[string]interface{}{"kind": "execute"}},
	}
	out := applyFilters(in, SearchFilters{Kind: []string{"fix"}, IncludeArchived: true})
	if len(out) != 1 || out[0].ID != "r1" {
		t.Errorf("expected kind=fix to keep r1 only, got %+v", out)
	}
}

func TestApplyFilters_TargetScenarioMatchesRecordScenario(t *testing.T) {
	in := []AISearchResult{
		{Entity: EntityRecord, ID: "r1", Payload: map[string]interface{}{"scenario": "audio-tools"}},
		{Entity: EntityRecord, ID: "r2", Payload: map[string]interface{}{"scenario": "swarm-manager"}},
	}
	out := applyFilters(in, SearchFilters{TargetScenario: "audio-tools", IncludeArchived: true})
	if len(out) != 1 || out[0].ID != "r1" {
		t.Errorf("expected target-scenario filter to narrow records, got %+v", out)
	}
}

func TestApplyFilters_StatusSkippedForRecords(t *testing.T) {
	in := []AISearchResult{
		{Entity: EntityRecord, ID: "r1", Payload: map[string]interface{}{"kind": "fix"}},
	}
	// Records have no "status" field; a status filter must not exclude them.
	out := applyFilters(in, SearchFilters{Status: []string{"open"}, IncludeArchived: true})
	if len(out) != 1 {
		t.Errorf("status filter must not apply to records (no status field), got %+v", out)
	}
}

func TestSearchRecords_RehydratesFromStore(t *testing.T) {
	vs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Score: 0.91, Payload: map[string]interface{}{
			"entity_type": "record", "record_id": "rec-A", "kind": "fix", "scenario": "audio-tools",
		}},
		{ID: "p2", Score: 0.80, Payload: map[string]interface{}{
			"entity_type": "record", "record_id": "rec-B", "kind": "execute", "scenario": "swarm-manager",
		}},
	}}
	store := &inMemoryRecordStore{items: map[string]records.Record{
		"rec-A": {ID: "rec-A", Kind: records.KindFix, Scenario: "audio-tools", Trigger: "t"},
		"rec-B": {ID: "rec-B", Kind: records.KindExecute, Scenario: "swarm-manager", Trigger: "t"},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	hits, err := svc.SearchRecords(context.Background(), "q", records.SearchFilter{Limit: 5}, store)
	if err != nil {
		t.Fatalf("SearchRecords: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 rehydrated hits, got %d", len(hits))
	}
	if hits[0].Record.ID != "rec-A" || hits[1].Record.ID != "rec-B" {
		t.Errorf("unexpected rehydrate order: %+v", hits)
	}
}

func TestSearchRecords_KindFilterAppliedBeforeRehydrate(t *testing.T) {
	vs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Score: 0.91, Payload: map[string]interface{}{
			"entity_type": "record", "record_id": "rec-A", "kind": "fix", "scenario": "s",
		}},
		{ID: "p2", Score: 0.80, Payload: map[string]interface{}{
			"entity_type": "record", "record_id": "rec-B", "kind": "execute", "scenario": "s",
		}},
	}}
	store := &inMemoryRecordStore{items: map[string]records.Record{
		"rec-A": {ID: "rec-A", Kind: records.KindFix, Scenario: "s"},
		"rec-B": {ID: "rec-B", Kind: records.KindExecute, Scenario: "s"},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	hits, err := svc.SearchRecords(context.Background(), "q", records.SearchFilter{Kind: records.KindFix, Limit: 5}, store)
	if err != nil {
		t.Fatalf("SearchRecords: %v", err)
	}
	if len(hits) != 1 || hits[0].Record.ID != "rec-A" {
		t.Errorf("expected kind filter to keep rec-A only, got %+v", hits)
	}
}

func TestSearchRecords_ScenarioFilterAppliedBeforeRehydrate(t *testing.T) {
	vs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Payload: map[string]interface{}{"record_id": "rec-A", "kind": "fix", "scenario": "audio-tools"}},
		{ID: "p2", Payload: map[string]interface{}{"record_id": "rec-B", "kind": "fix", "scenario": "swarm-manager"}},
	}}
	store := &inMemoryRecordStore{items: map[string]records.Record{
		"rec-A": {ID: "rec-A", Kind: records.KindFix, Scenario: "audio-tools"},
		"rec-B": {ID: "rec-B", Kind: records.KindFix, Scenario: "swarm-manager"},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	hits, err := svc.SearchRecords(context.Background(), "q", records.SearchFilter{Scenario: "audio-tools", Limit: 5}, store)
	if err != nil {
		t.Fatalf("SearchRecords: %v", err)
	}
	if len(hits) != 1 || hits[0].Record.ID != "rec-A" {
		t.Errorf("expected scenario filter to keep rec-A only, got %+v", hits)
	}
}

func TestSearchRecords_SkipsOrphansWithoutFailing(t *testing.T) {
	vs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Payload: map[string]interface{}{"record_id": "rec-orphan", "kind": "fix", "scenario": "s"}},
		{ID: "p2", Payload: map[string]interface{}{"record_id": "rec-A", "kind": "fix", "scenario": "s"}},
	}}
	store := &inMemoryRecordStore{items: map[string]records.Record{
		"rec-A": {ID: "rec-A", Kind: records.KindFix, Scenario: "s"},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	hits, err := svc.SearchRecords(context.Background(), "q", records.SearchFilter{Limit: 5}, store)
	if err != nil {
		t.Fatalf("SearchRecords: %v", err)
	}
	if len(hits) != 1 || hits[0].Record.ID != "rec-A" {
		t.Errorf("expected orphan hit to be silently dropped, got %+v", hits)
	}
}

func TestSearchRecords_ErrorsWhenUnconfigured(t *testing.T) {
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	// No SetRecordStore.
	store := &inMemoryRecordStore{}
	if _, err := svc.SearchRecords(context.Background(), "q", records.SearchFilter{}, store); err == nil {
		t.Error("expected error when record store unconfigured")
	}
}

func TestRecordSearcherAdapter_DelegatesAndConverts(t *testing.T) {
	vs := &recordSearchStore{results: []SearchResult{
		{ID: "p1", Score: 0.77, Payload: map[string]interface{}{"record_id": "rec-A", "kind": "fix", "scenario": "s"}},
	}}
	store := &inMemoryRecordStore{items: map[string]records.Record{
		"rec-A": {ID: "rec-A", Kind: records.KindFix, Scenario: "s"},
	}}
	svc := NewService(fakeEmbedderOK(), nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	adapter := NewRecordSearcherAdapter(svc, store)
	hits, err := adapter.SearchRecords("q", records.SearchFilter{Limit: 5})
	if err != nil {
		t.Fatalf("adapter SearchRecords: %v", err)
	}
	if len(hits) != 1 || hits[0].Record.ID != "rec-A" || hits[0].Score < 0.7 {
		t.Errorf("unexpected adapter result: %+v", hits)
	}
}

func TestRecordSearcherAdapter_NilSafe(t *testing.T) {
	var a *RecordSearcherAdapter
	if _, err := a.SearchRecords("q", records.SearchFilter{}); !errors.Is(err, records.ErrSearchUnavailable) {
		t.Errorf("nil adapter should return ErrSearchUnavailable, got %v", err)
	}
}
