package manifestvalidation

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// stubLoader / stubSchema / stubProto are minimal seams for unit tests.

type stubLoader struct {
	raw  []byte
	path string
	err  error
}

func (s stubLoader) Load(_ context.Context, _ string) ([]byte, string, error) {
	return s.raw, s.path, s.err
}

type stubSchema struct {
	findings []Finding
	err      error
}

func (s stubSchema) Validate(_ context.Context, _ []byte) ([]Finding, error) {
	return s.findings, s.err
}

type stubProto struct {
	surface ProtoSurface
	err     error
}

func (s stubProto) Load(_ context.Context, _ string) (ProtoSurface, error) {
	return s.surface, s.err
}

// validManifest is a structurally and contract-valid manifest bound to the
// stub proto surface used by most tests.
const validManifest = `{
  "name": "fixture",
  "groups": [
    {
      "name": "g1",
      "commands": [
        {
          "name": "do",
          "binding": {"kind":"connect-rpc","service":"Svc","method":"Do"},
          "governance": {"effect":"read","run_eligible":true},
          "architecture": {"primitive":"proto_list"}
        }
      ]
    }
  ]
}`

func newServiceWith(loader ManifestLoader, schema SchemaValidator, protos ProtoLoader) *Service {
	return New(Deps{Manifests: loader, Schema: schema, Protos: protos})
}

// A genuinely proto-less scenario (no own proto surface) keeps the soft
// skip-with-warning when it has no manifest.
func TestValidateScenario_ManifestMissing_ProtoLess(t *testing.T) {
	svc := newServiceWith(
		stubLoader{err: os.ErrNotExist},
		stubSchema{},
		stubProto{}, // empty surface
	)
	r, err := svc.ValidateScenario(context.Background(), "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Passed != true {
		t.Fatalf("missing manifest for a proto-less scenario must be warning (passed=true), got passed=%v", r.Passed)
	}
	if len(r.Findings) != 1 || r.Findings[0].Code != CodeManifestMissing || r.Findings[0].Severity != SeverityWarning {
		t.Fatalf("expected one manifest_missing warning, got %+v", r.Findings)
	}
}

// The loophole fix: a scenario that exposes its own proto RPC surface but has no
// cli/manifest.json is a hard ERROR (passed=false), not a skip — the manifest is
// the mandatory single source of truth for its CLI.
func TestValidateScenario_ManifestMissing_ProtoBearingIsError(t *testing.T) {
	svc := newServiceWith(
		stubLoader{err: os.ErrNotExist},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, err := svc.ValidateScenario(context.Background(), "proto-bearing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Passed {
		t.Fatalf("missing manifest for a proto-bearing scenario must fail (passed=false), got passed=true")
	}
	// A proto-bearing scenario without a manifest yields the manifest.required
	// error plus an arch.unclassifiable warning: there is no manifest to classify
	// command architecture from, so the capability sits at L0 rather than falsely
	// reporting top maturity by absence of findings.
	if !findingHasCode(r.Findings, CodeManifestRequired) {
		t.Fatalf("expected manifest_required error, got %+v", r.Findings)
	}
	req := findingByCode(r.Findings, CodeManifestRequired)
	if req == nil || req.Severity != SeverityError {
		t.Fatalf("expected manifest_required to be an error, got %+v", req)
	}
	unclass := findingByCode(r.Findings, CodeArchUnclassifiable)
	if unclass == nil || unclass.Severity != SeverityWarning {
		t.Fatalf("expected arch.unclassifiable warning, got %+v", r.Findings)
	}
}

func TestValidateTarget_ProjectMissingManifestIsError(t *testing.T) {
	svc := newServiceWith(
		stubLoader{err: os.ErrNotExist},
		stubSchema{},
		stubProto{},
	)
	report, err := svc.ValidateTarget(context.Background(), Target{Kind: TargetKindProject, ID: ProjectTargetID})
	if err != nil {
		t.Fatalf("ValidateTarget: %v", err)
	}
	if report.Passed || findingByCode(report.Findings, CodeManifestRequired) == nil {
		t.Fatalf("missing project manifest must be an error: %+v", report)
	}
	if findingByCode(report.Findings, CodeManifestMissing) != nil {
		t.Fatalf("project target must not use the scenario warning: %+v", report.Findings)
	}
}

func TestValidateTarget_ProjectUsesProjectPipeline(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest), path: "cli/manifest.json"},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	report, err := svc.ValidateTarget(context.Background(), Target{Kind: TargetKindProject, ID: ProjectTargetID})
	if err != nil {
		t.Fatalf("ValidateTarget: %v", err)
	}
	if report.Scenario != ProjectTargetID || findingByCode(report.Findings, CodeProtoOrphanMethod) != nil {
		t.Fatalf("project target did not use the project pipeline: %+v", report)
	}
}

