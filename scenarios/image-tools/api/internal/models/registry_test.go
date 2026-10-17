package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"image-tools/internal/operations"
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

// TestSeedRegistryLintIsClean is the build-time guard that every weight-backed
// seed row — enabled OR disabled — declares a concrete fetch strategy
// (assets[]/repo/local_path) or is honestly marked source.manual. It is what
// stops a landing-page-only stub from shipping and failing at a user's install.
func TestSeedRegistryLintIsClean(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	lint := r.RegistryLint()
	if !lint.OK || len(lint.Findings) != 0 {
		t.Fatalf("seed registry-lint must be clean (every weight-backed row has a fetch strategy or source.manual); findings: %+v", lint.Findings)
	}
}

// TestRegistryLintFlagsLandingPageStub proves the lint catches a weight-backed row
// whose only source is a documentation download_url, and that marking it manual
// (or giving it a concrete strategy) clears the finding.
func TestRegistryLintFlagsLandingPageStub(t *testing.T) {
	base := `{
	  "schema_version": "1.0.0",
	  "models": [
	    {
	      "id": "stub", "name": "Landing Page Stub", "operations": ["upscale"],
	      "default_for": [], "tier": "nice-to-have", "backend": "onnxruntime",
	      "hardware": {"cpu_capable": true}, "capability_labels": {"commercial_use": "yes"},
	      "enabled": %s,
	      "source": {%s"download_url": "https://example.com/model-landing-page"}
	    }
	  ]
	}`
	// disabled stub with only a download_url → warning.
	r, err := Parse([]byte(fmt.Sprintf(base, "false", "")))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	lint := r.RegistryLint()
	if len(lint.Findings) != 1 || lint.Findings[0].Code != "model_without_fetch_strategy" {
		t.Fatalf("disabled stub should yield one model_without_fetch_strategy finding: %+v", lint.Findings)
	}
	if lint.Findings[0].Severity != FindingWarning {
		t.Errorf("disabled stub finding should be a warning, got %q", lint.Findings[0].Severity)
	}
	if !lint.OK {
		t.Error("a disabled-only (warning) lint should still be OK=true")
	}

	// enabled stub → error (and not OK).
	r, _ = Parse([]byte(fmt.Sprintf(base, "true", "")))
	if lint := r.RegistryLint(); lint.OK || lint.Findings[0].Severity != FindingError {
		t.Fatalf("enabled stub must be an error: %+v", lint.Findings)
	}

	// marked manual → clean.
	r, _ = Parse([]byte(fmt.Sprintf(base, "false", `"manual": true, `)))
	if lint := r.RegistryLint(); !lint.OK || len(lint.Findings) != 0 {
		t.Fatalf("manual-marked stub must lint clean: %+v", lint.Findings)
	}
}

// parseSeedWithEnabled re-parses the bundled seed with the given model ids forced
// enabled, returning the indexed registry. It edits the raw JSON (not a typed
// Model) so the overlay exercises the same Parse path the loader uses.
func parseSeedWithEnabled(t *testing.T, ids ...string) *Registry {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(seedBytes, &doc); err != nil {
		t.Fatalf("unmarshal seed: %v", err)
	}
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	for _, raw := range doc["models"].([]any) {
		m := raw.(map[string]any)
		if want[m["id"].(string)] {
			m["enabled"] = true
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal seed overlay: %v", err)
	}
	r, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse seed overlay: %v", err)
	}
	return r
}

// TestNewEditInstructModelsInstallabilityContract pins the per-model doctor
// verdict for the three diffusers instruction-edit models:
//   - qwen-image-edit ships ENABLED: it has a repo fetch strategy with a PINNED
//     revision (Phase 2) and a proven (Ready) family adapter, and was validated
//     end-to-end on a GPU host. It is doctor-clean.
//   - flux-2-klein-4b and longcat-image-edit-turbo ship DISABLED: they have no
//     fetch strategy yet AND their family adapters are not proven, so force-enabling
//     either fires both `enabled_model_without_assets` and
//     `enabled_edit_model_family_not_ready`.
//
// The shipped seed (with qwen enabled) is clean.
func TestNewEditInstructModelsInstallabilityContract(t *testing.T) {
	enabledProven := "qwen-image-edit"
	pending := []string{"flux-2-klein-4b", "longcat-image-edit-turbo"}

	base, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !base.DoctorCatalog().OK {
		t.Fatal("shipped seed catalog should be clean with qwen enabled")
	}

	// qwen ships enabled, weight-backed, with a pinned repo revision.
	qwen, ok := base.ByID(enabledProven)
	if !ok || !qwen.Enabled {
		t.Fatalf("%s should ship enabled after the e2e proof", enabledProven)
	}
	if strings.TrimSpace(qwen.Source.Repo.Revision) == "" {
		t.Fatalf("%s repo source must pin a revision", enabledProven)
	}

	// flux/longcat ship disabled and must stay so until their adapters are proven.
	for _, id := range pending {
		m, ok := base.ByID(id)
		if !ok {
			t.Fatalf("seed missing %q", id)
		}
		if m.Enabled {
			t.Fatalf("%s ships enabled but its family adapter is not proven", id)
		}
	}

	// Force-enabling the pending pair fires both invariants.
	r := parseSeedWithEnabled(t, pending...)
	report := r.DoctorCatalog()
	if report.OK {
		t.Fatal("DoctorCatalog should not be OK with the pending models enabled")
	}
	byModel := make(map[string][]string)
	for _, f := range report.Findings {
		byModel[f.ModelID] = append(byModel[f.ModelID], f.Code)
	}
	for _, id := range pending {
		codes := byModel[id]
		if !containsCode(codes, "enabled_model_without_assets") {
			t.Errorf("%s: expected enabled_model_without_assets, got %v", id, codes)
		}
		if !containsCode(codes, "enabled_edit_model_family_not_ready") {
			t.Errorf("%s: expected enabled_edit_model_family_not_ready, got %v", id, codes)
		}
	}
}

