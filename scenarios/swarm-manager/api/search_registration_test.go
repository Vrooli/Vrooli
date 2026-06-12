package main

import (
	"path/filepath"
	"runtime"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
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
	file, err := aisearch.LoadSearchFile(searchJSONForTest(t))
	if err != nil {
		t.Fatalf("load .vrooli/search.json: %v", err)
	}
	if len(file.Providers) != 1 {
		t.Fatalf("want exactly 1 provider, got %d", len(file.Providers))
	}

	descriptors, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("map search.json to registry descriptors: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("want 1 descriptor, got %d", len(descriptors))
	}

	d := descriptors[0]
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
