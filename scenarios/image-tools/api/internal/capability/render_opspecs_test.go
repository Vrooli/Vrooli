package capability

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGeneratedOpSpecsIsUpToDate is the drift gate for the deterministic
// parameter forms. The UI used to carry a hand-maintained map that could name a
// field the wire did not have (or miss a new one) with no build failure; the
// forms now come from the manifest and are cross-checked against the OpParams
// descriptor during generation.
//
// Fix by running `make capability-gen` from the scenario root.
func TestGeneratedOpSpecsIsUpToDate(t *testing.T) {
	want, err := RenderOpSpecs()
	if err != nil {
		t.Fatalf("RenderOpSpecs: %v", err)
	}
	path := filepath.Join("..", "..", "..", "ui", "src", "features", "workspace", "opSpecs.generated.ts")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated op specs %s: %v (run `make capability-gen`)", path, err)
	}
	if string(got) != want {
		t.Fatalf("generated op specs are stale; run `make capability-gen` from the scenario root")
	}
}

// TestEveryDeclaredFormFieldExistsOnTheWire proves the generator's second
// direction directly: a manifest may not name a presentation field whose
// OpParams message has no such member. RenderOpSpecs enforces this, and this test
// keeps the failure legible when someone edits the seed by hand.
func TestEveryDeclaredFormFieldExistsOnTheWire(t *testing.T) {
	specs, err := opsRunSpecs()
	if err != nil {
		t.Fatalf("opsRunSpecs: %v", err)
	}
	wire := make(map[string]map[string]struct{}, len(specs))
	for _, spec := range specs {
		set := make(map[string]struct{}, len(spec.flags))
		for _, f := range spec.flags {
			set[f.field] = struct{}{}
		}
		wire[spec.op] = set
	}
	for _, m := range All() {
		if m.Presentation == nil {
			continue
		}
		for _, f := range m.Presentation.Fields {
			if _, ok := wire[m.ID][f.Name]; !ok {
				t.Errorf("operation %q form field %q has no OpParams member", m.ID, f.Name)
			}
		}
	}
}