func TestValidateScenario_ManifestParseError(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte("not json")},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if r.Passed {
		t.Fatalf("invalid JSON should fail")
	}
	if !findingHasCode(r.Findings, CodeManifestParseError) {
		t.Fatalf("expected manifest_parse_error, got %+v", r.Findings)
	}
}

func TestValidateScenario_SchemaFindingsPropagate(t *testing.T) {
	schema := stubSchema{findings: []Finding{{
		Severity: SeverityError, Code: CodeManifestSchemaError, Location: "x", Message: "bad",
	}}}
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest)},
		schema,
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if r.Passed {
		t.Fatalf("schema error should fail")
	}
	if !findingHasCode(r.Findings, CodeManifestSchemaError) {
		t.Fatalf("expected schema error finding, got %+v", r.Findings)
	}
}

func TestValidateScenario_ProtoBuildFailed(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest)},
		stubSchema{},
		stubProto{err: errors.New("buf exploded")},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if r.Passed {
		t.Fatalf("proto build failure should fail")
	}
	if !findingHasCode(r.Findings, CodeProtoBuildFailed) {
		t.Fatalf("expected proto_build_failed, got %+v", r.Findings)
	}
}

func TestValidateScenario_BindingUnknownService(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Other", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !findingHasCode(r.Findings, CodeBindingUnknownSvc) {
		t.Fatalf("expected binding_unknown_service, got %+v", r.Findings)
	}
}

func TestValidateScenario_BindingUnknownMethod(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Other"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !findingHasCode(r.Findings, CodeBindingUnknownMethod) {
		t.Fatalf("expected binding_unknown_method, got %+v", r.Findings)
	}
}

func TestValidateScenario_OrphanMethod(t *testing.T) {
	// Manifest binds Svc.Do, but proto declares Svc.Do AND Svc.Extra.
	// Svc.Extra is neither bound nor omitted -> orphan.
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do", "Extra"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !findingHasCode(r.Findings, CodeProtoOrphanMethod) {
		t.Fatalf("expected proto_orphan_method, got %+v", r.Findings)
	}
	if !findingMessageContains(r.Findings, CodeProtoOrphanMethod, "Svc.Extra") {
		t.Fatalf("orphan finding should name Svc.Extra: %+v", r.Findings)
	}
}

