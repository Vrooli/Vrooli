package inference

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ai-gateway/internal/rolecatalog"
)

func TestCatalogRejectsStaleConcreteModelSelector(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rolecatalog.LoadCatalog(path); err != nil {
		t.Fatalf("load maintained catalog: %v", err)
	}

	legacy := strings.Replace(string(contents), `"provider": "ollama",`, `"provider": "ollama", "model": "stale:tag",`, 1)
	if legacy == string(contents) {
		t.Fatal("test fixture did not find the first Ollama candidate")
	}
	legacyPath := filepath.Join(t.TempDir(), "legacy-catalog.json")
	if err := os.WriteFile(legacyPath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := rolecatalog.LoadCatalog(legacyPath); err == nil || !strings.Contains(err.Error(), `unknown field "model"`) {
		t.Fatalf("rolecatalog.LoadCatalog() error = %v, want stale model selector rejection", err)
	}
}

func TestCatalogRejectsParallelModelAndQuantizationSelectors(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	needle := `"resource_role": "chat.small"`
	if !strings.Contains(string(contents), needle) {
		t.Fatal("maintained catalog has no compact qualification candidate")
	}
	for _, tc := range []struct {
		name  string
		field string
	}{
		{name: "model", field: `"model": "qwen3.5:4b"`},
		{name: "quantization", field: `"quantization": "Q4_K_M"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy := strings.Replace(string(contents), needle, needle+", "+tc.field, 1)
			legacyPath := filepath.Join(t.TempDir(), "legacy-catalog.json")
			if err := os.WriteFile(legacyPath, []byte(legacy), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := rolecatalog.LoadCatalog(legacyPath); err == nil || !strings.Contains(err.Error(), `unknown field "`+tc.name+`"`) {
				t.Fatalf("rolecatalog.LoadCatalog() error = %v; want rejection of duplicated %s metadata", err, tc.name)
			}
		})
	}
}

func TestCatalogDeclaresBoundedCodeRoleOutsideProviderCandidates(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	catalog, err := rolecatalog.LoadCatalog(path)
	if err != nil {
		t.Fatalf("load maintained catalog: %v", err)
	}
	contract, ok := catalog.CodeRoles["policy.inventory-reservation"]
	if !ok {
		t.Fatal("policy.inventory-reservation is not declared as a code role")
	}
	if contract.HandlerID != "inventory-reservation-policy/v1" || contract.MaxSourceBytes != 4096 || contract.MaxOutputBytes != 4096 {
		t.Fatalf("code role contract = %+v; want versioned handler and bounded 4 KiB I/O", contract)
	}
	if len(catalog.Roles["policy.inventory-reservation"].Candidates) != 0 {
		t.Fatal("code role unexpectedly has a provider candidate list")
	}
	got := catalog.RoleNames()
	containsCodeRole := false
	for _, role := range got {
		containsCodeRole = containsCodeRole || role == "policy.inventory-reservation"
	}
	if !sort.StringsAreSorted(got) || !containsCodeRole {
		t.Fatalf("catalog role names = %v; want sorted names including the code role", got)
	}
}

func TestCodeRoleValidationRejectsUnboundedOrAmbiguousContracts(t *testing.T) {
	valid := rolecatalog.CodeRoleContract{
		HandlerID:      "inventory-reservation-policy/v1",
		MaxSourceBytes: 4096,
		MaxOutputBytes: 4096,
		SchemaContract: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{"type": "string"},
				"net":    map[string]any{"type": "integer"},
			},
			"required":             []any{"action", "net"},
			"additionalProperties": false,
		},
	}
	base := func(contract rolecatalog.CodeRoleContract) rolecatalog.RoleCatalog {
		return rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
			"extract.structured": {Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "incumbent"}}},
		}, CodeRoles: map[string]rolecatalog.CodeRoleContract{"policy.inventory-reservation": contract}}
	}
	if err := base(valid).Validate(); err != nil {
		t.Fatalf("validate bounded code role: %v", err)
	}

	tooLarge := valid
	tooLarge.MaxSourceBytes = rolecatalog.MaxCodeRoleSourceBytes + 1
	if err := base(tooLarge).Validate(); err == nil || !strings.Contains(err.Error(), "max_source_bytes") {
		t.Fatalf("validate oversized code role error = %v; want max_source_bytes refusal", err)
	}
	invalidSchema := valid
	invalidSchema.SchemaContract = map[string]any{"type": "string"}
	if err := base(invalidSchema).Validate(); err == nil || !strings.Contains(err.Error(), "root type must be object") {
		t.Fatalf("validate non-object code schema error = %v; want object schema refusal", err)
	}
	overlap := base(valid)
	overlap.CodeRoles["extract.structured"] = valid
	if err := overlap.Validate(); err == nil || !strings.Contains(err.Error(), "both a provider role and a code role") {
		t.Fatalf("validate overlapping role error = %v; want duplicate role refusal", err)
	}
}

func TestCatalogKeepsCompactQualificationCandidatesOutOfRuntimeSelection(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	catalog, err := rolecatalog.LoadCatalog(path)
	if err != nil {
		t.Fatalf("load maintained catalog: %v", err)
	}

	extract := catalog.Roles["extract.structured"]
	if len(extract.Candidates) == 0 || extract.Candidates[0].ResourceRole != "chat.default" {
		t.Fatalf("active extract candidate = %+v; want incumbent chat.default pending qualification evidence", extract.Candidates)
	}
	if len(extract.QualificationCandidates) != 1 {
		t.Fatalf("extract qualification candidates = %+v; want one compact candidate", extract.QualificationCandidates)
	}
	qualified := extract.QualificationCandidates[0]
	if qualified.Candidate.Provider != "ollama" || qualified.Candidate.ResourceRole != "chat.small" || qualified.Status != rolecatalog.QualificationPending {
		t.Fatalf("extract qualification candidate = %+v; want pending ollama/chat.small", qualified)
	}

	summary := catalog.Roles["summarize.faithful"]
	if len(summary.Candidates) == 0 || summary.Candidates[0].ResourceRole != "summarize.default" {
		t.Fatalf("active summary candidate = %+v; want incumbent summarize.default pending caller evidence", summary.Candidates)
	}
	if len(summary.QualificationCandidates) != 1 || summary.QualificationCandidates[0].Candidate.ResourceRole != "chat.small" || summary.QualificationCandidates[0].Status != rolecatalog.QualificationPending {
		t.Fatalf("summary qualification candidate = %+v; want pending chat.small", summary.QualificationCandidates)
	}

	classification := catalog.Roles["classify.fast"]
	if len(classification.Candidates) == 0 || classification.Candidates[0].Provider != "ollama" || classification.Candidates[0].ResourceRole != "classify.routing" {
		t.Fatalf("active classification lane = %+v; want local classify.routing incumbent", classification.Candidates)
	}
}

func TestCatalogKeepsSemanticEscalationSeparateFromOrdinaryCandidates(t *testing.T) {
	role := rolecatalog.InferenceRole{
		Candidates:                   []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.small", Reason: "compact primary"}},
		SemanticEscalationCandidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "explicit larger lane"}},
	}
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{"extract.structured": role}}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate separated escalation stage: %v", err)
	}

	role.SemanticEscalationCandidates = []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.small", Reason: "duplicate retry target"}}
	catalog.Roles["extract.structured"] = role
	if err := catalog.Validate(); err == nil || !strings.Contains(err.Error(), "repeats ordinary candidate") {
		t.Fatalf("validate duplicate escalation candidate error = %v; want duplicate-stage rejection", err)
	}
}

func TestCatalogQualificationRequiresEvidenceAfterDecision(t *testing.T) {
	role := rolecatalog.InferenceRole{
		Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "incumbent"}},
		QualificationCandidates: []rolecatalog.QualificationCandidate{{
			Candidate: rolecatalog.Candidate{Provider: "ollama", ResourceRole: "chat.small", Reason: "compact candidate"},
			Status:    rolecatalog.QualificationAccepted,
		}},
	}
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{"extract.structured": role}}
	if err := catalog.Validate(); err == nil || !strings.Contains(err.Error(), "requires evidence") {
		t.Fatalf("validate evidence-less qualification error = %v; want evidence requirement", err)
	}
	role.QualificationCandidates[0].Evidence = "runs/lm4-extract-qualification/LEDGER.md#case-1"
	catalog.Roles["extract.structured"] = role
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate qualification with evidence: %v", err)
	}
}
