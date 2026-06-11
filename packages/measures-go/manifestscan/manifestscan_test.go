package manifestscan

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	measures "github.com/vrooli/measures-go"
)

// stubSource is a SchemaSource returning fixed proto params (no descriptor).
type stubSource struct {
	params []measures.ParamSchema
	err    error
}

func (s stubSource) RequestParams(_, _ string) ([]measures.ParamSchema, error) {
	return s.params, s.err
}

const measureManifest = `{
  "name": "fixture",
  "groups": [
    {
      "name": "backlog",
      "commands": [
        {
          "name": "completed",
          "binding": {"kind":"connect-rpc","service":"StatsService","method":"BacklogCompletionCount"},
          "governance": {"effect":"read","run_eligible":true},
          "measure": {
            "intent": "How many backlog items were completed in a time window.",
            "questions": ["how many backlog items did we complete this week", "backlog items closed last month"],
            "params": {
              "window": {"type":"time_window","default":"this_week"},
              "initiative": {"type":"enum","values_source":"initiative_names"}
            },
            "result": {"kind":"scalar","value_field":"count","unit":"items","summary_template":"{count} items ({window})"}
          }
        },
        {
          "name": "noop",
          "binding": {"kind":"connect-rpc","service":"StatsService","method":"Other"},
          "governance": {"effect":"read","run_eligible":true}
        }
      ]
    }
  ],
  "measures": {
    "omitted": [{"domain":"queue","reason":"ephemeral; no historical value"}],
    "domains": [{"domain":"settings","stateful":false,"reason":"config only"}]
  }
}`

func TestParse_ExtractsMeasuresAndMeta(t *testing.T) {
	parsed, err := Parse([]byte(measureManifest))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(parsed.Commands) != 1 {
		t.Fatalf("expected 1 command with a measure, got %d", len(parsed.Commands))
	}
	cm := parsed.Commands[0]
	if cm.Group != "backlog" || cm.Command != "completed" {
		t.Fatalf("unexpected command identity: %+v", cm)
	}
	if cm.Domain != "backlog" { // defaulted from group
		t.Fatalf("domain should default to group name, got %q", cm.Domain)
	}
	if cm.MeasureName() != "backlog.completed" {
		t.Fatalf("measure name = %q", cm.MeasureName())
	}
	if cm.Binding.Service != "StatsService" || cm.Binding.Method != "BacklogCompletionCount" {
		t.Fatalf("binding not parsed: %+v", cm.Binding)
	}
	if cm.Governance.Effect != measures.EffectRead || !cm.Governance.RunEligible {
		t.Fatalf("governance not parsed: %+v", cm.Governance)
	}
	if len(cm.Measure.Questions) != 2 || cm.Measure.Result.ValueField != "count" {
		t.Fatalf("measure body not parsed: %+v", cm.Measure)
	}
	if len(parsed.Omitted) != 1 || parsed.Omitted[0].Domain != "queue" {
		t.Fatalf("omitted not parsed: %+v", parsed.Omitted)
	}
	if len(parsed.Domains) != 1 || parsed.Domains[0].Stateful {
		t.Fatalf("domain override not parsed: %+v", parsed.Domains)
	}
}

func TestParse_DomainOverride(t *testing.T) {
	const m = `{"name":"f","groups":[{"name":"g","commands":[
      {"name":"c","binding":{"kind":"connect-rpc","service":"S","method":"M"},"governance":{"effect":"read","run_eligible":true},
       "measure":{"domain":"records","intent":"i","questions":["q"],"result":{"kind":"scalar","value_field":"v"}}}]}]}`
	parsed, err := Parse([]byte(m))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Commands[0].Domain != "records" {
		t.Fatalf("explicit domain override should win, got %q", parsed.Commands[0].Domain)
	}
}

func TestParse_NoMeasures(t *testing.T) {
	const m = `{"name":"f","groups":[{"name":"g","commands":[
      {"name":"c","binding":{"kind":"connect-rpc","service":"S","method":"M"},"governance":{"effect":"read","run_eligible":true}}]}]}`
	parsed, err := Parse([]byte(m))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Commands) != 0 {
		t.Fatalf("expected no measure commands, got %d", len(parsed.Commands))
	}
}

