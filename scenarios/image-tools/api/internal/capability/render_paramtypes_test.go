package capability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeneratedParamTypesAreUpToDate is the drift gate for the typed operation
// parameter views. The Go types and their decoders are what argument builders
// read, so a stale file means a reader can disagree with the schema it claims to
// implement.
//
// Fix by running `make capability-gen` from the scenario root.
func TestGeneratedParamTypesAreUpToDate(t *testing.T) {
	want, err := RenderParamTypes()
	if err != nil {
		t.Fatalf("RenderParamTypes: %v", err)
	}
	path := filepath.Join("..", "opparams", "params.generated.go")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated param types %s: %v (run `make capability-gen`)", path, err)
	}
	if string(got) != want {
		t.Fatalf("generated param types are stale; run `make capability-gen` from the scenario root")
	}
}

// TestGeneratedParamTypesTSAreUpToDate is the same gate for the TypeScript
// mirror the UI builders are typed against.
func TestGeneratedParamTypesTSAreUpToDate(t *testing.T) {
	want, err := RenderParamTypesTS()
	if err != nil {
		t.Fatalf("RenderParamTypesTS: %v", err)
	}
	path := filepath.Join("..", "..", "..", "ui", "src", "features", "workspace", "opParams.generated.ts")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated TS param types %s: %v (run `make capability-gen`)", path, err)
	}
	if string(got) != want {
		t.Fatalf("generated TS param types are stale; run `make capability-gen` from the scenario root")
	}
}

// TestParamTypeCoverage proves the generator's second direction: every operation
// that declares a params_schema got a typed view, and no operation without one
// silently gained a type. A schema key that reaches manifest data but not the
// generated file is exactly the drift this substrate exists to prevent.
func TestParamTypeCoverage(t *testing.T) {
	rendered, err := RenderParamTypes()
	if err != nil {
		t.Fatalf("RenderParamTypes: %v", err)
	}
	for _, m := range All() {
		want := len(m.ParamsSchema) > 0
		got := strings.Contains(rendered, "type "+goTypeName(m.ID)+"Params struct")
		if want != got {
			t.Errorf("operation %q: params_schema=%t but generated type present=%t", m.ID, want, got)
		}
		if !want {
			continue
		}
		doc, err := parseSchemaObject(m.ParamsSchema)
		if err != nil {
			t.Fatalf("operation %q: parse schema: %v", m.ID, err)
		}
		for _, prop := range sortedKeys(doc.properties) {
			if !strings.Contains(rendered, `json:"`+prop) {
				t.Errorf("operation %q: schema property %q has no generated field", m.ID, prop)
			}
		}
	}
}

// TestRenderParamTypesHandlesTopLevelArrayAndRequiredScalar covers the two
// schema shapes no current operation declares but the generator promises to
// support. Without this guard a future operation could generate a decoder that
// unmarshals an array into a single item or assigns a pointer to a required
// value field — both compile, and both fail only at execution.
func TestRenderParamTypesHandlesTopLevelArrayAndRequiredScalar(t *testing.T) {
	doc, err := parseSchemaObject([]byte(`{
		"type": "object",
		"additionalProperties": false,
		"required": ["count"],
		"properties": {
			"count": {"type": "integer"},
			"weights": {"type": "array", "items": {"type": "number"}}
		}
	}`))
	if err != nil {
		t.Fatalf("parse synthetic schema: %v", err)
	}
	decls, err := collectTypes("SyntheticParams", doc)
	if err != nil {
		t.Fatalf("collectTypes: %v", err)
	}
	root := decls[len(decls)-1].declaration
	if !strings.Contains(root, "Count int") {
		t.Errorf("required scalar should be a value field, got:\n%s", root)
	}
	if !strings.Contains(root, "Weights []float64") {
		t.Errorf("array should be a slice field, got:\n%s", root)
	}
	decoder, err := renderDecoder("synthetic", "SyntheticParams", doc)
	if err != nil {
		t.Fatalf("renderDecoder: %v", err)
	}
	if !strings.Contains(decoder, "var v []float64") {
		t.Errorf("array decoder should unmarshal into a slice, got:\n%s", decoder)
	}
	if !strings.Contains(decoder, "out.Count = v") {
		t.Errorf("required scalar decoder should assign the value, got:\n%s", decoder)
	}
}
