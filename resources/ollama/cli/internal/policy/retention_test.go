package policy

import "testing"

func TestRequiredModelsKeepsPrimariesAndPinsButNotFallbacks(t *testing.T) {
	p := Policy{
		Roles: map[string]Role{
			"chat.small":   {Model: "qwen3.5:4b", Fallbacks: []string{"llama3.2:3b", "llama3.2:1b"}},
			"classify":     {Model: "qwen3.5:4b", Fallbacks: []string{"qwen3:4b"}},
			"vision":       {Model: "qwen3-vl:4b", Fallbacks: []string{"llava:7b"}},
			"embedding":    {Model: "nomic-embed-text:latest"},
			"code.local":   {Model: "gemma4:12b", Fallbacks: []string{"llama3.1:8b"}},
			"write":        {Model: "gemma4:12b"},
			"summarize":    {Model: "qwen3:1.7b"},
			"uncensored":   {Model: "huihui_ai/granite4.1-abliterated:8b"},
			"vision.judge": {Model: "qwen3-vl:4b"},
		},
		Models:    map[string]Model{"llama3.2:1b": {}},
		Retention: Retention{Pinned: []string{"llama3.2:1b"}},
	}
	required := p.RequiredModels()
	for _, model := range []string{"qwen3.5:4b", "qwen3-vl:4b", "nomic-embed-text:latest", "gemma4:12b", "qwen3:1.7b", "huihui_ai/granite4.1-abliterated:8b", "llama3.2:1b"} {
		if required[model] == "" {
			t.Errorf("%s must be required", model)
		}
	}
	for _, model := range []string{"llama3.2:3b", "qwen3:4b", "llava:7b", "llama3.1:8b"} {
		if reason, ok := required[model]; ok {
			t.Errorf("fallback %s must be eviction-eligible, got required (%s)", model, reason)
		}
	}
	if got := required["qwen3.5:4b"]; got != "primary model of role chat.small, classify" {
		t.Errorf("shared primary reason = %q", got)
	}
	if got := required["llama3.2:1b"]; got != "pinned by retention.pinned" {
		t.Errorf("pin reason = %q", got)
	}
}

func TestRetentionPinsMustNameCatalogModels(t *testing.T) {
	p := Policy{Models: map[string]Model{"a:1b": {}}, Retention: Retention{Pinned: []string{"a:1b", "missing:2b"}}}
	errs := p.Retention.validate(p)
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want one for the uncataloged pin", errs)
	}
}
