package navigation_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	"flow-verifier/internal/flows/schemas"
)

func TestKindRegistered(t *testing.T) {
	k, ok := kind.Get(navigation.Name)
	if !ok {
		t.Fatalf("navigation kind not registered")
	}
	if k.Name() != "navigation" {
		t.Fatalf("Name() = %q, want navigation", k.Name())
	}
}

func TestLoadMinimalExample(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationMinimalExample, "schemas/examples/navigation-minimal.json")
	if err != nil {
		t.Fatalf("load minimal example: %v", err)
	}
	if spec.FlowID() != "example.minimal.ui" {
		t.Fatalf("FlowID = %q, want example.minimal.ui", spec.FlowID())
	}
	if spec.Kind() != "navigation" {
		t.Fatalf("Kind = %q, want navigation", spec.Kind())
	}
	if spec.SchemaVersion() != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", spec.SchemaVersion())
	}
}

func TestLoadFullExample(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load full example: %v", err)
	}
	if spec.FlowID() != "reference-react-vite.app.ui" {
		t.Fatalf("FlowID = %q", spec.FlowID())
	}
	nav, ok := spec.(*navigation.Spec)
	if !ok {
		t.Fatalf("spec is not *navigation.Spec")
	}
	g := nav.Graph()
	if len(g.Contract.Routes) != 11 {
		t.Fatalf("Routes count = %d, want 11", len(g.Contract.Routes))
	}
	if len(g.Contract.ReachabilityInvariants) != 8 {
		t.Fatalf("ReachabilityInvariants count = %d, want 8", len(g.Contract.ReachabilityInvariants))
	}
	if len(g.Contract.DeepLinkPolicy) != 4 {
		t.Fatalf("DeepLinkPolicy count = %d, want 4", len(g.Contract.DeepLinkPolicy))
	}
}

func TestRoundTripFullExample(t *testing.T) {
	var in, out any
	if err := json.Unmarshal(schemas.NavigationFullExample, &in); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	nav := spec.(*navigation.Spec)
	emitted, err := json.Marshal(nav.Graph().Contract)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(emitted, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	// $schema is dropped on emit (it carries no semantic load and the
	// struct's omitempty on the empty default would drop it anyway when
	// not set). Drop it from the input side too before comparing.
	canon := func(v any) {
		m, ok := v.(map[string]any)
		if !ok {
			return
		}
		delete(m, "$schema")
		if rs, ok := m["routes"].([]any); ok {
			for _, r := range rs {
				if rm, ok := r.(map[string]any); ok {
					if ps, ok := rm["parents"].([]any); ok && len(ps) == 0 {
						delete(rm, "parents")
					}
				}
			}
		}
	}
	canon(in)
	canon(out)
	inJSON, _ := json.Marshal(in)
	outJSON, _ := json.Marshal(out)
	if string(inJSON) != string(outJSON) {
		t.Fatalf("round-trip differs.\nin:  %s\nout: %s", inJSON, outJSON)
	}
}

func TestInvalidAffordanceToRejected(t *testing.T) {
	// Schema-valid but structurally invalid: affordance.to points at a
	// route id that does not exist.
	raw := []byte(`{
"$schema":"https://vrooli.dev/schemas/navigation.v1.json",
"schemaVersion":1,"kind":"navigation",
"flowId":"x.y.z","domain":"x","description":"d",
"contexts":{"viewport":{"kind":"enum","values":["mobile","desktop"],"default":"desktop"}},
"routes":[{"id":"home","path":"/","page":"p","parents":[]}],
"containers":[{"id":"c","kind":"persistent","host_routes":["*"],"disclosure":"always_visible"}],
"affordances":[{"id":"a","to":"nope","presentations":[{"in":"c","label":"L","test_id":"t","reachable_via":["mouse"]}]}]
}`)
	k, _ := kind.Get(navigation.Name)
	_, err := k.Load(raw, "test.json")
	if err == nil {
		t.Fatalf("expected structural error, got nil")
	}
	if !strings.Contains(err.Error(), `affordance "a": to "nope"`) {
		t.Fatalf("error message missing affordance.to diagnosis: %v", err)
	}
}

func TestSchemaInvalidRejected(t *testing.T) {
	// Schema-invalid: missing required "kind" field.
	raw := []byte(`{
"schemaVersion":1,"flowId":"x.y.z","domain":"x","description":"d",
"contexts":{"v":{"kind":"bool","default":false}},
"routes":[{"id":"home","path":"/","page":"p"}],
"containers":[],
"affordances":[]
}`)
	k, _ := kind.Get(navigation.Name)
	_, err := k.Load(raw, "test.json")
	if err == nil {
		t.Fatalf("expected schema error, got nil")
	}
}

func TestVerifyFullExamplePasses(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res, err := k.Verify(context.Background(), spec)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !res.Passed {
		var msgs []string
		for _, f := range res.Findings {
			if !f.Passed {
				msgs = append(msgs, f.Message)
			}
		}
		t.Fatalf("verify expected to pass, %d failures:\n%s", len(msgs), strings.Join(msgs, "\n"))
	}
	if len(res.Findings) < 8 {
		t.Fatalf("expected at least 8 invariant findings, got %d", len(res.Findings))
	}
}

func TestVerifyDetectsUnreachable(t *testing.T) {
	// Tamper with the full example: drop the nav_settings affordance so
	// settings_index becomes unreachable. The settings_reachable_quickly
	// invariant must fail.
	var doc map[string]any
	if err := json.Unmarshal(schemas.NavigationFullExample, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	affs := doc["affordances"].([]any)
	kept := affs[:0]
	for _, a := range affs {
		am := a.(map[string]any)
		if am["id"] != "nav_settings" && am["id"] != "nav_home" {
			kept = append(kept, a)
		}
	}
	doc["affordances"] = kept
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(raw, "tampered.json")
	if err != nil {
		t.Fatalf("load tampered: %v", err)
	}
	res, err := k.Verify(context.Background(), spec)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if res.Passed {
		t.Fatalf("expected verify to fail")
	}
	found := false
	for _, f := range res.Findings {
		if f.ID == "settings_reachable_quickly" && !f.Passed {
			found = true
			if !strings.Contains(f.Message, "not reachable") && !strings.Contains(f.Message, "settings_index") {
				t.Errorf("expected unreachable message, got: %s", f.Message)
			}
		}
	}
	if !found {
		t.Fatalf("expected settings_reachable_quickly failure in findings")
	}
}
