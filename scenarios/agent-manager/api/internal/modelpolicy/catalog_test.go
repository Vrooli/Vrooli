package modelpolicy

import (
	"encoding/json"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

func TestCheckedInCatalogValidatesAndProducesImmutableRevision(t *testing.T) {
	revision, err := Load(ResolvePath())
	if err != nil {
		t.Fatalf("load checked-in catalog: %v", err)
	}
	if !strings.HasPrefix(revision.Digest(), "sha256:") || len(revision.Digest()) != len("sha256:")+64 {
		t.Fatalf("unexpected digest format %q", revision.Digest())
	}

	first := revision.Catalog()
	if first.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", first.SchemaVersion, CurrentSchemaVersion)
	}
	if first.DefaultPolicy != "claude-code.fast" {
		t.Fatalf("default policy = %q", first.DefaultPolicy)
	}
	first.DefaultPolicy = "tampered"
	first.Runners[domain.RunnerTypeCodex] = Inventory{}

	second := revision.Catalog()
	if second.DefaultPolicy == "tampered" || len(second.Runners[domain.RunnerTypeCodex].Models) == 0 {
		t.Fatal("revision exposed mutable catalog state")
	}
}

func TestParseRejectsUnknownJSONField(t *testing.T) {
	raw := marshalCatalog(t, validCatalog())
	raw = []byte(strings.Replace(string(raw), `"schemaVersion":1`, `"schemaVersion":1,"legacyPreset":"FAST"`, 1))
	_, err := Parse(raw)
	assertFieldError(t, err, "unknown field")
}

func TestCatalogValidationDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Catalog)
		wantField string
		wantText  string
	}{
		{
			name: "unknown runner",
			mutate: func(c *Catalog) {
				c.Policies["codex.fast"] = Policy{Intent: PolicyIntentFast, Candidates: []Candidate{{
					Runner: "future-runner", Selection: Selection{Type: SelectionTypeRunnerDefault},
				}}}
			},
			wantField: "modelPolicyCatalog.policies.codex.fast.candidates[0].runner",
			wantText:  "unknown runner",
		},
		{
			name: "unknown model",
			mutate: func(c *Catalog) {
				candidate := c.Policies["codex.fast"]
				candidate.Candidates[0].Selection.Model = "retired-model"
				c.Policies["codex.fast"] = candidate
			},
			wantField: "modelPolicyCatalog.policies.codex.fast.candidates[0].selection.model",
			wantText:  "undeclared model",
		},
		{
			name: "duplicate candidate",
			mutate: func(c *Catalog) {
				policy := c.Policies["codex.fast"]
				policy.Candidates = []Candidate{policy.Candidates[0], policy.Candidates[0]}
				c.Policies["codex.fast"] = policy
			},
			wantField: "modelPolicyCatalog.policies.codex.fast.candidates[1]",
			wantText:  "duplicate candidate",
		},
		{
			name: "unreachable after runner default",
			mutate: func(c *Catalog) {
				policy := c.Policies["codex.fast"]
				policy.Candidates = []Candidate{
					{Runner: domain.RunnerTypeCodex, Selection: Selection{Type: SelectionTypeRunnerDefault}},
					{Runner: domain.RunnerTypeCodex, Selection: Selection{Type: SelectionTypeModel, Model: "gpt-current"}},
				}
				c.Policies["codex.fast"] = policy
			},
			wantField: "modelPolicyCatalog.policies.codex.fast.candidates[1]",
			wantText:  "unreachable",
		},
		{
			name: "empty model is not a runner default sentinel",
			mutate: func(c *Catalog) {
				policy := c.Policies["codex.fast"]
				policy.Candidates[0].Selection.Model = ""
				c.Policies["codex.fast"] = policy
			},
			wantField: "modelPolicyCatalog.policies.codex.fast.candidates[0].selection.model",
			wantText:  "non-empty",
		},
		{
			name: "cheap policy cannot use runner default",
			mutate: func(c *Catalog) {
				policy := c.Policies["codex.fast"]
				policy.Intent = PolicyIntentCheap
				policy.Candidates = []Candidate{{Runner: domain.RunnerTypeCodex, Selection: Selection{Type: SelectionTypeRunnerDefault}}}
				c.Policies["codex.fast"] = policy
			},
			wantField: "modelPolicyCatalog.policies.codex.fast.candidates[0].selection.type",
			wantText:  "cost is unknown",
		},
		{
			name: "invalid default",
			mutate: func(c *Catalog) {
				c.DefaultPolicy = "missing"
			},
			wantField: "modelPolicyCatalog.defaultPolicy",
			wantText:  "unknown policy",
		},
		{
			name: "static model in dynamic namespace",
			mutate: func(c *Catalog) {
				inventory := c.Runners[domain.RunnerTypeCodex]
				inventory.Models = append(inventory.Models, Model{ID: "ollama/local", Description: "invalid static entry"})
				c.Runners[domain.RunnerTypeCodex] = inventory
			},
			wantField: "modelPolicyCatalog.runners.codex.models[1].id",
			wantText:  "dynamic inventory prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := validCatalog()
			tt.mutate(catalog)
			_, err := Parse(marshalCatalog(t, catalog))
			assertFieldError(t, err, tt.wantField)
			assertFieldError(t, err, tt.wantText)
		})
	}
}

func TestCatalogModelIDsReturnsCopy(t *testing.T) {
	catalog := validCatalog()
	ids := catalog.ModelIDs(domain.RunnerTypeCodex)
	ids[0] = "tampered"
	if catalog.Runners[domain.RunnerTypeCodex].Models[0].ID == "tampered" {
		t.Fatal("ModelIDs exposed mutable inventory")
	}
}

func validCatalog() *Catalog {
	return &Catalog{
		SchemaVersion: CurrentSchemaVersion,
		Metadata: Metadata{
			CatalogID: "test-catalog",
			UpdatedAt: "2026-07-09",
			Sources: []Source{{
				Name: "test", Reference: "fixture", VerifiedAt: "2026-07-09",
			}},
		},
		DefaultPolicy: "codex.fast",
		Runners: map[domain.RunnerType]Inventory{
			domain.RunnerTypeCodex: {
				Models:                []Model{{ID: "gpt-current", Description: "Current model"}},
				SupportsRunnerDefault: true,
				DynamicModelPrefixes:  []string{"ollama/"},
			},
		},
		Policies: map[string]Policy{
			"codex.fast": {
				Intent: PolicyIntentFast,
				Candidates: []Candidate{
					{Runner: domain.RunnerTypeCodex, Selection: Selection{Type: SelectionTypeModel, Model: "gpt-current"}},
					{Runner: domain.RunnerTypeCodex, Selection: Selection{Type: SelectionTypeRunnerDefault}},
				},
			},
		},
	}
}

func marshalCatalog(t *testing.T, catalog *Catalog) []byte {
	t.Helper()
	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	return raw
}

func assertFieldError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err, want)
	}
}
