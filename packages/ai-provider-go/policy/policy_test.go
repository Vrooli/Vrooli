package policy

import (
	"strings"
	"testing"
)

const fixture = `{
  "schema_version": "test-1",
  "provider": {"id": "example", "family": "openai-compatible", "capabilities": ["chat", "tools"]},
  "roles": {
    "chat.default": {
      "model": "vendor/chat-a",
      "fallbacks": ["vendor/chat-b"],
      "endpoint": "chat",
      "required_capabilities": ["chat"],
      "description": "default chat",
      "preference": 10,
      "request_defaults": {"temperature": 0.7, "max_tokens": 1024},
      "sampling_support": {"temperature": "honored"},
      "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-10-30", "sample_count": 0}
    },
    "image.generate.logo": {
      "model": "vendor/img-vec",
      "fallbacks": [],
      "endpoint": "images",
      "required_capabilities": ["image_output", "svg_output"],
      "description": "logo",
      "preference": 20,
      "request_defaults": {"output_format": "png", "background": "transparent"},
      "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-10-30", "sample_count": 0}
    }
  },
  "models": {
    "vendor/chat-a": {
      "provider": "vendor", "family": "chat",
      "capabilities": ["chat", "text_input", "text_output"],
      "modalities": {"input": ["text"], "output": ["text"]},
      "endpoints": ["chat"], "default_eligible": true,
      "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-10-30", "sample_count": 0}}
    },
    "vendor/chat-b": {
      "provider": "vendor", "family": "chat",
      "capabilities": ["chat", "text_input", "text_output"],
      "modalities": {"input": ["text"], "output": ["text"]},
      "endpoints": ["chat"], "default_eligible": true,
      "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-10-30", "sample_count": 0}}
    },
    "vendor/img-vec": {
      "provider": "vendor", "family": "img",
      "capabilities": ["text_input", "image_output", "svg_output", "transparent_background"],
      "modalities": {"input": ["text"], "output": ["image", "vector"]},
      "endpoints": ["images"], "default_eligible": true,
      "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-10-30", "sample_count": 0}}
    }
  },
  "constraints": {
    "endpoints": ["chat", "images"],
    "capability_vocabulary": ["chat", "text_input", "text_output", "image_output", "svg_output", "transparent_background"],
    "role_preference_order": ["chat.default", "image.generate.logo"],
    "direct_model_exception_required_fields": ["reason", "owner", "review_after"],
    "provenance_required": true,
    "provenance_source_kinds": ["manual_policy", "provider_docs"],
    "modality_vocabulary": ["text", "image", "vector", "video", "audio"]
  },
  "provenance": {
    "policy": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-10-30", "sample_count": 0}
  }
}`

func TestValidateAndResolve(t *testing.T) {
	catalog, err := Parse([]byte(fixture), CloudOptions())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	resolved, err := ResolveRole(catalog, "chat.default")
	if err != nil {
		t.Fatalf("ResolveRole: %v", err)
	}
	if resolved.Model != "vendor/chat-a" || resolved.Endpoint != "chat" {
		t.Fatalf("resolved = %+v", resolved)
	}
	if got := resolved.SamplingSupport["temperature"]; got != SamplingHonored {
		t.Fatalf("sampling support = %q", got)
	}
	if resolved.RequestDefaults == nil || resolved.RequestDefaults.MaxTokens == nil || *resolved.RequestDefaults.MaxTokens != 1024 {
		t.Fatalf("request defaults not propagated: %+v", resolved.RequestDefaults)
	}
	resolution, err := Resolve(catalog, ResolveRequest{ModelRoles: []RoleRequest{{Role: "chat.default"}, {Role: "image.generate.logo"}}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolution.Roles) != 2 {
		t.Fatalf("roles = %d", len(resolution.Roles))
	}
}

func TestExcludedModelFallsBack(t *testing.T) {
	catalog, err := Parse([]byte(fixture), CloudOptions())
	if err != nil {
		t.Fatal(err)
	}
	catalog.ExcludedModels = []string{"vendor/chat-a"}
	resolved, err := ResolveRole(catalog, "chat.default")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Model != "vendor/chat-b" {
		t.Fatalf("model = %q, want fallback", resolved.Model)
	}
	if _, err := ResolveModel(catalog, "vendor/chat-a"); err == nil || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("excluded model error = %v", err)
	}
}

func TestValidationRejectsBrokenCatalog(t *testing.T) {
	cases := map[string]func(string) string{
		"endpoint mismatch": func(body string) string {
			return strings.Replace(body, `"endpoint": "chat",`, `"endpoint": "images",`, 1)
		},
		"missing capability": func(body string) string {
			return strings.Replace(body, `"required_capabilities": ["chat"],`, `"required_capabilities": ["telepathy"],`, 1)
		},
		"bad provenance": func(body string) string {
			return strings.Replace(body, `"source_kind": "manual_policy", "confidence": "high", "source": "fixture"`, `"source_kind": "made_up", "confidence": "high", "source": "fixture"`, 1)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(mutate(fixture)), CloudOptions()); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func TestLocalValidation(t *testing.T) {
	body := `{
      "schema_version": "local-1",
      "roles": {"chat.default": {"model": "local:latest", "required_capabilities": ["chat"], "preference": 1, "provenance": {"source_kind": "manual_policy", "confidence": "medium", "source": "fixture", "observed_at": "2026-10-30"}}},
      "models": {"local:latest": {"family": "local", "capabilities": ["chat"], "modalities": {"input": ["text"], "output": ["text"]}, "disk_size_gb_estimate": 1, "ram_gb_estimate": 1, "vram_gb_estimate": 0.1, "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "medium", "source": "fixture", "observed_at": "2026-10-30"}}}},
      "constraints": {"role_preference_order": ["chat.default"], "provenance_source_kinds": ["manual_policy"], "modality_vocabulary": ["text"]}
    }`
	catalog, err := Parse([]byte(body), LocalOptions())
	if err != nil {
		t.Fatalf("local parse: %v", err)
	}
	if _, err := ResolveRole(catalog, "chat.default"); err != nil {
		t.Fatalf("local resolve: %v", err)
	}
}

func TestResolveGeneratePrecedence(t *testing.T) {
	temp := 0.0
	max := 512
	defaults := &RequestDefaults{Temperature: &temp, MaxTokens: &max}
	gotTemp, gotMax := defaults.ResolveGenerate(-1, 0)
	if gotTemp == nil || *gotTemp != 0 || gotMax != 512 {
		t.Fatalf("precedence = %v/%d", gotTemp, gotMax)
	}
	flag := 1.2
	gotTemp, gotMax = defaults.ResolveGenerate(flag, 4096)
	if gotTemp == nil || *gotTemp != 1.2 || gotMax != 4096 {
		t.Fatalf("flag precedence = %v/%d", gotTemp, gotMax)
	}
	gotTemp, gotMax = (*RequestDefaults)(nil).ResolveGenerate(-1, 0)
	if gotTemp != nil || gotMax != 0 {
		t.Fatalf("absent = %v/%d", gotTemp, gotMax)
	}
}
