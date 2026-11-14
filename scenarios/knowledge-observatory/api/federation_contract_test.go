package main

import (
	"encoding/json"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

// TestDocSearchHitFederationContract freezes the JSON shape search-hub federates
// for the KO `doc` leaf. The shared engine rewrite (aisearch-go graduation) must
// not drift these keys: search-hub's providers.MapResults + the live provider
// descriptor map {id, relative_path, score, snippet, path}. If this test fails,
// the search-hub provider descriptor must change in lockstep — do not "fix" the
// test by loosening it.
func TestDocSearchHitFederationContract(t *testing.T) {
	hit := projectDocHits([]pkg.SearchResult{{
		ID:           "ko-docs:docs/a.md#0",
		RelativePath: "docs/a.md",
		Score:        0.87,
		Snippet:      "a snippet",
		Path:         "docs/a.md",
		Payload: map[string]any{
			"scenario":     "cli-health",
			"doc_type":     "reference",
			"title":        "A Doc",
			"heading_path": "A > B",
		},
	}})
	if len(hit) != 1 {
		t.Fatalf("expected 1 projected hit, got %d", len(hit))
	}

	raw, err := json.Marshal(hit[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// The five federation-contract keys MUST be present with these exact names.
	for _, key := range []string{"id", "relative_path", "score", "snippet", "path"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("federation contract broken: missing key %q in %s", key, raw)
		}
	}

	// Spot-check the values round-trip (projection wired the right fields).
	var decoded docSearchHit
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode docSearchHit: %v", err)
	}
	if decoded.ID != "ko-docs:docs/a.md#0" || decoded.RelativePath != "docs/a.md" ||
		decoded.Path != "docs/a.md" || decoded.Snippet != "a snippet" || decoded.Score != 0.87 {
		t.Fatalf("federation values wrong: %+v", decoded)
	}
	if decoded.Scenario != "cli-health" || decoded.DocType != "reference" ||
		decoded.Title != "A Doc" || decoded.HeadingPath != "A > B" {
		t.Fatalf("enrichment fields wrong: %+v", decoded)
	}
}