func containsCode(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
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
	if !r.IsOperation("upscale") || r.IsOperation("teleport") {
		t.Fatalf("operation membership mismatch")
	}
	// The vocabulary is the SSOT (internal/operations), not declared in the seed;
	// Operations() exposes it verbatim regardless of which ops the seed's models use.
	if got := r.Operations(); !reflect.DeepEqual(got, operations.Names()) {
		t.Fatalf("operations = %v, want SSOT vocabulary %v", got, operations.Names())
	}
	models := r.ForOperation("upscale")
	if len(models) != 1 || models[0].ID != "good-default" {
		t.Fatalf("upscale models = %+v", models)
	}
	models[0].ID = "mutated"
	if again := r.ForOperation("upscale"); again[0].ID != "good-default" {
		t.Fatalf("ForOperation should return a copy, got %+v", again)
	}
	m, ok := r.ByID("good-default")
	if !ok || !m.ServesOperation("upscale") || m.ServesOperation("denoise") || !m.IsDefaultFor("upscale") || m.IsDefaultFor("denoise") {
		t.Fatalf("model operation/default helpers returned unexpected values: %+v ok=%v", m, ok)
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
		{"no models", "no models", mutate(func(d map[string]any) { d["models"] = []any{} })},
		{"missing id", "missing id", mutate(func(d map[string]any) { firstModel(d)["id"] = "" })},
		{"missing name", "missing name", mutate(func(d map[string]any) { firstModel(d)["name"] = "" })},
		{"no operations", "no operations", mutate(func(d map[string]any) { firstModel(d)["operations"] = []any{} })},
		{"bad tier", "invalid tier", mutate(func(d map[string]any) { firstModel(d)["tier"] = "supreme" })},
		{"missing backend", "missing backend", mutate(func(d map[string]any) { firstModel(d)["backend"] = "" })},
		{"op not in vocab", "not in vocabulary", mutate(func(d map[string]any) { firstModel(d)["operations"] = []any{"teleport"} })},
		{"default_for not in ops", "not in this model", mutate(func(d map[string]any) { firstModel(d)["default_for"] = []any{"denoise"} })},
		{"duplicate op on model", "listed twice", mutate(func(d map[string]any) {
			firstModel(d)["operations"] = []any{"upscale", "upscale"}
		})},
		{"negative figures", "negative", mutate(func(d map[string]any) {
			firstModel(d)["size_mb_approx"] = -1
		})},
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

func TestSeedInvariantPolicyFailures(t *testing.T) {
	mutate := func(fn func(m map[string]any, doc map[string]any)) *Registry {
		t.Helper()
		var doc map[string]any
		if err := json.Unmarshal([]byte(validSeed), &doc); err != nil {
			t.Fatal(err)
		}
		model := doc["models"].([]any)[0].(map[string]any)
		fn(model, doc)
		out, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		r, err := Parse(out)
		if err != nil {
			t.Fatalf("Parse should pass structural validation: %v", err)
		}
		return r
	}

	cases := []struct {
		name string
		want string
		reg  *Registry
	}{
		{
			name: "commercial no",
			want: "commercial_use=no",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["capability_labels"] = map[string]any{"commercial_use": "no"}
			}),
		},
		{
			name: "enabled conditional",
			want: "conditional",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["capability_labels"] = map[string]any{"commercial_use": "conditional", "commercial_use_notes": "requires review"}
			}),
		},
		{
			name: "conditional missing notes",
			want: "commercial_use_notes",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["enabled"] = false
				m["capability_labels"] = map[string]any{"commercial_use": "conditional"}
			}),
		},
		{
			name: "blocklist overlap",
			want: "blocklist",
			reg: mutate(func(m map[string]any, doc map[string]any) {
				doc["blocklist"] = append(doc["blocklist"].([]any), map[string]any{
					"id": m["id"], "operations": []any{"upscale"}, "license": "blocked", "reason": "test overlap",
				})
			}),
		},
		{
			name: "missing default",
			want: "no seeded default",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["default_for"] = []any{}
			}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.reg.validateSeedInvariants(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected invariant error containing %q, got %v", tc.want, err)
			}
		})
	}
}
