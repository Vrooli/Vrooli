package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ai-gateway/internal/catalog"
	"ai-gateway/internal/rolecatalog"
	"ai-gateway/internal/routing"
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
	validSupplier := `{"id":"hosted-supplier","family":"openai-compatible","locality":"remote","endpoint":"https://example.invalid/v1","policy":` + string(ac2SupplierFixturePolicy("hosted-supplier", "proposed", "[]", "[]", "{}")) + `}`
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
		{name: "registered provider even when not projected", content: strings.ReplaceAll(validSupplier, "hosted-supplier", "ollaya"), path: "hosted.json", want: "duplicates registered provider declaration"},
		{name: "resource backed declaration", content: strings.ReplaceAll(validSupplier, "hosted-supplier", "unregistered-resource"), path: "hosted.json", want: "duplicates resource declaration"},
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

func TestAC2ProposedNanoGPTDeclarationProjectsWithoutRoutesOrCredentialValues(t *testing.T) {
	root := repositoryRoot(t)
	inputs, err := loadInputs(root)
	if err != nil {
		t.Fatalf("loadInputs() error = %v", err)
	}
	entries, err := providerEntries(root, inputs)
	if err != nil {
		t.Fatalf("providerEntries() error = %v", err)
	}
	if err := addSupplierEntries(root, entries); err != nil {
		t.Fatalf("addSupplierEntries() error = %v", err)
	}

	raw, ok := entries["nanogpt"]
	if !ok {
		t.Fatal("NanoGPT supplier declaration missing from projected provider catalog")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var declaration struct {
		ID       string `json:"id"`
		Endpoint string `json:"endpoint"`
		Policy   struct {
			Provider struct {
				Terms struct {
					Review struct {
						Status string `json:"status"`
					} `json:"review"`
				} `json:"terms"`
				Routes []json.RawMessage `json:"hosted_routes"`
			} `json:"provider"`
			Roles map[string]json.RawMessage `json:"roles"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(data, &declaration); err != nil {
		t.Fatalf("parse projected declaration: %v", err)
	}
	if declaration.ID != "nanogpt" || declaration.Endpoint != "https://api.nano-gpt.com/api/v1" {
		t.Fatalf("projected identity/endpoint = %q/%q", declaration.ID, declaration.Endpoint)
	}
	if declaration.Policy.Provider.Terms.Review.Status != "proposed" || len(declaration.Policy.Provider.Routes) != 0 || len(declaration.Policy.Roles) != 0 {
		t.Fatalf("projection activated a proposal: status=%q routes=%d roles=%d", declaration.Policy.Provider.Terms.Review.Status, len(declaration.Policy.Provider.Routes), len(declaration.Policy.Roles))
	}
	for _, secretLike := range []string{"NANOGPT_API_KEY", "api_key", "token"} {
		if strings.Contains(string(data), secretLike) {
			t.Fatalf("supplier projection unexpectedly contains key material or a credential field %q", secretLike)
		}
	}
	routerOneRaw, ok := entries["router-one"]
	if !ok {
		t.Fatal("Router One PAYG supplier declaration missing from projected provider catalog")
	}
	routerOneData, err := json.Marshal(routerOneRaw)
	if err != nil {
		t.Fatal(err)
	}
	var routerOne struct {
		ID       string `json:"id"`
		Endpoint string `json:"endpoint"`
		Policy   struct {
			Provider struct {
				Terms struct {
					Review struct {
						Status string `json:"status"`
					} `json:"review"`
				} `json:"terms"`
				Routes     []json.RawMessage `json:"hosted_routes"`
				Agreements []json.RawMessage `json:"agreements"`
			} `json:"provider"`
			Roles  map[string]json.RawMessage `json:"roles"`
			Models map[string]struct {
				DefaultEligible *bool    `json:"default_eligible"`
				Caveats         []string `json:"caveats"`
			} `json:"models"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(routerOneData, &routerOne); err != nil {
		t.Fatalf("parse projected Router One declaration: %v", err)
	}
	if routerOne.ID != "router-one" || routerOne.Endpoint != "https://api.router.one/v1" || routerOne.Policy.Provider.Terms.Review.Status != "proposed" || len(routerOne.Policy.Provider.Routes) != 0 || len(routerOne.Policy.Provider.Agreements) != 0 || len(routerOne.Policy.Roles) != 0 || len(routerOne.Policy.Models) != 3 {
		t.Fatalf("Router One projection activated proposal or lost sourced model drafts: %+v", routerOne)
	}
	for id, model := range routerOne.Policy.Models {
		if model.DefaultEligible == nil || *model.DefaultEligible || len(model.Caveats) < 2 {
			t.Errorf("Router One model %q lacks its ineligible source-rights caveat: %+v", id, model)
		}
	}
}

func TestAC2SupplierDeclarationGeneratesInIsolatedJ13BundleFixture(t *testing.T) {
	root := fixtureRoot(t, validFixtureRoleCatalog)
	declarationDir := filepath.Join(repositoryRoot(t), "scenarios", "ai-gateway", "config", "suppliers")
	declarations, err := os.ReadDir(declarationDir)
	if err != nil {
		t.Fatalf("read repository supplier declarations: %v", err)
	}
	candidateIDs := map[string]bool{}
	for _, declaration := range declarations {
		if declaration.IsDir() || filepath.Ext(declaration.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(declarationDir, declaration.Name()))
		if err != nil {
			t.Fatalf("read supplier declaration %q: %v", declaration.Name(), err)
		}
		id := strings.TrimSuffix(declaration.Name(), filepath.Ext(declaration.Name()))
		candidateIDs[id] = true
		writeFixtureFile(t, root, filepath.Join("scenarios", "ai-gateway", "config", "suppliers", declaration.Name()), string(data))
	}
	if len(candidateIDs) != 18 {
		t.Fatalf("isolated J13 fixture supplier declaration count = %d, want 18", len(candidateIDs))
	}
	output := filepath.Join(t.TempDir(), "providers.generated.json")
	if err := run(root, output, false); err != nil {
		t.Fatalf("run(isolated J13 fixture) error = %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var packaged catalog.Bundle
	if err := json.Unmarshal(data, &packaged); err != nil {
		t.Fatalf("decode generated J13 bundle: %v", err)
	}
	for id := range candidateIDs {
		provider, ok := packaged.Providers[id]
		if !ok {
			t.Errorf("isolated generated J13 bundle omitted candidate %q", id)
			continue
		}
		if provider.Endpoint == "" {
			if _, err := (routing.ResourcePolicySource{Catalog: &packaged}).Endpoint(context.Background(), id); err == nil {
				t.Errorf("endpoint-less packaged candidate %q resolved an endpoint", id)
			}
		}
	}
	apmix, ok := packaged.Providers["apmix"]
	if !ok {
		t.Fatal("isolated generated J13 bundle omitted APmix")
	}
	if apmix.Endpoint != "https://api.apmix.ai/v1" {
		t.Fatalf("isolated J13 bundle APmix endpoint = %q; want documented API base", apmix.Endpoint)
	}
	var apmixPolicy struct {
		Provider struct {
			Terms struct {
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models map[string]struct {
			DefaultEligible *bool `json:"default_eligible"`
			Provenance      map[string]struct {
				ObservedAt string `json:"observed_at"`
			} `json:"provenance"`
		} `json:"models"`
	}
	if err := json.Unmarshal(apmix.Policy, &apmixPolicy); err != nil {
		t.Fatalf("decode APmix policy from isolated generated J13 bundle: %v", err)
	}
	if len(apmixPolicy.Models) != 24 || apmixPolicy.Provider.Terms.Review.Status != "proposed" || len(apmixPolicy.Provider.Routes) != 0 || len(apmixPolicy.Provider.Agreements) != 0 {
		t.Fatalf("APmix isolated J13 projection = models:%d review:%q routes:%d agreements:%d; want 24 draft identities and no executable authority", len(apmixPolicy.Models), apmixPolicy.Provider.Terms.Review.Status, len(apmixPolicy.Provider.Routes), len(apmixPolicy.Provider.Agreements))
	}
	for id, model := range apmixPolicy.Models {
		if model.DefaultEligible == nil || *model.DefaultEligible {
			t.Errorf("APmix isolated J13 model %q is not explicitly ineligible", id)
		}
		if model.Provenance["catalog"].ObservedAt != "2026-11-23" || model.Provenance["terms"].ObservedAt != "2026-11-23" {
			t.Errorf("APmix isolated J13 model %q lost dated provenance: %+v", id, model.Provenance)
		}
	}
	snapshot, err := (routing.ResourcePolicySource{Catalog: &packaged}).Snapshot(context.Background(), "recorded-account")
	if err != nil {
		t.Fatalf("load generated J13 bundle into packaged policy source: %v", err)
	}
	apmixEndpoint, err := (routing.ResourcePolicySource{Catalog: &packaged}).Endpoint(context.Background(), "apmix")
	if err != nil {
		t.Fatalf("resolve APmix transport metadata from generated J13 bundle: %v", err)
	}
	if apmixEndpoint.URL != "https://api.apmix.ai/v1" || apmixEndpoint.APIKey != "" {
		t.Fatalf("APmix J13 bundle endpoint=%q API-key-present=%t; want documented transport base and no credential", apmixEndpoint.URL, apmixEndpoint.APIKey != "")
	}
	seen := map[string]bool{}
	for _, record := range snapshot.Records {
		seen[record.Supplier] = true
		if candidateIDs[record.Supplier] && record.Terms.Review.Status != "proposed" {
			t.Errorf("packaged candidate %q review status = %q, want proposed", record.Supplier, record.Terms.Review.Status)
		}
	}
	for id := range candidateIDs {
		if !seen[id] {
			t.Errorf("packaged policy source omitted candidate record %q", id)
		}
	}
	for _, route := range snapshot.Routes {
		if candidateIDs[route.Supplier] {
			t.Errorf("packaged proposal candidate %q produced executable route %q", route.Supplier, route.ID)
		}
	}
	for _, agreement := range snapshot.Agreements {
		if candidateIDs[agreement.Supplier] {
			t.Errorf("packaged proposal candidate %q produced executable agreement", agreement.Supplier)
		}
	}
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
		policy := `{"roles":{"chat.default":{}}}`
		if provider == "openrouter" {
			policy = `{"roles":{"chat.default":{}},"provider":{"id":"openrouter","terms":{"plan_rules":[{"plan":"standard","clause":"Terms §7","reason":"Standard resale prohibited.","use_classes":{"in_app":"denied"},"conditions":["separate agreement required"],"source_anchors":[{"supplier_id":"openrouter","claim_ref":"standard","url":"https://openrouter.ai/terms","anchor":"Terms §7","document_date":"2026-08-31","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"Standard resale prohibited.","does_not_establish":"enterprise rights remain unknown"}]}],"evidence":{"url":"https://openrouter.ai/terms","document_date":"2026-08-31","read_at":"2026-11-23"}},"hosted_routes":[]},"models":{"fixture-openrouter-model":{"caveats":["This model's upstream terms have not been evaluated."],"provenance":{"terms":{"source":"https://openrouter.ai/terms; upstream model terms unreviewed","observed_at":"2026-11-23","source_anchors":[{"supplier_id":"openrouter","model_id":"fixture-openrouter-model","url":"https://openrouter.ai/terms","anchor":"Terms §§5.1–5.2","document_date":"2026-08-31","observed_at":"2026-11-23","retrieval_status":"read_public_page","supports":"Model-specific terms apply.","does_not_establish":"model rights remain unknown"}]},"identity":{"source":"https://example.invalid/models/fixture-openrouter-model; exact fixture-openrouter-model ID","observed_at":"2026-11-23"}}}}}`
		}
		writeFixtureFile(t, root, filepath.Join("resources", provider, "model-policy.json"), policy)
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
