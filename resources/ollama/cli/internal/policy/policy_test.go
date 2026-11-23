package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
)

func TestLoadFileRejectsRetiredRuntimeContextTierPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, []byte(`{"roles":{"chat.default":{"runtime_policy":{"context_tiers":[4096]}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "runtime_policy is unsupported") {
		t.Fatalf("LoadFile() error = %v, want explicit retired-context-policy refusal", err)
	}
}

func TestLoadFile_CurrentPolicyIsValid(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got := p.Roles["embedding.default"].Model; got != "nomic-embed-text:latest" {
		t.Fatalf("embedding.default model = %q", got)
	}
	if got := p.Models["nomic-embed-text:latest"].EmbeddingDimensions; got != 768 {
		t.Fatalf("nomic embedding dimensions = %d", got)
	}
}

func TestAllFifteenOllamaModelsHaveExplicitRightsDisposition(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Models) != 15 {
		t.Fatalf("model census=%d, want 15", len(p.Models))
	}
	ctx := sharedpolicy.RightsContext{ID: "operator-stated-business-context", Version: "2026-11-14", EntityScope: "operator-group", RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement"}
	now := time.Date(2026, 11, 14, 17, 0, 0, 0, time.UTC)
	for name, model := range p.Models {
		if model.Rights == nil {
			t.Errorf("models.%s has no explicit rights disposition", name)
			continue
		}
		if model.Rights.Classification == sharedpolicy.RightsUnverified && strings.TrimSpace(model.Rights.UnverifiedReason) == "" {
			t.Errorf("models.%s is unknown without an evidence-gap reason", name)
		}
		if len(model.Rights.ReviewTriggers) == 0 {
			t.Errorf("models.%s has no rights review triggers", name)
		}
		decision := sharedpolicy.EvaluateRights(model.Rights, sharedpolicy.RightsUseLocalInference, ctx, now)
		if model.Rights.Classification == sharedpolicy.RightsUnverified && decision.Status != sharedpolicy.RightsUnknown {
			t.Errorf("models.%s unknown source facts evaluated as %s", name, decision.Status)
		}
	}
}

func TestValidateRejectsMalformedRuntimeArtifactDigest(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	ref := p.Roles["chat.small"].Model
	model := p.Models[ref]
	model.Rights = &sharedpolicy.ArtifactRights{ArtifactID: "fixture/model", Revision: "fixture-v1"}
	model.RuntimeIdentity = &sharedpolicy.RuntimeArtifactIdentity{
		Provider: sharedpolicy.RuntimeProviderOllama, Model: ref,
		ArtifactID: model.Rights.ArtifactID, ArtifactRevision: model.Rights.Revision,
		DigestKind: sharedpolicy.RuntimeDigestOllamaModelContents, Digest: "not-a-sha256",
		Source: sharedpolicy.RuntimeSourceOllamaTags, ObservedAt: "2026-11-11",
	}
	p.Models[ref] = model
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "runtime_identity") {
		t.Fatalf("malformed runtime digest must be rejected, got %v", err)
	}
}

func TestResolveRoleWithRightsFailsClosedWhenArtifactEvidenceIsUnknown(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Keep the unknown-evidence branch explicit even as current catalog
	// entries gain verified rights records.
	model := p.Models[p.Roles["chat.small"].Model]
	model.Rights = nil
	model.RuntimeIdentity = nil
	p.Models[p.Roles["chat.small"].Model] = model
	resolved, err := p.ResolveRoleWithRights("chat.small", sharedpolicy.RightsUseLocalInference, sharedpolicy.RightsContext{
		ID: "operator-business-context/current", Version: "fixture-v1", EntityScope: "operator-group",
		RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement",
	}, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.RightsDecision == nil || resolved.RightsDecision.Status != sharedpolicy.RightsUnknown {
		t.Fatalf("unattested catalog model was not unknown: %+v", resolved.RightsDecision)
	}
}

func TestQwen35FourBUnrestrictedRightsBindCurrentRuntime(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := sharedpolicy.RightsContext{
		ID: "operator-stated-business-context", Version: "2026-11-10",
		RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement",
	}
	resolved, err := p.ResolveRoleWithRights("chat.small", sharedpolicy.RightsUseLocalInference, ctx, time.Date(2026, 11, 14, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Model != "qwen3.5:4b" || resolved.RightsDecision == nil || resolved.RightsDecision.Status != sharedpolicy.RightsAllowed {
		t.Fatalf("chat.small rights decision = %+v; want the catalogued unconditional qwen grant", resolved.RightsDecision)
	}
	modelFacts := p.Models[resolved.Model]
	if err := requireObservedRuntimeIdentity(modelFacts, resolved.Model, "2a654d98e6fba55d452b7043684e9b57a947e393bbffa62485a7aac05ee4eefd"); err != nil {
		t.Fatalf("exact current model digest was not accepted: %v", err)
	}
	if err := requireObservedRuntimeIdentity(modelFacts, resolved.Model, strings.Repeat("f", 64)); err == nil {
		t.Fatal("changed model digest must not match the rights binding")
	}
}

func TestQwen35NineBUnrestrictedRightsBindCurrentRuntime(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := sharedpolicy.RightsContext{
		ID: "operator-stated-business-context", Version: "2026-11-10",
		RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement",
	}
	resolved, err := p.ResolveRoleWithRights("chat.quality", sharedpolicy.RightsUseLocalInference, ctx, time.Date(2026, 11, 12, 7, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Model != "qwen3.5:9b" || resolved.RightsDecision == nil || resolved.RightsDecision.Status != sharedpolicy.RightsAllowed {
		t.Fatalf("chat.quality rights decision = %+v; want the catalogued unconditional qwen grant", resolved.RightsDecision)
	}
	modelFacts := p.Models[resolved.Model]
	if err := requireObservedRuntimeIdentity(modelFacts, resolved.Model, "6488c96fa5faab64bb65cbd30d4289e20e6130ef535a93ef9a49f42eda893ea7"); err != nil {
		t.Fatalf("exact current model digest was not accepted: %v", err)
	}
	if err := requireObservedRuntimeIdentity(modelFacts, resolved.Model, strings.Repeat("f", 64)); err == nil {
		t.Fatal("changed model digest must not match the rights binding")
	}
}

func TestLoadDefaultRightsContextUsesTheOperatorOwnedGatewayContext(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	context, err := LoadDefaultRightsContext(func(key string) string {
		if key == "VROOLI_SOURCE_ROOT" {
			return root
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if context.ID != "operator-stated-business-context" || context.Version == "" || context.NormalUseLicensing != sharedpolicy.NormalUseLicensingAnyClass || context.NormalUseLicensingEvidence == "" || context.RevenueStatus != "pre_revenue" || context.EntityScope != "operator-group" || context.MonthlyActiveUsers["2023-06"] != "0" || context.MonthlyActiveUsers["2024-06"] != "0" || context.MonthlyActiveUsers["2024-08"] != "0" {
		t.Fatalf("unexpected operator context: %+v", context)
	}
}

func TestLoadDefaultRightsContextFailsClosedOnMissingConfiguredFile(t *testing.T) {
	_, err := LoadDefaultRightsContext(func(key string) string {
		if key == "VROOLI_MODEL_RIGHTS_CONTEXT_PATH" {
			return filepath.Join(t.TempDir(), "missing.json")
		}
		return ""
	})
	if err == nil {
		t.Fatal("missing trusted context was accepted")
	}
}

func TestResolveRoleWithRightsBindsTheResolvedModel(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	ref := p.Roles["chat.small"].Model
	model := p.Models[ref]
	model.Rights = &sharedpolicy.ArtifactRights{
		ArtifactID: "fixture/conditional-model", Revision: "fixture-weights-v1", SHA256: strings.Repeat("a", 64),
		LicenseID: "LicenseRef-fixture", Classification: sharedpolicy.RightsConditional,
		Evidence:    sharedpolicy.RightsEvidence{URL: "https://example.invalid/LICENSE", Revision: "license-v1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-10"},
		Conditions:  []sharedpolicy.RightsCondition{{ID: "revenue-v1", Kind: "annual_revenue", Comparator: "lt", Threshold: 10000000, Unit: "USD", EntityScope: "operator-group", Interpretation: "re-evaluate at threshold"}},
		Obligations: []string{"retain_license_and_notices"},
	}
	p.Models[ref] = model
	ctx := sharedpolicy.RightsContext{ID: "operator-business-context/current", Version: "business-v1", EntityScope: "operator-group", RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "operator_statement"}
	resolved, err := p.ResolveRoleWithRights("chat.small", sharedpolicy.RightsUseLocalInference, ctx, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.RightsDecision == nil || resolved.RightsDecision.Status != sharedpolicy.RightsAllowed || resolved.RightsDecision.ArtifactID != model.Rights.ArtifactID {
		t.Fatalf("decision did not bind resolved artifact: %+v", resolved)
	}
}

func TestValidateRejectsUnknownRoleModel(t *testing.T) {
	p := Policy{
		SchemaVersion: "test",
		Roles: map[string]Role{
			"chat.default": {
				Model:                "missing:latest",
				RequiredCapabilities: []string{"generate"},
				Preference:           1,
				Provenance:           testProvenance("manual_policy"),
			},
		},
		Models: map[string]Model{
			"known:latest": testModel(),
		},
		Constraints: Constraints{
			ProvenanceSourceKinds: []string{"manual_policy", "static_estimate"},
		},
		Provenance: map[string]Provenance{"policy": testProvenance("manual_policy")},
	}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), `roles.chat.default.model "missing:latest" is not in models`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsMissingModalities(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	model := p.Models["qwen3.5:4b"]
	model.Modalities = Modalities{}
	p.Models["qwen3.5:4b"] = model
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "models.qwen3.5:4b.modalities.input must not be empty") {
		t.Fatalf("expected missing modality validation error, got %v", err)
	}
}

func TestValidateRejectsUnknownProvenanceKind(t *testing.T) {
	p := Policy{
		SchemaVersion: "test",
		Roles: map[string]Role{
			"chat.default": {
				Model:                "known:latest",
				RequiredCapabilities: []string{"generate"},
				Preference:           1,
				Provenance:           testProvenance("untracked"),
			},
		},
		Models: map[string]Model{
			"known:latest": testModel(),
		},
		Constraints: Constraints{
			ProvenanceSourceKinds: []string{"manual_policy", "static_estimate"},
		},
		Provenance: map[string]Provenance{"policy": testProvenance("manual_policy")},
	}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), `roles.chat.default.provenance.source_kind "untracked" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRolesAndDirectModels(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	resolution, err := p.Resolve(ResolveRequest{
		ModelRoles: []RoleRequest{{Role: "embedding.default", Reason: "semantic index"}},
		Models: []DirectModelRequest{{
			Name:        "llama3.2",
			Tag:         "3b",
			Reason:      "temporary comparison",
			Owner:       "scenario-team",
			ReviewAfter: "2026-07-10",
		}},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	refs := resolvedRefs(resolution)
	if strings.Join(refs, ",") != "nomic-embed-text:latest,llama3.2:3b" {
		t.Fatalf("refs = %v", refs)
	}
	if len(resolution.Warnings) != 1 || !strings.Contains(resolution.Warnings[0], `direct model "llama3.2:3b"`) {
		t.Fatalf("warnings = %#v", resolution.Warnings)
	}
}

func TestResolveModelReferenceRequiresExactCatalogIdentity(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := p.ResolveModelReference("nomic-embed-text")
	if err != nil || resolved.Model != "nomic-embed-text:latest" {
		t.Fatalf("latest alias resolved as %+v err=%v", resolved, err)
	}
	if _, err := p.ResolveModelReference("not-in-catalog:latest"); err == nil {
		t.Fatal("uncatalogued model identity was accepted")
	}
}

func TestResolveRejectsUnknownRole(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	_, err = p.Resolve(ResolveRequest{ModelRoles: []RoleRequest{{Role: "missing.role"}}})
	if err == nil {
		t.Fatal("expected unknown role error")
	}
	if !strings.Contains(err.Error(), `unknown model role "missing.role"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveWarnsForDirectModelMetadata(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	resolution, err := p.Resolve(ResolveRequest{
		DeprecatedModel: "legacy:latest",
		Models:          []DirectModelRequest{{Name: "untracked:latest"}},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	joined := strings.Join(resolution.Warnings, "\n")
	for _, want := range []string{
		"dependency field model is deprecated",
		`direct model "untracked:latest" is missing exception metadata: reason, owner, review_after`,
		`direct model "untracked:latest" is not in model-policy catalog`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("warnings missing %q:\n%s", want, joined)
		}
	}
}

func resolvedRefs(resolution Resolution) []string {
	out := make([]string, 0, len(resolution.Models))
	for _, model := range resolution.Models {
		out = append(out, model.Ref)
	}
	return out
}

func testModel() Model {
	return Model{
		Family:             "known",
		Capabilities:       []string{"generate"},
		Modalities:         Modalities{Input: []Modality{ModalityText}, Output: []Modality{ModalityText}},
		DiskSizeGBEstimate: 1,
		RAMGBEstimate:      1,
		VRAMGBEstimate:     1,
		Provenance: map[string]Provenance{
			"identity": testProvenance("manual_policy"),
		},
	}
}

func testProvenance(kind string) Provenance {
	return Provenance{
		SourceKind:  kind,
		Confidence:  "medium",
		Source:      "test",
		ObservedAt:  "2026-06-10",
		SampleCount: 0,
	}
}

func fptr(f float64) *float64 { return &f }
func iptr(i int) *int         { return &i }

func TestSamplingResolve_ClampAndPresence(t *testing.T) {
	// Out-of-range values clamp into the safe envelope.
	s := (&SamplingDefaults{Temperature: fptr(5.0), TopP: fptr(1.5), TopK: iptr(-3)}).Resolve()
	if s.Temperature != maxTemperature || s.TopP != maxTopP || s.TopK != minTopK {
		t.Fatalf("clamp failed: %+v", s)
	}
	if !s.HasTemperature || !s.HasTopP || !s.HasTopK {
		t.Fatalf("expected all present, got %+v", s)
	}
	// In-range values pass through.
	s2 := (&SamplingDefaults{Temperature: fptr(0.1)}).Resolve()
	if s2.Temperature != 0.1 || !s2.HasTemperature || s2.HasTopP || s2.HasTopK {
		t.Fatalf("partial sampling wrong: %+v", s2)
	}
}

func TestSamplingResolve_NilIsAbsentFallback(t *testing.T) {
	var s *SamplingDefaults
	got := s.Resolve()
	if got.HasTemperature || got.HasTopP || got.HasTopK {
		t.Fatalf("nil sampling must resolve to all-absent fallback, got %+v", got)
	}
}

func TestResolveRole_CarriesClampedSampling(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := p.ResolveRole("code.local")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Sampling == nil || !resolved.Sampling.HasTemperature || resolved.Sampling.Temperature != 0.1 {
		t.Fatalf("code.local sampling = %+v", resolved.Sampling)
	}
	// A role without sampling_defaults resolves to an all-absent Sampling.
	emb, err := p.ResolveRole("embedding.default")
	if err != nil {
		t.Fatal(err)
	}
	if emb.Sampling != nil && emb.Sampling.HasTemperature {
		t.Fatalf("embedding.default should have no pinned sampling, got %+v", emb.Sampling)
	}
}

func TestValidateRejectsUnknownSamplingSupport(t *testing.T) {
	base := func() Policy {
		return Policy{
			SchemaVersion: "test",
			Roles: map[string]Role{
				"chat.default": {
					Model:                "known:latest",
					RequiredCapabilities: []string{"generate"},
					Preference:           1,
					Provenance:           testProvenance("manual_policy"),
				},
			},
			Models:      map[string]Model{"known:latest": testModel()},
			Constraints: Constraints{ProvenanceSourceKinds: []string{"manual_policy", "static_estimate"}, ModalityVocabulary: []string{"text"}},
			Provenance:  map[string]Provenance{"policy": testProvenance("manual_policy")},
		}
	}

	badState := base()
	role := badState.Roles["chat.default"]
	role.SamplingSupport = map[string]SamplingSupportState{"temperature": "probably"}
	badState.Roles["chat.default"] = role
	if err := badState.Validate(); err == nil || !strings.Contains(err.Error(), `is not an allowed support state`) {
		t.Fatalf("expected support-state rejection, got %v", err)
	}

	// A typo'd key must fail load rather than degrade silently to "undeclared",
	// which would be a wrong declaration wearing a valid-looking policy.
	badKey := base()
	role = badKey.Roles["chat.default"]
	role.SamplingSupport = map[string]SamplingSupportState{"temperatue": SamplingHonored}
	badKey.Roles["chat.default"] = role
	if err := badKey.Validate(); err == nil || !strings.Contains(err.Error(), `is not a known sampling parameter`) {
		t.Fatalf("expected unknown-parameter rejection, got %v", err)
	}

	// A cap below 1 is not a cap.
	badCap := base()
	role = badCap.Roles["chat.default"]
	role.MaxTokens = iptr(0)
	badCap.Roles["chat.default"] = role
	if err := badCap.Validate(); err == nil || !strings.Contains(err.Error(), "max_tokens 0 must be >= 1") {
		t.Fatalf("expected max_tokens rejection, got %v", err)
	}

	// An undeclared role reports unknown, never an error.
	if got := base().Roles["chat.default"].SupportFor("temperature"); got != SamplingUnknown {
		t.Fatalf("undeclared support = %q, want %q", got, SamplingUnknown)
	}
}
