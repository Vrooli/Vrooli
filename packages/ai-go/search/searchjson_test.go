package aisearch

import (
	"strings"
	"testing"
)

const sampleSearchJSON = `{
  "version": "1.0.0",
  "providers": [{
    "provider_id": "demo.commands",
    "provider_group": "demo",
    "bucket": "BUCKET_DO",
    "type": "command",
    "description": "demo",
    "scope": "SCOPE_PROJECT",
    "config_writable": false,
    "config_pinned_reason": "demo provider pins tuning for parser coverage",
    "endpoint": {"http_json": {"path": "/x"}},
    "result_mapping": {"id_field": "name"},
    "tuning": {
      "engine": "dense",
      "embed_task_prefix": true,
      "rerank_enabled": true,
      "rerank_blend": true,
      "rerank_shortlist": 50,
      "floor": {"max_gap": 0, "hard_floor": 0}
    },
    "scoring": {"recall_at": 5, "recall_target": 0.8, "mrr_at": 3, "deep_k": 50},
    "tests": {
      "name": "demo commands — primary",
      "description": "rank-centric demo corpus",
      "coverage": {
        "required_tags": ["strong"],
        "required_tag_groups": [
          {"id": "operator-workflows", "description": "Common operator command intents.", "tags": ["strong", "workflow"], "min_reviewed_positive": 1}
        ]
      },
      "cases": [
        {"id": "c1", "query": "restart a scenario", "scope": "scenario:cli-health", "expect_ids": ["restart"], "tags": ["strong"], "expect_within_top_k": 3},
        {"id": "n1", "query": "asdf qwer", "expect_no_strong_hit": true, "expect_max_score": 0.1, "tags": ["gibberish"]}
      ]
    }
  }]
}`

func TestParseSearchFile(t *testing.T) {
	f, err := ParseSearchFile([]byte(sampleSearchJSON))
	if err != nil {
		t.Fatalf("ParseSearchFile: %v", err)
	}
	if f.Version != "1.0.0" || len(f.Providers) != 1 {
		t.Fatalf("unexpected top-level: %+v", f)
	}
	p, ok := f.Provider("demo.commands")
	if !ok {
		t.Fatal("provider demo.commands not found")
	}
	if p.Tuning.Engine != EngineDense || !p.Tuning.EmbedTaskPrefix || !p.Tuning.RerankBlend {
		t.Errorf("tuning not parsed: %+v", p.Tuning)
	}
	if p.ConfigWritable == nil || *p.ConfigWritable || p.ConfigPinnedReason == "" {
		t.Errorf("config writability not parsed: writable=%v reason=%q", p.ConfigWritable, p.ConfigPinnedReason)
	}
	if scoring := p.ResolvedScoring(); scoring.GateK != 5 || scoring.RecallTarget != 0.8 || scoring.DeepK != 50 {
		t.Errorf("scoring not parsed: %+v", scoring)
	}
	// Descriptor sub-objects round-trip as raw JSON (not interpreted here).
	if !strings.Contains(string(p.Endpoint), "http_json") {
		t.Errorf("endpoint raw not preserved: %s", p.Endpoint)
	}
	// One unified case list: a positive and a negative live side by side.
	if len(p.Tests.Cases) != 2 {
		t.Fatalf("tests not parsed (want 2 cases incl. negative): %+v", p.Tests)
	}
	if p.Tests.ResolvedSuiteID("demo.commands") != "demo.commands.primary" {
		t.Errorf("default suite id not derived: %q", p.Tests.ResolvedSuiteID("demo.commands"))
	}
	if len(p.Tests.Coverage.RequiredTags) != 1 || p.Tests.Coverage.RequiredTags[0] != "strong" {
		t.Errorf("coverage required tags not parsed: %+v", p.Tests.Coverage)
	}
	if len(p.Tests.Coverage.RequiredTagGroups) != 1 || p.Tests.Coverage.RequiredTagGroups[0].ID != "operator-workflows" {
		t.Errorf("coverage groups not parsed: %+v", p.Tests.Coverage)
	}
	c := p.Tests.Cases[0]
	if len(c.ExpectIDs) != 1 || c.ExpectWithinTopK != 3 || c.ResolvedScope().Kind != ScopeScenario {
		t.Errorf("positive case not parsed rank-centrically: %+v", c)
	}
	neg := p.Tests.Cases[1]
	if !neg.ExpectNoStrongHit || neg.ExpectMaxScore != 0.1 {
		t.Errorf("negative case not parsed: %+v", neg)
	}
}

