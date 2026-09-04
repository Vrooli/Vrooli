package conversationsearch_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureCorpus struct {
	ContentPolicy struct {
		SyntheticOnly             bool `json:"synthetic_only"`
		ContainsCredentials       bool `json:"contains_credentials"`
		ContainsOperatorHomePaths bool `json:"contains_operator_home_paths"`
	} `json:"content_policy"`
	Runs []struct {
		ID           string `json:"id"`
		Harness      string `json:"harness"`
		ProjectScope string `json:"project_scope"`
		Purged       bool   `json:"purged"`
		Events       []struct {
			ID           string `json:"id"`
			ContentClass string `json:"content_class"`
			Deleted      bool   `json:"deleted"`
			DuplicateOf  string `json:"duplicate_of"`
			Recipe       *struct {
				Repeat int `json:"repeat"`
			} `json:"content_recipe"`
		} `json:"events"`
	} `json:"runs"`
	Queries []struct {
		CoverageGroup       string   `json:"coverage_group"`
		ExpectedOrder       []string `json:"expected_order"`
		ExpectedDegradation string   `json:"expected_degradation"`
	} `json:"queries"`
}

func TestGoldenCorpusCoversRequiredAmbiguities(t *testing.T) {
	path := filepath.Join("testdata", "golden_corpus.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if strings.Contains(string(raw), "/home/") || strings.Contains(string(raw), "01b005dc") || strings.Contains(string(raw), "66df0b78") {
		t.Fatal("synthetic fixture must not contain operator-local paths or live identifiers")
	}

	var corpus fixtureCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if !corpus.ContentPolicy.SyntheticOnly || corpus.ContentPolicy.ContainsCredentials || corpus.ContentPolicy.ContainsOperatorHomePaths {
		t.Fatalf("unsafe fixture policy: %+v", corpus.ContentPolicy)
	}

	harnesses, projects, groups := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var deleted, purged, toolNoise, oversized, duplicate bool
	for _, run := range corpus.Runs {
		harnesses[run.Harness] = true
		projects[run.ProjectScope] = true
		purged = purged || run.Purged
		for _, event := range run.Events {
			deleted = deleted || event.Deleted
			toolNoise = toolNoise || event.ContentClass == "tool_output"
			duplicate = duplicate || event.DuplicateOf != ""
			oversized = oversized || (event.Recipe != nil && event.Recipe.Repeat >= 8192)
		}
	}
	for _, query := range corpus.Queries {
		groups[query.CoverageGroup] = true
		if query.CoverageGroup == "pagination" && len(query.ExpectedOrder) != 2 {
			t.Fatal("pagination case must fix the equal-score order")
		}
		if query.CoverageGroup == "degradation" && query.ExpectedDegradation == "" {
			t.Fatal("degradation case must declare its typed outcome")
		}
	}

	for _, group := range []string{"golden-positive", "exact-positive", "negative", "privacy", "filters", "pagination", "degradation"} {
		if !groups[group] {
			t.Errorf("missing query coverage group %q", group)
		}
	}
	if len(harnesses) < 2 || len(projects) < 2 || !deleted || !purged || !toolNoise || !oversized || !duplicate {
		t.Fatalf("fixture ambiguity coverage incomplete: harnesses=%d projects=%d deleted=%v purged=%v tool=%v oversized=%v duplicate=%v", len(harnesses), len(projects), deleted, purged, toolNoise, oversized, duplicate)
	}
}
