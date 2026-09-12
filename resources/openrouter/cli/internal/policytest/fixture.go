package policytest

// FixturePolicyJSON is a minimal valid OpenRouter policy used across package
// tests (policy, policycmd, ensure). It exercises both chat and images endpoint
// families, a fallback, and request_defaults.
const FixturePolicyJSON = `{
  "schema_version": "test-1",
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
      "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-06-30", "sample_count": 0}
    },
    "image.generate.logo": {
      "model": "vendor/img-vec",
      "fallbacks": [],
      "endpoint": "images",
      "required_capabilities": ["image_output", "svg_output"],
      "description": "logo",
      "preference": 20,
      "request_defaults": {"output_format": "png", "background": "transparent"},
      "provenance": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-06-30", "sample_count": 0}
    }
  },
  "models": {
    "vendor/chat-a": {
      "provider": "vendor", "family": "chat",
      "capabilities": ["chat", "text_input", "text_output"],
      "modalities": {"input": ["text"], "output": ["text"]},
      "endpoints": ["chat"], "default_eligible": true,
      "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-06-30", "sample_count": 0}}
    },
    "vendor/chat-b": {
      "provider": "vendor", "family": "chat",
      "capabilities": ["chat", "text_input", "text_output"],
      "modalities": {"input": ["text"], "output": ["text"]},
      "endpoints": ["chat"], "default_eligible": true,
      "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-06-30", "sample_count": 0}}
    },
    "vendor/img-vec": {
      "provider": "vendor", "family": "img",
      "capabilities": ["text_input", "image_output", "svg_output", "transparent_background"],
      "modalities": {"input": ["text"], "output": ["image", "vector"]},
      "endpoints": ["images"], "default_eligible": true,
      "provenance": {"catalog": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-06-30", "sample_count": 0}}
    }
  },
  "constraints": {
    "endpoints": ["chat", "images"],
    "capability_vocabulary": ["chat", "text_input", "text_output", "image_output", "svg_output", "transparent_background"],
    "role_preference_order": ["chat.default", "image.generate.logo"],
    "direct_model_exception_required_fields": ["reason", "owner", "review_after"],
    "provenance_required": true,
    "provenance_source_kinds": ["manual_policy", "chatgpt_survey"],
    "modality_vocabulary": ["text", "image", "vector", "video", "audio"]
  },
  "provenance": {
    "policy": {"source_kind": "manual_policy", "confidence": "high", "source": "fixture", "observed_at": "2026-06-30", "sample_count": 0}
  }
}`