func TestKnownManifestParamType(t *testing.T) {
	for _, ok := range []string{"", "time_window", "enum"} {
		if !KnownManifestParamType(ok) {
			t.Errorf("%q should be known", ok)
		}
	}
	for _, bad := range []string{"int32", "string", "bogus", "datetime"} {
		if KnownManifestParamType(bad) {
			t.Errorf("%q should be unknown", bad)
		}
	}
}

func TestManifestParamTypes(t *testing.T) {
	parsed, err := Parse([]byte(measureManifest))
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Commands[0].ManifestParamTypes()
	if got["window"] != "time_window" || got["initiative"] != "enum" {
		t.Fatalf("ManifestParamTypes = %+v", got)
	}
}

func declWith(params map[string]measures.Param) measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name: "d.m", Domain: "d", Questions: []string{"q"},
		Params: params, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "v"},
		Effect: measures.EffectRead,
	}
}

func TestGradeTier(t *testing.T) {
	tw := measures.Param{Name: "window", Type: measures.ParamTypeTimeWindow}
	enum := measures.Param{Name: "init", Type: measures.ParamTypeEnum, EnumValues: []string{"a"}}
	bare := measures.Param{Name: "free", Type: "string"}

	cases := []struct {
		name   string
		params map[string]measures.Param
		want   Tier
	}{
		{"empty-is-full", nil, TierFull},
		{"all-canonical-or-constrained", map[string]measures.Param{"window": tw, "init": enum}, TierFull},
		{"mixed-is-partial", map[string]measures.Param{"window": tw, "free": bare}, TierPartial},
		{"all-bare-is-fallback", map[string]measures.Param{"free": bare}, TierFallback},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GradeTier(declWith(c.params)); got != c.want {
				t.Fatalf("GradeTier = %q, want %q", got, c.want)
			}
		})
	}
}

func TestAssemble_StubSource_FullTier(t *testing.T) {
	parsed, err := Parse([]byte(measureManifest))
	if err != nil {
		t.Fatal(err)
	}
	src := stubSource{params: []measures.ParamSchema{
		{Name: "window", Type: measures.ParamTypeTimeWindow, MessageType: measures.TimeWindowMessageName},
		{Name: "initiative", Type: "string"},
	}}
	decl, err := parsed.Commands[0].Assemble(src)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if decl.Name != "backlog.completed" {
		t.Fatalf("decl name = %q", decl.Name)
	}
	// window -> time_window (canonical); initiative -> dynamic enum via values_source.
	if !decl.Params["window"].IsCanonical() {
		t.Fatalf("window should be canonical: %+v", decl.Params["window"])
	}
	if decl.Params["initiative"].Type != measures.ParamTypeEnum {
		t.Fatalf("initiative should upgrade to enum: %+v", decl.Params["initiative"])
	}
	if got := GradeTier(decl); got != TierFull {
		t.Fatalf("tier = %q, want full", got)
	}
}

func TestAssemble_Drift(t *testing.T) {
	// Manifest names params window+initiative, but proto only has window ->
	// initiative is drift.
	parsed, err := Parse([]byte(measureManifest))
	if err != nil {
		t.Fatal(err)
	}
	src := stubSource{params: []measures.ParamSchema{
		{Name: "window", Type: measures.ParamTypeTimeWindow, MessageType: measures.TimeWindowMessageName},
	}}
	if _, err := parsed.Commands[0].Assemble(src); err == nil {
		t.Fatalf("expected drift error for manifest param with no proto field")
	}
}

// findRepoRoot walks up from dir until it finds the committed proto descriptor
// image, returning the repo root. Avoids a repo-contract-go dependency in the
// shared module (this test is the only consumer that needs the root).
func findRepoRoot(dir string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "packages", "proto", "gen", "descriptor", "image.binpb")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// TestDescriptorSchemaReader_RealImage proves the committed descriptor image
// resolves a real request message end-to-end. Skips when the image is absent.
func TestDescriptorSchemaReader_RealImage(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root, ok := findRepoRoot(filepath.Dir(file))
	if !ok {
		t.Skip("repo root with descriptor image not found")
	}
	r := NewDescriptorSchemaReader(root)
	params, err := r.RequestParams("ValidationService", "ValidateScenario")
	if err != nil {
		t.Skipf("descriptor image unavailable: %v", err)
	}
	var foundScenario bool
	for _, p := range params {
		if p.Name == "scenario" && p.Type == "string" {
			foundScenario = true
		}
	}
	if !foundScenario {
		t.Fatalf("expected a string param 'scenario', got %+v", params)
	}
}
