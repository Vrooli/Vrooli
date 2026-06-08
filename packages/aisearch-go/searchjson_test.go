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
    "tests": {
      "recall_at": 5,
      "recall_target": 0.8,
      "cases": [
        {"id": "c1", "query": "restart a scenario", "expected_paths": ["vrooli scenario restart"], "expect_ids": ["restart"], "tags": ["strong"]}
      ],
      "negatives": [
        {"id": "n1", "query": "asdf qwer", "expect_no_strong_hit": true, "expect_max_score": 0.1}
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
	// Descriptor sub-objects round-trip as raw JSON (not interpreted here).
	if !strings.Contains(string(p.Endpoint), "http_json") {
		t.Errorf("endpoint raw not preserved: %s", p.Endpoint)
	}
	if len(p.Tests.Cases) != 1 || len(p.Tests.Negatives) != 1 {
		t.Fatalf("tests not parsed: %+v", p.Tests)
	}
	c := p.Tests.Cases[0]
	if len(c.ExpectedPaths) != 1 || len(c.ExpectIDs) != 1 {
		t.Errorf("both label shapes must survive: %+v", c)
	}
	if p.Tests.Negatives[0].ExpectMaxScore != 0.1 {
		t.Errorf("negative expect_max_score not parsed: %+v", p.Tests.Negatives[0])
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
		{"unknown field", `{"version":"1.0.0","wat":1,"providers":[{"provider_id":"a","tuning":{"engine":"dense"}}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseSearchFile([]byte(tc.json)); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestResolvedTuning(t *testing.T) {
	p := ProviderConfig{Tuning: TuningConfig{Engine: EngineHybrid}}
	rt := p.ResolvedTuning()
	if rt.EmbedModel != DefaultEmbedModel || rt.RerankShortlist != DefaultRerankShortlist {
		t.Errorf("ResolvedTuning did not fill defaults: %+v", rt)
	}
}
