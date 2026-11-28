package capability_test

import (
	"testing"

	"image-tools/internal/capability"
	"image-tools/internal/models"
	"image-tools/internal/ops"
	"image-tools/internal/safety"
)

// TestManifestsAreTheOperationSet is the coverage gate in the direction that
// cannot drift: the manifest set is the declaration of record, so every
// operation an engine can run must be declared here, and no manifest may be
// duplicated. The reverse direction (declared but not runnable) is proven by the
// engine-agreement test in internal/ai.
func TestManifestsAreTheOperationSet(t *testing.T) {
	seen := make(map[string]bool, len(capability.All()))
	for _, m := range capability.All() {
		if seen[m.ID] {
			t.Errorf("operation %q is declared twice", m.ID)
		}
		seen[m.ID] = true
	}
	for _, op := range ops.Names() {
		if !seen[op] {
			t.Errorf("in-process operation %q has no capability manifest", op)
		}
	}
	if len(seen) < len(ops.Names()) {
		t.Errorf("manifest count %d is smaller than the in-process table", len(seen))
	}
}

// TestComputeManifestsAreInProcess keeps the compute engine honest: those ops run
// in the API binary, so they must be compute with no backend and no external
// requirement.
func TestComputeManifestsAreInProcess(t *testing.T) {
	for _, m := range capability.All() {
		if m.Engine != capability.EngineCompute {
			continue
		}
		if m.Mechanism != capability.MechanismCompute {
			t.Errorf("%q: compute-engine op mechanism = %q, want compute", m.ID, m.Mechanism)
		}
		if len(m.Backends) != 0 {
			t.Errorf("%q: compute-engine op declares backends %v", m.ID, m.Backends)
		}
		if m.Requires.Process || m.Requires.Weights || m.Requires.Network {
			t.Errorf("%q: compute-engine op declares an external requirement %+v", m.ID, m.Requires)
		}
		if m.ConsentWeight != capability.ConsentWeightNone {
			t.Errorf("%q: compute-engine op consent weight = %q, want none", m.ID, m.ConsentWeight)
		}
	}
}

// TestManifestBackendsExistInModelSeed ties the manifest's ordered backend
// preference to the model registry: every declared backend must be the backend of
// at least one seeded model that serves that operation. A typo or a retired
// backend fails here rather than at selection time.
func TestManifestBackendsExistInModelSeed(t *testing.T) {
	reg, err := models.Load()
	if err != nil {
		t.Fatalf("models.Load: %v", err)
	}
	byID := make(map[string]models.Model, len(reg.Models()))
	for _, model := range reg.Models() {
		byID[model.ID] = model
	}
	for _, m := range capability.All() {
		if len(m.Backends) == 0 {
			continue
		}
		// A preset has no models of its own: it is a named way of running another
		// operation, so its named model must serve that operation on one of the
		// backends the preset declares. The bare-operation check below would ask
		// for a seeded model that serves the preset id, which is not a thing.
		if m.Preset != nil {
			checkPresetModel(t, m, byID)
			continue
		}
		declared := make(map[string]struct{})
		for _, model := range reg.Models() {
			for _, op := range model.Operations {
				if op == m.ID {
					declared[model.Backend] = struct{}{}
					break
				}
			}
		}
		for _, b := range m.Backends {
			if _, ok := declared[b]; !ok {
				t.Errorf("manifest %q declares backend %q, but no seeded model for that operation uses it", m.ID, b)
			}
		}
	}
}

// checkPresetModel proves a preset names a real model that can run the operation
// behind it on the backend the preset declares. A typo in either name fails here,
// not at the first use.
func checkPresetModel(t *testing.T, m capability.Manifest, byID map[string]models.Model) {
	t.Helper()
	model, ok := byID[m.Preset.Model]
	if !ok {
		t.Errorf("preset %q names model %q, which is not in the seed", m.ID, m.Preset.Model)
		return
	}
	serves := false
	for _, op := range model.Operations {
		if op == m.Preset.Runs {
			serves = true
			break
		}
	}
	if !serves {
		t.Errorf("preset %q names model %q, which does not serve %q", m.ID, model.ID, m.Preset.Runs)
	}
	backendOK := false
	for _, b := range m.Backends {
		if b == model.Backend {
			backendOK = true
			break
		}
	}
	if !backendOK {
		t.Errorf("preset %q declares backends %v but its model %q runs on %q", m.ID, m.Backends, model.ID, model.Backend)
	}
}

// TestManifestRequirementClassMatchesMechanism keeps the requirement facts
// coherent: compute ops never require weights or network, infer ops require a
// process, and fetch ops require network.
func TestManifestRequirementClassMatchesMechanism(t *testing.T) {
	for _, m := range capability.All() {
		switch m.Mechanism {
		case capability.MechanismCompute:
			if m.Requires.Weights {
				t.Errorf("%q is compute but requires weights", m.ID)
			}
			if m.Requires.Network {
				t.Errorf("%q is compute but requires network", m.ID)
			}
		case capability.MechanismInfer:
			if !m.Requires.Process {
				t.Errorf("%q is infer but declares no process requirement", m.ID)
			}
		case capability.MechanismFetch:
			if !m.Requires.Network {
				t.Errorf("%q is fetch but does not require network", m.ID)
			}
		default:
			t.Errorf("%q has unknown mechanism %q", m.ID, m.Mechanism)
		}
	}
}

// TestManifestPortsAreDeclared proves the ports a composition step type-checks
// against are actually declared: every non-compute operation consumes something,
// and every operation produces a typed output. Ports *are* the I/O contract now,
// so this catches the empty-port manifest that would silently disable workflow
// validation for that operation.
func TestManifestPortsAreDeclared(t *testing.T) {
	for _, m := range capability.All() {
		if m.Engine == capability.EngineCompute {
			// In-process operations all edit an image or render one (rasterize).
			hasImage := false
			for _, in := range m.IO.Inputs {
				if in.Kind == "image" || in.Kind == "vector" {
					hasImage = true
				}
			}
			if !hasImage {
				t.Errorf("%q declares no image/vector input", m.ID)
			}
			continue
		}
		if len(m.IO.Inputs) == 0 {
			t.Errorf("%q declares no inputs", m.ID)
		}
		if m.IO.Output.Kind == "" {
			t.Errorf("%q declares no output kind", m.ID)
		}
	}
}

// TestManifestConsentWeightIsAuthoritative proves internal/safety reads the
// manifest set rather than carrying a second table. If someone re-adds a hand
// table that disagrees, this fails.
func TestManifestConsentWeightIsAuthoritative(t *testing.T) {
	for _, m := range capability.Vocabulary() {
		want := capability.ConsentWeightFor(m.ID)
		got := safety.OpWeight(m.ID)
		if string(got) != string(want) {
			t.Errorf("%q consent weight: safety=%q manifest=%q", m.ID, got, want)
		}
	}
}
