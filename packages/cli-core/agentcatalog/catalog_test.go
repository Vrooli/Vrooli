package agentcatalog

import (
	"os"
	"path/filepath"
	"strings"
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

func TestModelRoleRestrictionsRequireKnownDistinctRoles(t *testing.T) {
	for _, tc := range []struct {
		name       string
		restricted map[string][]string
		valid      bool
	}{
		{"explicit supervisor", map[string][]string{"premium": {"judgment.supervision"}}, true},
		{"empty model", map[string][]string{"": {"judgment.supervision"}}, false},
		{"untrimmed model", map[string][]string{" premium": {"judgment.supervision"}}, false},
		{"duplicate spelling", map[string][]string{"premium": {"judgment.supervision"}, "PREMIUM": {"judgment.supervision"}}, false},
		{"no role", map[string][]string{"premium": {}}, false},
		{"unknown role", map[string][]string{"premium": {"judgment.typo"}}, false},
		{"duplicate role", map[string][]string{"premium": {"judgment.supervision", "judgment.supervision"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := CodingRoleCatalog{
				SchemaVersion: CodingRolePolicySchemaVersion, Runner: "test",
				Provenance: CatalogProvenance{Source: "fixture", ObservedAt: "2026-09-26"},
				Roles:      make(map[string]CodingRole), RestrictedModels: tc.restricted,
			}
			for _, role := range []string{"code.default", "code.fast", "code.smart", "code.cheap", "judgment.supervision"} {
				catalog.Roles[role] = CodingRole{Model: "routine", Description: "fixture", Capabilities: []string{"code"}}
			}
			if err := validateCodingRoleCatalog(catalog, "test"); (err == nil) != tc.valid {
				t.Fatalf("valid=%t, validation error=%v", tc.valid, err)
			}
		})
	}
}

// The interactive default is not a role: a restricted model may be the default,
// an excluded or unknown one may not.
func TestInteractiveDefaultMustBeKnownAndNotExcluded(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model string
		valid bool
	}{
		{"restricted model is allowed", "premium", true},
		{"routine model", "routine", true},
		{"excluded model", "banned", false},
		{"excluded model, other case", "Banned", false},
		{"unknown model", "missing", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := CodingRoleCatalog{
				SchemaVersion: CodingRolePolicySchemaVersion, Runner: "test",
				Provenance:         CatalogProvenance{Source: "fixture", ObservedAt: "2026-09-26"},
				ExcludedModels:     []string{"banned"},
				RestrictedModels:   map[string][]string{"premium": {"code.fast"}},
				ModelAliases:       map[string]ModelAlias{"premium": {}, "routine": {}, "banned": {}, "Banned": {}},
				InteractiveDefault: &InteractiveDefault{Model: tc.model, Effort: "high"},
				Roles:              make(map[string]CodingRole),
			}
			for _, role := range []string{"code.default", "code.fast", "code.smart", "code.cheap"} {
				catalog.Roles[role] = CodingRole{Model: "routine", Description: "fixture", Capabilities: []string{"code"}}
			}
			if err := validateCodingRoleCatalog(catalog, "test"); (err == nil) != tc.valid {
				t.Fatalf("valid=%t, validation error=%v", tc.valid, err)
			}
		})
	}
}

func TestLiveCatalogFindingsTreatExcludedModelsAsNamed(t *testing.T) {
	catalog := CodingRoleCatalog{
		SchemaVersion:  CodingRolePolicySchemaVersion,
		Runner:         "codex",
		Provenance:     CatalogProvenance{Source: "fixture", ObservedAt: "2026-09-28"},
		ExcludedModels: []string{"gpt-6-astra", "gpt-5.6-sol"},
		Roles: map[string]CodingRole{
			"code.default": {Model: "gpt-6-luna", Description: "fixture", Capabilities: []string{"code"}},
		},
	}
	live := LiveModelCatalog{Models: []string{"gpt-6-luna", "gpt-6-astra", "gpt-5.6-sol", "future-model"}}

	findings := liveCatalogFindings(catalog, live)
	if len(findings) != 1 || findings[0].Type != "unnamed_live_model" || findings[0].Model != "future-model" {
		t.Fatalf("findings = %+v, want only future-model unnamed", findings)
	}
}

func TestLiveCatalogAbsenceIsNonBlockingUnlessCatalogIsExhaustive(t *testing.T) {
	catalog := CodingRoleCatalog{Roles: map[string]CodingRole{"code.default": {Model: "gpt-6-luna"}}}
	openFindings := liveCatalogFindings(catalog, LiveModelCatalog{Models: []string{"gpt-5.5"}})
	if len(openFindings) == 0 || openFindings[0].Type != "unconfirmed_primary_model" || openFindings[0].Severity != "warning" {
		t.Fatalf("open-world findings = %+v, want non-blocking unconfirmed primary", openFindings)
	}
	if !strings.Contains(openFindings[0].Message, "do not infer that it is unavailable") {
		t.Fatalf("open-world message = %q, want explicit non-availability warning", openFindings[0].Message)
	}
	exhaustiveFindings := liveCatalogFindings(catalog, LiveModelCatalog{Models: []string{"gpt-5.5"}, Exhaustive: true})
	if len(exhaustiveFindings) == 0 || exhaustiveFindings[0].Type != "missing_primary_model" || exhaustiveFindings[0].Severity != "error" {
		t.Fatalf("exhaustive findings = %+v, want blocking missing primary", exhaustiveFindings)
	}
}

// Models a launch can reach without a role (the interactive default and the
// usage-mode targets) are checked against the live catalog like role models.
func TestLiveCatalogFindingsCoverInteractiveDefaultAndUsageTiers(t *testing.T) {
	catalog := CodingRoleCatalog{
		Roles:              map[string]CodingRole{"code.default": {Model: "present"}},
		InteractiveDefault: &InteractiveDefault{Model: "gone-default"},
		UsageTiers:         &UsageTiers{SessionModel: "gone-session", ModelMap: map[string]string{"present": "gone-target"}},
	}
	for name, tc := range map[string]struct {
		live     LiveModelCatalog
		severity string
		prefix   string
	}{
		"exhaustive catalog": {LiveModelCatalog{Models: []string{"present"}, Exhaustive: true}, "error", "missing_"},
		"partial catalog":    {LiveModelCatalog{Models: []string{"present"}}, "warning", "unconfirmed_"},
	} {
		t.Run(name, func(t *testing.T) {
			got := map[string]string{}
			for _, f := range liveCatalogFindings(catalog, tc.live) {
				if f.Severity != tc.severity {
					t.Fatalf("%s severity = %s, want %s", f.Type, f.Severity, tc.severity)
				}
				got[f.Model] = f.Type
			}
			for model, kind := range map[string]string{"gone-default": "interactive_default_model", "gone-session": "usage_tier_model", "gone-target": "usage_tier_model"} {
				if got[model] != tc.prefix+kind {
					t.Fatalf("%s finding = %q, want %q (all: %v)", model, got[model], tc.prefix+kind, got)
				}
			}
		})
	}
	healthy := LiveModelCatalog{Models: []string{"present", "gone-default", "gone-session", "gone-target"}, Exhaustive: true}
	if findings := liveCatalogFindings(catalog, healthy); len(findings) != 0 {
		t.Fatalf("available models produced findings: %+v", findings)
	}
}
