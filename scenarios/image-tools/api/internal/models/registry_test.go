package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestLoadSeed asserts the bundled catalog loads, validates, and upholds the
// seed-integrity invariants (commercial-clean, ComfyUI-optional, every op has a
// default, blocklist disjoint).
func TestLoadSeed(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.Models()) == 0 {
		t.Fatal("no models loaded")
	}
	if r.SchemaVersion() == "" {
		t.Fatal("missing schema version")
	}

	// Every operation that has models must have exactly one default, and that
	// default must be enabled + CPU-capable (headless-completeness tenet).
	for _, op := range r.SortedOperations() {
		def, ok := r.DefaultFor(op)
		if !ok {
			t.Errorf("operation %q has no default", op)
			continue
		}
		if !def.Enabled {
			t.Errorf("default for %q (%s) is not enabled", op, def.ID)
		}
		if !def.Hardware.CPUCapable {
			t.Errorf("default for %q (%s) is not CPU-capable (violates headless tenet)", op, def.ID)
		}
		if def.RequiresComfyUI {
			t.Errorf("default for %q (%s) requires ComfyUI", op, def.ID)
		}
	}

	// Commercial-clean gate across the whole catalog: no outright non-commercial
	// models, and any "conditional" entry must be opt-in (disabled) + annotated.
	for _, m := range r.Models() {
		if m.CapabilityLabels.CommercialUse == CommercialUseNo {
			t.Errorf("model %s is commercial_use=no", m.ID)
		}
		if m.CapabilityLabels.CommercialUse == CommercialUseConditional {
			if m.Enabled {
				t.Errorf("conditional model %s must not be enabled by default", m.ID)
			}
			if m.CapabilityLabels.CommercialUseNotes == "" {
				t.Errorf("conditional model %s lacks commercial_use_notes", m.ID)
			}
		}
		if _, blocked := r.IsBlocked(m.ID); blocked {
			t.Errorf("model %s is also blocklisted", m.ID)
		}
	}

	if len(r.Blocklist()) == 0 {
		t.Error("expected a non-empty blocklist")
	}
}

func TestSeedDoctorCatalogIsInstallable(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	report := r.DoctorCatalog()
	if !report.OK {
		t.Fatalf("seed doctor should pass after Phase 1 catalog hardening; findings: %+v", report.Findings)
	}
	for _, f := range report.Findings {
		if f.Severity == FindingError {
			t.Fatalf("seed doctor returned error finding after Phase 1 catalog hardening: %+v", f)
		}
	}
}

func TestWeightlessBackendsDoNotRequireInstallAssets(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, id := range []string{"naturalize-detail-v1", "normals-from-depth", "laplacian-blur", "goimagehash", "gozxing"} {
		m, ok := r.ByID(id)
		if !ok {
			t.Fatalf("seed missing %q", id)
		}
		if m.RequiresWeights() {
			t.Fatalf("%s should not require downloadable model weights", id)
		}
	}
}

// validSeed is a minimal well-formed catalog used to test rejection of mutations.
const validSeed = `{
  "schema_version": "1.0.0",
  "operations_vocabulary": ["upscale", "denoise"],
  "models": [
    {
      "id": "good-default",
      "name": "Good Default",
      "operations": ["upscale"],
      "default_for": ["upscale"],
      "tier": "default",
      "backend": "go-native",
      "requires_comfyui": false,
      "hardware": {"cpu_capable": true, "gpu_required": false, "min_vram_gb": 0, "min_ram_gb": 2},
      "capability_labels": {"commercial_use": "yes"},
      "enabled": true
    }
  ],
  "blocklist": [
    {"id": "bad-model", "operations": ["upscale"], "license": "NC", "reason": "non-commercial"}
  ]
}`

func TestParseValid(t *testing.T) {
	r, err := Parse([]byte(validSeed))
	if err != nil {
		t.Fatalf("Parse valid: %v", err)
	}
	if got, ok := r.DefaultFor("upscale"); !ok || got.ID != "good-default" {
		t.Fatalf("default upscale: %+v ok=%v", got, ok)
	}
	if _, ok := r.IsBlocked("bad-model"); !ok {
		t.Fatal("bad-model should be blocked")
	}
}

// TestParseRejectsMalformed mutates the valid seed and asserts each defect is
// rejected (registry_test requirement: malformed entries rejected).
func TestParseRejectsMalformed(t *testing.T) {
	mutate := func(fn func(m map[string]any)) []byte {
		var doc map[string]any
		if err := json.Unmarshal([]byte(validSeed), &doc); err != nil {
			t.Fatal(err)
		}
		fn(doc)
		out, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	firstModel := func(doc map[string]any) map[string]any {
		return doc["models"].([]any)[0].(map[string]any)
	}

	cases := []struct {
		name string
		want string
		data []byte
	}{
		{"missing schema", "schema_version", mutate(func(d map[string]any) { delete(d, "schema_version") })},
		{"empty vocab", "operations_vocabulary", mutate(func(d map[string]any) { d["operations_vocabulary"] = []any{} })},
		{"no models", "no models", mutate(func(d map[string]any) { d["models"] = []any{} })},
		{"missing id", "missing id", mutate(func(d map[string]any) { firstModel(d)["id"] = "" })},
		{"missing name", "missing name", mutate(func(d map[string]any) { firstModel(d)["name"] = "" })},
		{"no operations", "no operations", mutate(func(d map[string]any) { firstModel(d)["operations"] = []any{} })},
		{"bad tier", "invalid tier", mutate(func(d map[string]any) { firstModel(d)["tier"] = "supreme" })},
		{"missing backend", "missing backend", mutate(func(d map[string]any) { firstModel(d)["backend"] = "" })},
		{"op not in vocab", "not in vocabulary", mutate(func(d map[string]any) { firstModel(d)["operations"] = []any{"teleport"} })},
		{"default_for not in ops", "not in this model", mutate(func(d map[string]any) { firstModel(d)["default_for"] = []any{"denoise"} })},
		{"bad commercial_use", "invalid commercial_use", mutate(func(d map[string]any) {
			firstModel(d)["capability_labels"] = map[string]any{"commercial_use": "maybe"}
		})},
		{"unrunnable model", "neither cpu_capable nor gpu_required", mutate(func(d map[string]any) {
			firstModel(d)["hardware"] = map[string]any{"cpu_capable": false, "gpu_required": false}
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.data)
			if err == nil {
				t.Fatalf("expected rejection for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.want)
			}
		})
	}
}

func TestParseRejectsDuplicateID(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(validSeed), &doc); err != nil {
		t.Fatal(err)
	}
	models := doc["models"].([]any)
	dupModel := map[string]any{
		"id": "good-default", "name": "Dup", "operations": []any{"denoise"},
		"default_for": []any{"denoise"}, "tier": "default", "backend": "x",
		"hardware":          map[string]any{"cpu_capable": true},
		"capability_labels": map[string]any{"commercial_use": "yes"}, "enabled": true,
	}
	doc["models"] = append(models, dupModel)
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "duplicate model id") {
		t.Fatalf("expected duplicate id rejection, got %v", err)
	}
}

func TestSeedInvariantCatchesComfyUI(t *testing.T) {
	bad := strings.Replace(validSeed, `"requires_comfyui": false`, `"requires_comfyui": true`, 1)
	r, err := Parse([]byte(bad))
	if err != nil {
		t.Fatalf("Parse should pass structural validation: %v", err)
	}
	if err := r.validateSeedInvariants(); err == nil || !strings.Contains(err.Error(), "requires_comfyui") {
		t.Fatalf("expected comfyui invariant failure, got %v", err)
	}
}
