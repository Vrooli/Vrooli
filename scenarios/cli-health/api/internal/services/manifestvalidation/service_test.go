package manifestvalidation

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
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
          "governance": {"effect":"read","run_eligible":true}
        }
      ]
    }
  ]
}`

func newServiceWith(loader ManifestLoader, schema SchemaValidator, protos ProtoLoader) *Service {
	return New(Deps{Manifests: loader, Schema: schema, Protos: protos})
}

func TestValidateScenario_ManifestMissing(t *testing.T) {
	svc := newServiceWith(
		stubLoader{err: os.ErrNotExist},
		stubSchema{},
		stubProto{},
	)
	r, err := svc.ValidateScenario(context.Background(), "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Passed != true {
		t.Fatalf("missing manifest must be warning (passed=true), got passed=%v", r.Passed)
	}
	if len(r.Findings) != 1 || r.Findings[0].Code != CodeManifestMissing || r.Findings[0].Severity != SeverityWarning {
		t.Fatalf("expected one manifest_missing warning, got %+v", r.Findings)
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

func TestValidateScenario_HappyPath(t *testing.T) {
	svc := newServiceWith(
		stubLoader{raw: []byte(validManifest), path: "fixture/cli/manifest.json"},
		stubSchema{},
		stubProto{surface: ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}},
	)
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("happy path should pass; findings=%+v", r.Findings)
	}
	if len(r.Findings) != 0 {
		t.Fatalf("happy path should be empty; got %+v", r.Findings)
	}
}

func TestValidateScenario_RequiresName(t *testing.T) {
	svc := New(Deps{})
	if _, err := svc.ValidateScenario(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
}

func findingHasCode(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func findingMessageContains(findings []Finding, code, needle string) bool {
	for _, f := range findings {
		if f.Code == code && strings.Contains(f.Message, needle) {
			return true
		}
	}
	return false
}
