package main

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"swarm-manager/internal/aisearch"

	aisearchfile "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
)

// searchJSONForTest resolves the committed .vrooli/search.json relative to this
// test file, independent of the process working directory.
func searchJSONForTest(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(here), "..", ".vrooli", "search.json")
}

// TestSearchJSONMapsToValidDescriptor is the Phase 2 guard that swarm-manager's
// `.vrooli/search.json` SSOT is well-formed and maps cleanly to the registry
// descriptor self-registration pushes to search-hub. It exercises the same
// load+map path startSearchRegistration drives at boot (aisearch.LoadSearchFile
// → searchregister.Descriptors), so a typo in the file's protojson sub-objects
// (endpoint / result_mapping) fails here rather than silently degrading at boot.
//
// It also pins the records leaf's rich mapping: the federated hit must carry the
// lesson (trigger→title, approach→snippet), not just an id — the whole point of
// federating against /records/search instead of the thin /search/ai endpoint.
func TestSearchJSONMapsToValidDescriptor(t *testing.T) {
	file, err := aisearchfile.LoadSearchFile(searchJSONForTest(t))
	if err != nil {
		t.Fatalf("load .vrooli/search.json: %v", err)
	}
	if len(file.Providers) != 2 {
		t.Fatalf("want exactly 2 providers, got %d", len(file.Providers))
	}

	descriptors, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("map search.json to registry descriptors: %v", err)
	}
	if len(descriptors) != 2 {
		t.Fatalf("want 2 descriptors, got %d", len(descriptors))
	}

	d := descriptors[0]
	for _, candidate := range descriptors {
		if candidate.GetProviderId() == "swarm-manager.records" {
			d = candidate
			break
		}
	}
	if got := d.GetProviderId(); got != "swarm-manager.records" {
		t.Errorf("provider_id = %q, want swarm-manager.records", got)
	}
	if got := d.GetType(); got != "record" {
		t.Errorf("type = %q, want record", got)
	}
	if got := d.GetBucket().String(); got != "BUCKET_KNOW" {
		t.Errorf("bucket = %q, want BUCKET_KNOW", got)
	}
	if d.GetEndpoint() == nil {
		t.Error("endpoint must be present for a live provider")
	}

	m := d.GetResultMapping()
	if m == nil {
		t.Fatal("result_mapping must be present")
	}
	if got := m.GetResultsPath(); got != "hits" {
		t.Errorf("results_path = %q, want hits", got)
	}
	if got := m.GetIdField(); got != "record.id" {
		t.Errorf("id_field = %q, want record.id", got)
	}
	if got := m.GetTitleField(); got != "record.trigger" {
		t.Errorf("title_field = %q, want record.trigger (the lesson's trigger)", got)
	}
	if got := m.GetSnippetField(); got != "record.approach" {
		t.Errorf("snippet_field = %q, want record.approach (the lesson's approach)", got)
	}
	if got := m.GetScoreScale().String(); got != "SCORE_SCALE_COSINE_0_1" {
		t.Errorf("score_scale = %q, want SCORE_SCALE_COSINE_0_1", got)
	}
}

type fakeAgentSessionAISearch struct {
	available bool
	responses map[aisearch.EntityType]*aisearch.AISearchResponse
	err       error
	requests  []aisearch.AISearchRequest
}

func (f *fakeAgentSessionAISearch) Available(context.Context) bool { return f.available }

func (f *fakeAgentSessionAISearch) Search(_ context.Context, req aisearch.AISearchRequest) (*aisearch.AISearchResponse, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	if response := f.responses[req.Entity]; response != nil {
		return response, nil
	}
	return &aisearch.AISearchResponse{Results: []aisearch.AISearchResult{}, Entity: req.Entity}, nil
}

func TestAgentSessionRelatedWorkSearcherMergesRanksAndDeduplicates(t *testing.T) {
	fake := &fakeAgentSessionAISearch{available: true, responses: map[aisearch.EntityType]*aisearch.AISearchResponse{
		aisearch.EntityBoth: {
			Fallback: aisearch.FallbackNone,
			Results: []aisearch.AISearchResult{
				{Entity: aisearch.EntityGoal, ID: "goal-a", Score: 0.72, Payload: map[string]interface{}{"name": "goal-a", "title": "Goal A", "status": "active", "priority": float64(4)}},
				{Entity: aisearch.EntityRecord, ID: "point-1", Score: 0.81, Payload: map[string]interface{}{"record_id": "rec-1", "title": "Prior fix", "kind": "fix", "scenario": "audio-tools"}},
			},
		},
		aisearch.EntityRecord: {
			Fallback: aisearch.FallbackNone,
			Results: []aisearch.AISearchResult{
				{Entity: aisearch.EntityRecord, ID: "point-1", Score: 0.91, Payload: map[string]interface{}{"record_id": "rec-1", "title": "Prior fix", "kind": "fix", "scenario": "audio-tools"}},
			},
		},
	}}

	entries, err := (agentSessionRelatedWorkSearcher{search: fake}).SearchRelatedWork(context.Background(), "session recall", 8)
	if err != nil {
		t.Fatalf("SearchRelatedWork() error = %v", err)
	}
	if len(fake.requests) != 2 || fake.requests[0].Entity != aisearch.EntityBoth || fake.requests[1].Entity != aisearch.EntityRecord {
		t.Fatalf("search requests = %+v, want both then record", fake.requests)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %+v, want two deduplicated entries", entries)
	}
	if entries[0].Ref != "record:rec-1" || entries[0].Score != 0.91 {
		t.Fatalf("top entry = %+v, want highest-scoring record duplicate", entries[0])
	}
	if entries[1].Ref != "goal:goal-a" || entries[1].Summary != "goal · active · priority 4" {
		t.Fatalf("goal entry = %+v", entries[1])
	}
}

func TestAgentSessionRelatedWorkSearcherReportsUnavailable(t *testing.T) {
	for _, fake := range []*fakeAgentSessionAISearch{
		{available: false},
		{available: true, err: errors.New("search failed")},
		{available: true, responses: map[aisearch.EntityType]*aisearch.AISearchResponse{
			aisearch.EntityBoth: {Fallback: aisearch.FallbackUnavailable},
		}},
	} {
		if _, err := (agentSessionRelatedWorkSearcher{search: fake}).SearchRelatedWork(context.Background(), "session recall", 8); err == nil {
			t.Fatal("SearchRelatedWork() error = nil, want unavailable error")
		}
	}
}
