package search_test

import (
	"path/filepath"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	"measures-health/internal/validation"
)

// TestSearchJSON_MapsToValidDescriptor loads the scenario-owned
// .vrooli/search.json (the registration SSOT) and runs it through the exact
// searchregister mapping the boot path uses, asserting it produces one valid
// measures provider descriptor with the carrier contract search-hub needs
// (type "measure", bucket STATE, result_mapping.measure_field "measure"). A
// malformed search.json fails here rather than degrading silently at boot.
func TestSearchJSON_MapsToValidDescriptor(t *testing.T) {
	repoRoot := validation.ResolveRepoRoot()
	path := filepath.Join(repoRoot, "scenarios", "measures-health", ".vrooli", "search.json")

	file, err := aisearch.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json: %v", err)
	}
	descriptors, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("map descriptors: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("expected exactly one provider, got %d", len(descriptors))
	}
	d := descriptors[0]
	if d.GetProviderId() != "measures-health.measures" {
		t.Fatalf("provider_id = %q", d.GetProviderId())
	}
	if d.GetType() != "measure" {
		t.Fatalf("type = %q, want measure", d.GetType())
	}
	if d.GetBucket() != registryv1.Bucket_BUCKET_STATE {
		t.Fatalf("bucket = %v, want BUCKET_STATE", d.GetBucket())
	}
	rm := d.GetResultMapping()
	if rm == nil {
		t.Fatal("result_mapping is nil")
	}
	if rm.GetMeasureField() != "measure" {
		t.Fatalf("measure_field = %q, want measure", rm.GetMeasureField())
	}
	if rm.GetResultsPath() != "results" || rm.GetScoreField() != "score" {
		t.Fatalf("results_path/score_field = %q/%q", rm.GetResultsPath(), rm.GetScoreField())
	}
	if d.GetEndpoint() == nil || d.GetStatusEndpoint() == nil {
		t.Fatal("endpoint and status_endpoint must both be present")
	}
}
