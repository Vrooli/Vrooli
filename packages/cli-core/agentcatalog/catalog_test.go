package agentcatalog

import (
	"os"
	"path/filepath"
	"testing"
)

// A policy that declares billing sources and per-role multi-source model lists
// must still load: those fields are declarative evidence beside the resolved
// model, not a new resolution contract.
func TestLoadCodingRoleCatalogAcceptsMultiSourceDeclarations(t *testing.T) {
	policy := `{
  "schema_version": "v1",
  "runner": "opencode",
  "provenance": {"source": "test", "observed_at": "2026-09-11"},
  "billing": {
    "mode": "unknown",
    "source": "operator-not-declared",
    "sources": [
      {"id": "opencode-go", "label": "OpenCode Go", "billing_mode": "subscription", "credential_key": "OPENCODE_GO_KEY", "endpoint": "https://opencode.ai/zen/go/v1"},
      {"id": "ollama", "label": "Local Ollama", "billing_mode": "local", "credential_key": "", "endpoint": "http://localhost:11434"}
    ]
  },
  "roles": {
    "code.default": {
      "models": [
        {"model": "opencode-go/deepseek-v4.1-flash", "source": "opencode-go", "effort": null},
        {"model": "gemma4:12b", "source": "ollama", "effort": "high"}
      ],
      "model": "opencode-go/deepseek-v4.1-flash",
      "fallbacks": ["ollama/gemma4:12b"],
      "description": "balanced",
      "capabilities": ["code"]
    },
    "code.fast": {"model": "m", "description": "d", "capabilities": ["code"]},
    "code.smart": {"model": "m", "description": "d", "capabilities": ["code"]},
    "code.cheap": {"model": "m", "description": "d", "capabilities": ["code"]}
  }
}`
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, _, err := LoadCodingRoleCatalog("opencode", path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if catalog.Billing == nil || len(catalog.Billing.Sources) != 2 || catalog.Billing.Sources[0].ID != "opencode-go" {
		t.Fatalf("billing sources not decoded: %+v", catalog.Billing)
	}
	role := catalog.Roles["code.default"]
	if role.Model != "opencode-go/deepseek-v4.1-flash" {
		t.Fatalf("resolved model changed: %q", role.Model)
	}
	if len(role.Models) != 2 || role.Models[0].Source != "opencode-go" || role.Models[0].Effort != nil {
		t.Fatalf("role models not decoded: %+v", role.Models)
	}
	if role.Models[1].Effort == nil || *role.Models[1].Effort != "high" {
		t.Fatalf("role model effort not decoded: %+v", role.Models[1])
	}
}
