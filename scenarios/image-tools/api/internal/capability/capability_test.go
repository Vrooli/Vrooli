package capability

import (
	"testing"

	"image-tools/internal/ops"
)

func TestLoadValidatesEmbeddedSeed(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.SchemaVersion() == "" {
		t.Error("schema version should be set")
	}
	if got := len(r.All()); got == 0 {
		t.Fatal("manifest set should not be empty")
	}
}

// TestEnginesAreDeclared keeps the executor declaration honest: every manifest
// names an engine, and the compute engine is the only one whose operations have
// no model backend.
func TestEnginesAreDeclared(t *testing.T) {
	for _, m := range All() {
		if !m.Engine.valid() {
			t.Errorf("manifest %q has invalid engine %q", m.ID, m.Engine)
		}
		if m.Engine == EngineCompute && len(m.Backends) != 0 {
			t.Errorf("compute manifest %q declares backends %v", m.ID, m.Backends)
		}
		if m.Engine.Runs() == false && m.Engine != EngineUnwired {
			t.Errorf("manifest %q engine %q has no runner and is not declared unwired", m.ID, m.Engine)
		}
	}
}

// TestSeedOrderMatchesRunnerOrder checks the compute engine's manifests keep the
// same order as the in-process table, so discovery output and generated files
// diff cleanly. The other engines are ordered by the seed alone.
func TestSeedOrderMatchesRunnerOrder(t *testing.T) {
	var gotCompute []string
	for _, m := range All() {
		if m.Engine == EngineCompute {
			gotCompute = append(gotCompute, m.ID)
		}
	}
	assertOrder(t, "compute", gotCompute, ops.Names())
}

func assertOrder(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s manifest count %d != runner count %d", label, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s manifest[%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}

// TestComputeEngineDefaultIO pins the default that lets 34 compute manifests omit
// their ports: exactly one image input and one image output. An in-process
// operation that deviates must declare its ports, and then this catches a
// deviation that forgot to.
func TestComputeEngineDefaultIO(t *testing.T) {
	for _, m := range All() {
		if m.Engine != EngineCompute {
			continue
		}
		if len(m.IO.Inputs) != 1 {
			t.Errorf("%q has %d inputs, want exactly 1", m.ID, len(m.IO.Inputs))
			continue
		}
		if kind := m.IO.Inputs[0].Kind; kind != "image" && kind != "vector" {
			t.Errorf("%q input kind = %q, want image or vector", m.ID, kind)
		}
		if !m.IO.Inputs[0].Required {
			t.Errorf("%q input is not required", m.ID)
		}
		if m.IO.Output.Kind == "" {
			t.Errorf("%q has no output kind", m.ID)
		}
	}
}

func TestConsentWeightDefaultsToNoneForUnknown(t *testing.T) {
	if got := ConsentWeightFor("no_such_operation"); got != ConsentWeightNone {
		t.Errorf("unknown op weight = %q, want none", got)
	}
	if got := ConsentWeightFor("face_swap"); got != ConsentWeightHigh {
		t.Errorf("face_swap weight = %q, want high", got)
	}
	if got := ConsentWeightFor("naturalize"); got != ConsentWeightLow {
		t.Errorf("naturalize weight = %q, want low", got)
	}
	if got := ConsentWeightFor("upscale"); got != ConsentWeightNone {
		t.Errorf("upscale weight = %q, want none", got)
	}
}

func TestWeightsIsDefensiveCopy(t *testing.T) {
	w := Weights()
	w["naturalize"] = ConsentWeightHigh
	if got := ConsentWeightFor("naturalize"); got != ConsentWeightLow {
		t.Errorf("Weights() must return a copy; canonical naturalize weight changed to %q", got)
	}
}

func TestSortedNamesIsSorted(t *testing.T) {
	names := Default().SortedNames()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("SortedNames not sorted at %d: %q > %q", i, names[i-1], names[i])
		}
	}
}