func TestParseSearchFileRejectsBad(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"no version", `{"providers":[{"provider_id":"a","tuning":{"engine":"dense"}}]}`},
		{"no providers", `{"version":"1.0.0","providers":[]}`},
		{"missing provider_id", `{"version":"1.0.0","providers":[{"tuning":{"engine":"dense"}}]}`},
		{"duplicate id", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"}},{"provider_id":"a","tuning":{"engine":"dense"}}]}`},
		{"bad tuning", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"nope"}}]}`},
		{"bad scoring", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"},"scoring":{"recall_target":2}}]}`},
		{"bad status", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"},"tests":{"cases":[{"id":"c","query":"q","status":"maybe"}]}}]}`},
		{"bad scope", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"},"tests":{"cases":[{"id":"c","query":"q","scope":"scenario:"}]}}]}`},
		{"positive without ids", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"},"tests":{"cases":[{"id":"c","query":"q","expect_within_top_k":5}]}}]}`},
		{"empty coverage tag", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"},"tests":{"coverage":{"required_tags":[""]},"cases":[{"id":"c","query":"q"}]}}]}`},
		{"coverage group no tags", `{"version":"1.0.0","providers":[{"provider_id":"a","tuning":{"engine":"dense"},"tests":{"coverage":{"required_tag_groups":[{"id":"empty"}]},"cases":[{"id":"c","query":"q"}]}}]}`},
		{"unknown field", `{"version":"1.0.0","wat":1,"providers":[{"provider_id":"a","tuning":{"engine":"dense"}}]}`},
		{"bad class", `{"version":"1.0.0","providers":[{"provider_id":"a","class":"turbo","tuning":{"engine":"dense"}}]}`},
		{"bad perf p95", `{"version":"1.0.0","providers":[{"provider_id":"a","performance":{"p95_ms":-1},"tuning":{"engine":"dense"}}]}`},
		{"bad perf degraded", `{"version":"1.0.0","providers":[{"provider_id":"a","performance":{"degraded_rate_max":2},"tuning":{"engine":"dense"}}]}`},
		{"missing pinned reason", `{"version":"1.0.0","providers":[{"provider_id":"a","config_writable":false,"tuning":{"engine":"dense"}}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseSearchFile([]byte(tc.json)); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestProviderClassTaxonomy(t *testing.T) {
	for _, class := range KnownProviderClasses {
		if !ValidProviderClass(class) {
			t.Errorf("known class %q reported invalid", class)
		}
	}
	if ValidProviderClass("") || ValidProviderClass("turbo") {
		t.Error("empty/unknown class must be invalid")
	}
	if !IndexedClass(ClassLocalIndex) || !IndexedClass(ClassAsync) {
		t.Error("local_index/async must be indexed classes")
	}
	if IndexedClass(ClassLocalLive) || IndexedClass(ClassExternal) {
		t.Error("local_live/external must not be indexed classes")
	}
	// A valid class parses.
	if _, err := ParseSearchFile([]byte(`{"version":"1.0.0","providers":[{"provider_id":"a","class":"external","tuning":{"engine":"dense"}}]}`)); err != nil {
		t.Errorf("valid class rejected: %v", err)
	}
}

func TestPerformancePolicyDefaults(t *testing.T) {
	if DefaultP95BudgetMs(ClassExternal) <= DefaultP95BudgetMs(ClassLocalIndex) {
		t.Error("external budget must be looser than local_index")
	}
	if DefaultP95BudgetMs(ClassAsync) <= DefaultP95BudgetMs(ClassExternal) {
		t.Error("async budget must be the loosest")
	}
	if err := (PerformanceConfig{P95Ms: 800, DegradedRateMax: 0.1, TelemetryRequired: true, MinimumSamples: 8}).Validate(); err != nil {
		t.Errorf("valid performance rejected: %v", err)
	}
	if err := (PerformanceConfig{MinimumSamples: -1}).Validate(); err == nil {
		t.Error("negative minimum samples unexpectedly accepted")
	}
	if got := DefaultMinimumSamples(ClassLocalIndex); got != 8 {
		t.Errorf("local index minimum samples = %d, want 8", got)
	}
	if got := DefaultMinimumSamples(ClassExternal); got != 1 {
		t.Errorf("external minimum samples = %d, want 1", got)
	}
	if _, err := ParseSearchFile([]byte(`{"version":"1.0.0","providers":[{"provider_id":"a","class":"external","performance":{"p95_ms":4000},"tuning":{"engine":"dense"}}]}`)); err != nil {
		t.Errorf("valid performance block rejected: %v", err)
	}
}

func TestResolvedTuning(t *testing.T) {
	p := ProviderConfig{Tuning: TuningConfig{Engine: EngineHybrid}}
	rt := p.ResolvedTuning()
	if rt.EmbedModel != DefaultEmbedModel || rt.RerankShortlist != DefaultRerankShortlist {
		t.Errorf("ResolvedTuning did not fill defaults: %+v", rt)
	}
}
