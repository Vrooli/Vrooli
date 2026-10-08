package manifestvalidation

import (
	"context"
	"testing"

	measures "github.com/vrooli/measures-go"
)

type stubMeasureSchema struct {
	params []measures.ParamSchema
	err    error
}

func (s stubMeasureSchema) RequestParams(_, _ string) ([]measures.ParamSchema, error) {
	return s.params, s.err
}

// measureManifest builds a manifest whose one command (g1/do -> Svc.Do) carries
// the given measure block JSON (a fragment like `"window":{"type":"time_window"}`).
func measureManifest(measureBlock string) []byte {
	return []byte(`{
      "name": "fixture",
      "groups": [{"name":"g1","commands":[{
        "name":"do",
        "binding":{"kind":"connect-rpc","service":"Svc","method":"Do"},
        "governance":{"effect":"read","run_eligible":true},
        "measure":` + measureBlock + `
      }]}]
    }`)
}

func windowProtoParams() []measures.ParamSchema {
	return []measures.ParamSchema{
		{Name: "window", Type: measures.ParamTypeTimeWindow, MessageType: measures.TimeWindowMessageName},
	}
}

func svcDoSurface() ProtoSurface {
	return ProtoSurface{Services: []ProtoService{{Name: "Svc", Methods: []string{"Do"}}}}
}

func newServiceWithMeasures(raw []byte, surface ProtoSurface, m MeasureSchemaReader) *Service {
	return New(Deps{
		Manifests: stubLoader{raw: raw, path: "fixture/cli/manifest.json"},
		Schema:    stubSchema{},
		Protos:    stubProto{surface: surface},
		Measures:  m,
	})
}

const goodMeasureBlock = `{
  "intent":"How many backlog items were completed in a time window.",
  "questions":["how many backlog items did we complete this week"],
  "params":{"window":{"type":"time_window","default":"this_week"}},
  "result":{"kind":"scalar","value_field":"count","unit":"items"}
}`

func TestMeasureCheck_TierFull(t *testing.T) {
	svc := newServiceWithMeasures(measureManifest(goodMeasureBlock), svcDoSurface(),
		stubMeasureSchema{params: windowProtoParams()})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("well-formed measure should pass; findings=%+v", r.Findings)
	}
	if !findingHasCode(r.Findings, CodeMeasureTier) {
		t.Fatalf("expected a measure.tier info finding, got %+v", r.Findings)
	}
	if !findingMessageContains(r.Findings, CodeMeasureTier, "tier=full") {
		t.Fatalf("expected tier=full, got %+v", r.Findings)
	}
}

func TestMeasureCheck_Drift(t *testing.T) {
	// Measure declares a param "ghost" that has no matching proto request field.
	block := `{"intent":"i","questions":["q"],"params":{"ghost":{"default":"x"}},"result":{"kind":"scalar","value_field":"count"}}`
	svc := newServiceWithMeasures(measureManifest(block), svcDoSurface(),
		stubMeasureSchema{params: windowProtoParams()})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if r.Passed {
		t.Fatalf("measure drift should fail")
	}
	if !findingHasCode(r.Findings, CodeMeasureInvalid) {
		t.Fatalf("expected measure.invalid, got %+v", r.Findings)
	}
}

func TestMeasureCheck_UnknownParamType(t *testing.T) {
	block := `{"intent":"i","questions":["q"],"params":{"window":{"type":"bogus"}},"result":{"kind":"scalar","value_field":"count"}}`
	svc := newServiceWithMeasures(measureManifest(block), svcDoSurface(),
		stubMeasureSchema{params: windowProtoParams()})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if r.Passed {
		t.Fatalf("unknown param type should fail")
	}
	if !findingHasCode(r.Findings, CodeMeasureUnknownType) {
		t.Fatalf("expected measure.unknown_param_type, got %+v", r.Findings)
	}
}

func TestMeasureCheck_SchemaUnread(t *testing.T) {
	svc := newServiceWithMeasures(measureManifest(goodMeasureBlock), svcDoSurface(),
		stubMeasureSchema{err: context.DeadlineExceeded})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	if !r.Passed {
		t.Fatalf("schema-unread is a warning, must not fail; findings=%+v", r.Findings)
	}
	if !findingHasCode(r.Findings, CodeMeasureSchemaUnread) {
		t.Fatalf("expected measure.schema_unread warning, got %+v", r.Findings)
	}
}

func TestMeasureCheck_NilSeamSkips(t *testing.T) {
	// No Measures seam -> measure validation is skipped entirely.
	svc := New(Deps{
		Manifests: stubLoader{raw: measureManifest(goodMeasureBlock)},
		Schema:    stubSchema{},
		Protos:    stubProto{surface: svcDoSurface()},
	})
	r, _ := svc.ValidateScenario(context.Background(), "s")
	for _, f := range r.Findings {
		switch f.Code {
		case CodeMeasureTier, CodeMeasureInvalid, CodeMeasureUnknownType, CodeMeasureSchemaUnread:
			t.Fatalf("nil measure seam should emit no measure findings, got %+v", f)
		}
	}
}

// --- schema accept/reject (against the real cli-manifest.schema.json) ---

func TestSchema_AcceptsMeasureBlock(t *testing.T) {
	root := findRepoRoot(t)
	v := NewJSONSchemaValidator(root)
	findings, err := v.Validate(context.Background(), measureManifest(goodMeasureBlock))
	if err != nil {
		t.Fatalf("validator setup error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("a well-formed measure block should validate; findings=%+v", findings)
	}
}

func TestSchema_RejectsMeasureMissingValueField(t *testing.T) {
	root := findRepoRoot(t)
	v := NewJSONSchemaValidator(root)
	// result has no value_field (required).
	block := `{"intent":"i","questions":["q"],"result":{"kind":"scalar"}}`
	findings, err := v.Validate(context.Background(), measureManifest(block))
	if err != nil {
		t.Fatalf("validator setup error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatalf("a measure result missing value_field must be rejected")
	}
}
