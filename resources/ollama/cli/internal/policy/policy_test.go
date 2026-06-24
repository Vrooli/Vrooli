package policy

import (
	"path/filepath"
	"strings"
	"testing"
)

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
