package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-gateway/internal/rolecatalog"
)

func TestParseResourceRoles(t *testing.T) {
	roles, err := parseResourceRoles([]byte(`{"roles":{"chat.default":{},"vision.default":{}}}`))
	if err != nil {
		t.Fatalf("parseResourceRoles() error = %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("parseResourceRoles() = %v", roles)
	}
	if _, err := parseResourceRoles([]byte(`{"roles":{}}`)); err == nil || !strings.Contains(err.Error(), "declares no roles") {
		t.Fatalf("parseResourceRoles() error = %v, want empty policy rejection", err)
	}
	if _, err := parseResourceRoles([]byte(`{`)); err == nil || !strings.Contains(err.Error(), "parse model policy") {
		t.Fatalf("parseResourceRoles() error = %v, want parse failure", err)
	}
}

func TestValidateRoleCandidates(t *testing.T) {
	resourceRoles := map[string]map[string]struct{}{
		"ollama": {"chat.default": {}, "vision.default": {}},
		"ollaya": {"decide.local": {}},
		"lpbs":   {"anything": {}},
	}
	valid := rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
		"classify.fast": {Candidates: []rolecatalog.Candidate{
			{Provider: "ollama", ResourceRole: "chat.default", Reason: "local"},
		}},
		"decide.fast": {
			Kind: rolecatalog.RoleKindDecision,
			Candidates: []rolecatalog.Candidate{
				{Provider: "ollaya", ResourceRole: "decide.local", Reason: "local"},
				{Provider: "ai-gateway", ResourceRole: "classify.fast", Reason: "typed fallback"},
			},
		},
		"metered.fallback": {Candidates: []rolecatalog.Candidate{
			{Provider: "lpbs", ResourceRole: "no-policy-needed", Reason: "metered"},
		}},
	}}
	if err := validateRoleCandidates(valid, resourceRoles); err != nil {
		t.Fatalf("validateRoleCandidates() error = %v", err)
	}

	tests := []struct {
		name    string
		catalog rolecatalog.RoleCatalog
		wantErr string
	}{
		{
			name: "unknown provider",
			catalog: rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
				"classify.fast": {Candidates: []rolecatalog.Candidate{{Provider: "not-a-provider", ResourceRole: "chat.default", Reason: "typo"}}},
			}},
			wantErr: `provider "not-a-provider" is not a registered provider`,
		},
		{
			name: "missing resource role",
			catalog: rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
				"classify.fast": {Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.missing", Reason: "typo"}}},
			}},
			wantErr: `resource_role "chat.missing" is missing from resources/ollama/model-policy.json`,
		},
		{
			name: "unknown gateway role",
			catalog: rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
				"decide.fast": {
					Kind:       rolecatalog.RoleKindDecision,
					Candidates: []rolecatalog.Candidate{{Provider: "ai-gateway", ResourceRole: "classify.missing", Reason: "typo"}},
				},
				"classify.fast": {Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "local"}}},
			}},
			wantErr: `names undeclared gateway role "classify.missing"`,
		},
		{
			name: "gateway decision role",
			catalog: rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
				"decide.fast": {
					Kind:       rolecatalog.RoleKindDecision,
					Candidates: []rolecatalog.Candidate{{Provider: "ai-gateway", ResourceRole: "decide.slow", Reason: "wrong kind"}},
				},
				"decide.slow": {
					Kind:       rolecatalog.RoleKindDecision,
					Candidates: []rolecatalog.Candidate{{Provider: "ollaya", ResourceRole: "decide.local", Reason: "local"}},
				},
			}},
			wantErr: `names decision gateway role "decide.slow"`,
		},
		{
			name: "qualification candidate typo",
			catalog: rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
				"classify.fast": {
					Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "local"}},
					QualificationCandidates: []rolecatalog.QualificationCandidate{{
						Candidate: rolecatalog.Candidate{Provider: "ollama", ResourceRole: "chat.qualification-typo", Reason: "pending"},
						Status:    rolecatalog.QualificationPending,
					}},
				},
			}},
			wantErr: `qualification candidate resource_role "chat.qualification-typo"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRoleCandidates(test.catalog, resourceRoles)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateRoleCandidates() error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateServiceModelRoles(t *testing.T) {
	resourceRoles := map[string]map[string]struct{}{
		"ollama":     {"chat.default": {}, "vision.default": {}},
		"openrouter": {"extract.structured": {}, "vision.default": {}},
	}
	validService := `{"dependencies":{"resources":{
		"ollama":{"type":"ollama","model_roles":["chat.default",{"role":"vision.default"}]},
		"openrouter":{"type":"openrouter","model_roles":["extract.structured"]}
	}}}`

	t.Run("valid", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, serviceRelPath, validService)
		if err := validateServiceModelRoles(root, resourceRoles); err != nil {
			t.Fatalf("validateServiceModelRoles() error = %v", err)
		}
	})

	t.Run("gateway role is not a resource role", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, serviceRelPath, `{"dependencies":{"resources":{
			"ollama":{"type":"ollama","model_roles":["locate.visual"]}
		}}}`)
		if err := validateServiceModelRoles(root, resourceRoles); err == nil {
			t.Fatal("validateServiceModelRoles() accepted a gateway role that the resource does not declare")
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, serviceRelPath, `{"dependencies":{"resources":{
			"mystery":{"type":"mystery"}
		}}}`)
		err := validateServiceModelRoles(root, resourceRoles)
		if err == nil || !strings.Contains(err.Error(), `dependency resource "mystery" is not a registered provider`) {
			t.Fatalf("validateServiceModelRoles() error = %v", err)
		}
	})

	t.Run("unknown model role", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, serviceRelPath, `{"dependencies":{"resources":{
			"ollama":{"type":"ollama","model_roles":["chat.typo"]}
		}}}`)
		err := validateServiceModelRoles(root, resourceRoles)
		if err == nil || !strings.Contains(err.Error(), `model_roles entry "chat.typo" is missing from resources/ollama/model-policy.json`) {
			t.Fatalf("validateServiceModelRoles() error = %v", err)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, serviceRelPath, `{`)
		err := validateServiceModelRoles(root, resourceRoles)
		if err == nil || !strings.Contains(err.Error(), "parse "+serviceRelPath) {
			t.Fatalf("validateServiceModelRoles() error = %v", err)
		}
	})
}

func TestAddSupplierEntriesEnforcesSingleDeclarationOwner(t *testing.T) {
	validSupplier := `{"id":"hosted-supplier","family":"openai-compatible","locality":"remote","endpoint":"https://example.invalid/v1","policy":{"roles":{}}}`
	t.Run("wrapperless supplier is accepted", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, "scenarios/ai-gateway/config/suppliers/hosted.json", validSupplier)
		entries := map[string]any{}
		if err := addSupplierEntries(root, entries); err != nil {
			t.Fatalf("addSupplierEntries() error = %v", err)
		}
		if _, ok := entries["hosted-supplier"]; !ok {
			t.Fatal("supplier declaration was not projected")
		}
	})

	tests := []struct {
		name, content, path, want string
	}{
		{name: "registered provider even when not projected", content: `{"id":"ollaya","family":"openai-compatible","locality":"remote","endpoint":"https://example.invalid","policy":{}}`, path: "hosted.json", want: "duplicates registered provider declaration"},
		{name: "resource backed declaration", content: `{"id":"unregistered-resource","family":"openai-compatible","locality":"remote","endpoint":"https://example.invalid","policy":{}}`, path: "hosted.json", want: "duplicates resource declaration"},
		{name: "invalid locality", content: `{"id":"hosted-supplier","family":"openai-compatible","locality":"regional","endpoint":"https://example.invalid","policy":{}}`, path: "hosted.json", want: "invalid supplier declaration"},
		{name: "invalid policy JSON", content: `{"id":"hosted-supplier","family":"openai-compatible","locality":"remote","endpoint":"https://example.invalid","policy":not-json}`, path: "hosted.json", want: "invalid supplier declaration"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFixtureFile(t, root, "scenarios/ai-gateway/config/suppliers/"+test.path, test.content)
			if test.name == "resource backed declaration" {
				writeFixtureFile(t, root, "resources/unregistered-resource/resource.json", `{}`)
			}
			err := addSupplierEntries(root, map[string]any{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("addSupplierEntries() error = %v, want containing %q", err, test.want)
			}
		})
	}

	t.Run("duplicate supplier ids", func(t *testing.T) {
		root := t.TempDir()
		writeFixtureFile(t, root, "scenarios/ai-gateway/config/suppliers/first.json", validSupplier)
		writeFixtureFile(t, root, "scenarios/ai-gateway/config/suppliers/second.json", validSupplier)
		err := addSupplierEntries(root, map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "is declared by both") {
			t.Fatalf("addSupplierEntries() error = %v, want duplicate supplier rejection", err)
		}
	})
}

func TestRunRepositoryIntegration(t *testing.T) {
	root := repositoryRoot(t)
	output := filepath.Join(root, "scenarios", "ai-gateway", "api", "internal", "catalog", "providers.generated.json")
	if err := run(root, output, true); err != nil {
		t.Fatalf("run(real repository, --check) error = %v", err)
	}
}

func TestRunFixtureRoot(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		root := fixtureRoot(t, validFixtureRoleCatalog)
		output := filepath.Join(root, "providers.generated.json")
		if err := run(root, output, false); err != nil {
			t.Fatalf("run(fixture) error = %v", err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var bundle struct {
			Providers map[string]json.RawMessage `json:"providers"`
		}
		if err := json.Unmarshal(data, &bundle); err != nil {
			t.Fatalf("parse generated output: %v", err)
		}
		for _, provider := range []string{"ollama", "openrouter", "orcarouter", "abliteration"} {
			if _, ok := bundle.Providers[provider]; !ok {
				t.Errorf("generated output omitted projected provider %q", provider)
			}
		}
	})

	t.Run("role candidate typo fails generation", func(t *testing.T) {
		root := fixtureRoot(t, strings.Replace(validFixtureRoleCatalog, `"chat.default"`, `"chat.missing"`, 1))
		err := run(root, filepath.Join(root, "providers.generated.json"), false)
		if err == nil || !strings.Contains(err.Error(), `resource_role "chat.missing" is missing from resources/ollama/model-policy.json`) {
			t.Fatalf("run(fixture with typo) error = %v", err)
		}
	})

	t.Run("service model role typo fails generation", func(t *testing.T) {
		root := fixtureRoot(t, validFixtureRoleCatalog)
		writeFixtureFile(t, root, serviceRelPath, `{"dependencies":{"resources":{
			"ollama":{"type":"ollama","model_roles":["chat.typo"]}
		}}}`)
		err := run(root, filepath.Join(root, "providers.generated.json"), false)
		if err == nil || !strings.Contains(err.Error(), `model_roles entry "chat.typo" is missing from resources/ollama/model-policy.json`) {
			t.Fatalf("run(fixture with service typo) error = %v", err)
		}
	})
}

const validFixtureRoleCatalog = `{
  "schema_version": 1,
  "roles": {
    "classify.fast": {
      "description": "fixture role",
      "timeout_ms": 1000,
      "candidates": [
        {"provider": "ollama", "resource_role": "chat.default", "reason": "fixture local candidate"}
      ]
    }
  }
}`

func fixtureRoot(t *testing.T, roleCatalog string) string {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, roleCatalogRelPath, roleCatalog)
	writeFixtureFile(t, root, "packages/monetization-go/ai-features.generated.json", `{"version":1,"features":[]}`)
	writeFixtureFile(t, root, "packages/monetization-go/credit-features.generated.json", `{"version":1,"features":[]}`)
	writeFixtureFile(t, root, serviceRelPath, `{"dependencies":{"resources":{
		"ollama":{"type":"ollama","model_roles":["chat.default"]}
	}}}`)
	for _, provider := range []string{"ollama", "openrouter", "orcarouter", "abliteration", "pii-detector", "ollaya"} {
		writeFixtureFile(t, root, filepath.Join("resources", provider, "model-policy.json"), `{"roles":{"chat.default":{}}}`)
	}
	for _, provider := range []string{"ollama", "openrouter", "orcarouter", "abliteration"} {
		writeFixtureFile(t, root, filepath.Join("resources", provider, "resource.json"), `{"endpoint":"https://example.invalid"}`)
	}
	return root
}

func writeFixtureFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if pathExists(filepath.Join(dir, ".git")) || pathExists(filepath.Join(dir, "VISION.md")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root from gen-provider-catalog package")
		}
		dir = parent
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
