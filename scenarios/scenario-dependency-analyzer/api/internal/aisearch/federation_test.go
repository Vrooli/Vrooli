package aisearch

import (
	"path/filepath"
	"strings"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
)

// TestSearchJSONFederationDescriptors locks the federation contract: the shipped
// .vrooli/search.json parses, validates, and produces exactly the three SDA
// provider descriptors — dependencies, scenarios, resources — each pointing at
// its own Search RPC with a camelCase result_mapping. This is the boundary
// search-hub routes on, so a drift here would silently break discovery.
func TestSearchJSONFederationDescriptors(t *testing.T) {
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
	if len(descs) != 3 {
		t.Fatalf("descriptors = %d, want exactly three SDA providers", len(descs))
	}

	byID := map[string]int{}
	for i, d := range descs {
		byID[d.GetProviderId()] = i
	}
	for _, want := range []string{
		"scenario-dependency-analyzer.dependencies",
		"scenario-dependency-analyzer.scenarios",
		"scenario-dependency-analyzer.resources",
	} {
		if _, ok := byID[want]; !ok {
			t.Fatalf("missing provider %q", want)
		}
	}

	type want struct {
		typ        string
		pathSuffix string
		idField    string
		scoreField string
		descWords  []string
	}
	cases := map[string]want{
		"scenario-dependency-analyzer.dependencies": {
			typ: "dependency", pathSuffix: "/SearchApprovedDependencies",
			idField: "packageName", scoreField: "relevanceScore",
			descWords: []string{"dependency", "package"},
		},
		"scenario-dependency-analyzer.scenarios": {
			typ: "scenarios", pathSuffix: "/SearchInterfaceGraph",
			idField: "scenario", scoreField: "relevanceScore",
			descWords: []string{"depend", "connection"},
		},
		"scenario-dependency-analyzer.resources": {
			typ: "resources", pathSuffix: "/SearchResourceUsage",
			idField: "resource", scoreField: "relevanceScore",
			descWords: []string{"resource", "use"},
		},
	}

	for id, w := range cases {
		d := descs[byID[id]]
		if d.GetType() != w.typ {
			t.Errorf("%s: type = %q, want %q", id, d.GetType(), w.typ)
		}
		if got := d.GetBucket().String(); got != "BUCKET_REUSE" {
			t.Errorf("%s: bucket = %q, want BUCKET_REUSE", id, got)
		}
		hj := d.GetEndpoint().GetHttpJson()
		if hj == nil || !strings.HasSuffix(hj.GetPath(), w.pathSuffix) {
			t.Errorf("%s: endpoint path = %q, want suffix %q", id, hj.GetPath(), w.pathSuffix)
		}
		// protojson responses are camelCase — a snake_case mapping silently yields
		// empty titles + zero scores in the federated hit.
		rm := d.GetResultMapping()
		if rm.GetScoreField() != w.scoreField {
			t.Errorf("%s: score_field = %q, want %q", id, rm.GetScoreField(), w.scoreField)
		}
		if rm.GetIdField() != w.idField || rm.GetTitleField() != w.idField {
			t.Errorf("%s: id/title field must be camelCase %q, got id=%q title=%q", id, w.idField, rm.GetIdField(), rm.GetTitleField())
		}
		desc := strings.ToLower(d.GetDescription())
		for _, word := range w.descWords {
			if !strings.Contains(desc, word) {
				t.Errorf("%s: description should mention %q (classifier-sharp): %q", id, word, d.GetDescription())
			}
		}
	}
}
