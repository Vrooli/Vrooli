package aisearch

import (
	"path/filepath"
	"strings"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
)

// TestSearchJSONFederationDescriptor locks the federation contract: the shipped
// .vrooli/search.json parses, validates, and produces a single dependency-typed
// provider descriptor pointing at the SearchApprovedDependencies RPC. This is the
// boundary search-hub routes on, so a drift here would silently break discovery.
func TestSearchJSONFederationDescriptor(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".vrooli", "search.json")

	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json: %v", err)
	}
	if err := file.Validate(); err != nil {
		t.Fatalf("search.json invalid: %v", err)
	}

	descs, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("build descriptors: %v", err)
	}
	if len(descs) != 1 {
		t.Fatalf("descriptors = %d, want exactly one dependency provider", len(descs))
	}
	d := descs[0]
	if d.GetProviderId() != "scenario-dependency-analyzer.dependencies" {
		t.Fatalf("provider_id = %q", d.GetProviderId())
	}
	if d.GetType() != "dependency" {
		t.Fatalf("type = %q, want dependency (kept out of the record,skill,doc recall preset)", d.GetType())
	}
	if got := d.GetBucket().String(); got != "BUCKET_REUSE" {
		t.Fatalf("bucket = %q, want BUCKET_REUSE (dependency packages are build-with-it)", got)
	}
	hj := d.GetEndpoint().GetHttpJson()
	if hj == nil || !strings.HasSuffix(hj.GetPath(), "/SearchApprovedDependencies") {
		t.Fatalf("endpoint path = %q, want the SearchApprovedDependencies RPC", hj.GetPath())
	}
	if desc := d.GetDescription(); !strings.Contains(strings.ToLower(desc), "dependency") || !strings.Contains(strings.ToLower(desc), "package") {
		t.Fatalf("description should be classifier-sharp about dependencies/packages: %q", desc)
	}
	// The Connect RPC response is camelCase (protojson default), so the result
	// mapping MUST use camelCase field names — snake_case silently yields empty
	// titles + zero scores in the federated hit.
	rm := d.GetResultMapping()
	if rm.GetScoreField() != "relevanceScore" {
		t.Fatalf("score_field = %q, want camelCase relevanceScore", rm.GetScoreField())
	}
	if rm.GetIdField() != "packageName" || rm.GetTitleField() != "packageName" {
		t.Fatalf("id/title field must be camelCase packageName, got id=%q title=%q", rm.GetIdField(), rm.GetTitleField())
	}
}