func TestValidateScenario_SharedBindingDoesNotRequireFullCoverage(t *testing.T) {
	manifest := `{
      "name": "fixture",
      "groups": [{"name":"validate","commands":[{"name":"scenario","binding":{"kind":"connect-rpc","service":"ScenarioValidationService","method":"ValidateScenario"},"governance":{"effect":"read","run_eligible":true}}]}]
    }`
	svc := newServiceWith(
		stubLoader{raw: []byte(manifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{
			Shared: []ProtoService{{Name: "ScenarioValidationService", Methods: []string{"ValidateScenario", "FutureSharedMethod"}}},
		}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("shared binding should pass without orphan coverage requirements; findings=%+v", r.Findings)
	}
}

func TestValidateScenario_OmissionCoversOrphan(t *testing.T) {
	manifest := `{
      "name": "fixture",
      "groups": [{"name":"g1","commands":[{"name":"do","binding":{"kind":"connect-rpc","service":"Svc","method":"Do"},"governance":{"effect":"read","run_eligible":true}}]}],
      "omitted": [{"service":"Svc","method":"Extra","reason":"internal-only"}]
    }`
	svc := newServiceWith(
		stubLoader{raw: []byte(manifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do", "Extra"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("omitted entry should cover Svc.Extra; findings=%+v", r.Findings)
	}
}

func TestValidateScenario_OrphanOmission(t *testing.T) {
	manifest := `{
      "name": "fixture",
      "groups": [{"name":"g1","commands":[{"name":"do","binding":{"kind":"connect-rpc","service":"Svc","method":"Do"},"governance":{"effect":"read","run_eligible":true}}]}],
      "omitted": [{"service":"Svc","method":"Ghost","reason":"stale"}]
    }`
	svc := newServiceWith(
		stubLoader{raw: []byte(manifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !findingHasCode(r.Findings, CodeOmissionOrphan) {
		t.Fatalf("expected omission_orphan, got %+v", r.Findings)
	}
	if !r.Passed {
		t.Fatalf("stale omission alone is warning, must not fail")
	}
}

func TestValidateScenario_DuplicateBinding(t *testing.T) {
	manifest := `{
      "name": "fixture",
      "groups": [
        {"name":"g1","commands":[
          {"name":"a","binding":{"kind":"connect-rpc","service":"Svc","method":"Do"},"governance":{"effect":"read","run_eligible":true}},
          {"name":"b","binding":{"kind":"connect-rpc","service":"Svc","method":"Do"},"governance":{"effect":"read","run_eligible":true}}
        ]}
      ]
    }`
	svc := newServiceWith(
		stubLoader{raw: []byte(manifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !findingHasCode(r.Findings, CodeBindingDuplicate) {
		t.Fatalf("expected binding_duplicate, got %+v", r.Findings)
	}
}

func TestValidateScenario_LocalCommandsAreNotProtoBindings(t *testing.T) {
	manifest := `{
      "name": "fixture",
      "groups": [{"name":"flat","flat":true,"commands":[
        {"name":"status","binding":{"kind":"local"},"governance":{"effect":"read","run_eligible":true}},
        {"name":"playbooks","binding":{"kind":"local"},"governance":{"effect":"read","run_eligible":true}}
      ]}]
    }`
	svc := newServiceWith(
		stubLoader{raw: []byte(manifest)},
		stubSchema{},
		stubProto{surface: ProtoSurface{}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if findingHasCode(r.Findings, CodeBindingUnknownSvc) || findingHasCode(r.Findings, CodeBindingDuplicate) {
		t.Fatalf("local commands must not be validated as proto bindings: %+v", r.Findings)
	}
}

func TestValidateScenario_LocalOnlyManifestDoesNotRequireProtoSchema(t *testing.T) {
	manifest := `{
      "name": "fixture",
      "groups": [{"name":"local","commands":[
        {"name":"status","binding":{"kind":"local"},"governance":{"effect":"read","run_eligible":true}}
      ]}]
    }`
	svc := newServiceWith(
		stubLoader{raw: []byte(manifest), path: "cli/manifest.json"},
		stubSchema{},
		stubProto{err: errors.New("proto schema should not be loaded")},
	)
	r, err := svc.ValidateScenario(context.Background(), "local-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if findingHasCode(r.Findings, CodeProtoBuildFailed) {
		t.Fatalf("local-only manifest must not require a proto schema: %+v", r.Findings)
	}
}

// stubArchEvidence is a fake ArchitectureEvidenceProvider returning canned
// cli-core primitive evidence, so service tests can prove verified-vs-unverified
// classification through the real ValidateScenario flow.
type stubArchEvidence struct {
	ev  ArchitectureEvidence
	err error
}

func (s stubArchEvidence) Evidence(context.Context, string) (ArchitectureEvidence, error) {
	return s.ev, s.err
}

func TestValidateScenario_HappyPath(t *testing.T) {
	// A fully-verified scenario: the declared proto_list primitive is proven by
	// matching cli-core evidence, so there is no not-yet-verified debt.
	svc := New(Deps{
		Manifests: stubLoader{raw: []byte(validManifest), path: "fixture/cli/manifest.json"},
		Schema:    stubSchema{},
		Protos:    stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
		ArchitectureEvidence: stubArchEvidence{ev: ArchitectureEvidence{Primitives: map[string]cliapp.PrimitiveClass{
			"g1 do": cliapp.PrimitiveProtoList,
		}}},
	})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("happy path should pass; findings=%+v", r.Findings)
	}
	if len(r.Findings) != 0 {
		t.Fatalf("happy path should be empty; got %+v", r.Findings)
	}
}

// Without an evidence provider, a declared primitive is not-yet-verified debt:
// an advisory warning that does not fail the phase (Passed stays true).
func TestValidateScenario_DeclaredPrimitiveUnverifiedWithoutEvidence(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest), path: "fixture/cli/manifest.json"},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("unverified declared primitive is advisory debt, must not fail the phase; findings=%+v", r.Findings)
	}
	f := findingByCode(r.Findings, CodeArchPrimitiveUnverif)
	if f == nil || f.Severity != SeverityWarning {
		t.Fatalf("expected an advisory primitive_unverified warning, got %+v", r.Findings)
	}
}

// A provider whose observed primitive contradicts the declaration is a gating
// error (Passed=false) surfaced through the full validation flow.
func TestValidateScenario_PrimitiveMismatchIsGating(t *testing.T) {
	svc := New(Deps{
		Manifests: stubLoader{raw: []byte(validManifest), path: "fixture/cli/manifest.json"},
		Schema:    stubSchema{},
		Protos:    stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
		ArchitectureEvidence: stubArchEvidence{ev: ArchitectureEvidence{Primitives: map[string]cliapp.PrimitiveClass{
			"g1 do": cliapp.PrimitiveProtoMutation,
		}}},
	})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if r.Passed {
		t.Fatalf("primitive mismatch must fail the phase, got passed=true; findings=%+v", r.Findings)
	}
	if findingByCode(r.Findings, CodeArchPrimitiveMismatch) == nil {
		t.Fatalf("expected primitive_mismatch, got %+v", r.Findings)
	}
}

func TestValidateScenario_RequiresName(t *testing.T) {
	svc := New(Deps{})
	if _, err := svc.ValidateScenario(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
}

func findingHasCode(findings []Finding, code string) bool {
	return findingByCode(findings, code) != nil
}

func findingByCode(findings []Finding, code string) *Finding {
	for i := range findings {
		if findings[i].Code == code {
			return &findings[i]
		}
	}
	return nil
}

func findingMessageContains(findings []Finding, code, needle string) bool {
	for _, f := range findings {
		if f.Code == code && strings.Contains(f.Message, needle) {
			return true
		}
	}
	return false
}
