package models

import (
	"encoding/json"
	"fmt"
	"image-tools/internal/capabilities"
	"image-tools/internal/capability"
	"reflect"
	"strings"
	"testing"
)

// TestLoadSeed asserts the bundled catalog loads, validates, and upholds the
// seed-integrity invariants (restrictive licenses annotated, ComfyUI-optional,
// every op has a default, exclusions disjoint, license gates agree with rows).
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
	// SD 1.5's 4.27 GB checkpoint cannot be safely admitted with the old
	// 2 GB catalog estimate. Keep the residency claim above the weight size so
	// model loading/activations are not silently unaccounted.
	sd15, ok := r.ByID("sd-1.5")
	if !ok || sd15.Hardware.MinVRAMGB < 6 {
		t.Fatalf("sd-1.5 minimum VRAM = %+v, want a conservative >=6 GiB claim", sd15.Hardware)
	}
	u2netp, ok := r.ByID("u2netp")
	if !ok || !u2netp.Hardware.CPUCapable || u2netp.Hardware.MinVRAMGB < 1 {
		t.Fatalf("u2netp hardware = %+v, want CPU fallback and a positive CUDA admission budget", u2netp.Hardware)
	}

	// Every operation with an available default must select an enabled,
	// CPU-capable model. Experimental GPU-only operations may remain unavailable
	// while their only model is disabled; catalog doctor reports that state as a
	// warning instead of promoting it or breaking unrelated startup.
	doctor := r.DoctorCatalog()
	for _, op := range r.SortedOperations() {
		// A preset names its own model and runs another operation; the catalog
		// contract for the model belongs to the operation it runs.
		if m, ok := capability.Get(op); ok && m.Preset != nil {
			continue
		}
		def, ok := r.DefaultFor(op)
		if !ok {
			expectedUnavailable := false
			for _, finding := range doctor.Findings {
				if finding.Code == "operation_without_installable_enabled_model" && finding.Operation == op && finding.Severity == FindingWarning {
					expectedUnavailable = true
					break
				}
			}
			if !expectedUnavailable {
				t.Errorf("operation %q has no default and is not explicitly marked unavailable", op)
			}
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

	// License-label hygiene across the whole catalog: a restrictive label (non-
	// commercial or conditional) must be annotated; non-commercial must be prohibited.
	for _, m := range r.Models() {
		var rights struct {
			ArtifactID       string   `json:"artifact_id"`
			Classification   string   `json:"classification"`
			UnverifiedReason string   `json:"unverified_reason"`
			ReviewTriggers   []string `json:"review_triggers"`
		}
		if len(m.Rights) == 0 || json.Unmarshal(m.Rights, &rights) != nil {
			t.Errorf("model %s has no parseable shared rights record", m.ID)
		} else {
			if rights.ArtifactID != m.ID || rights.Classification == "" {
				t.Errorf("model %s rights identity/classification incomplete: %+v", m.ID, rights)
			}
			if rights.Classification == "unknown" && rights.UnverifiedReason == "" {
				t.Errorf("model %s has unknown rights without an explicit reason", m.ID)
			}
			if len(rights.ReviewTriggers) == 0 {
				t.Errorf("model %s rights has no review triggers", m.ID)
			}
		}
		if m.CapabilityLabels.CommercialUse == CommercialUseNo || m.CapabilityLabels.CommercialUse == CommercialUseConditional {
			if m.CapabilityLabels.CommercialUseNotes == "" {
				t.Errorf("restricted-license model %s lacks commercial_use_notes", m.ID)
			}
		}
		if m.CapabilityLabels.CommercialUse == CommercialUseNo && rights.Classification != "prohibited" {
			t.Errorf("non-commercial model %s rights classification = %q, want prohibited so monetized and redistributed use stay refused", m.ID, rights.Classification)
		}
		if gate, ok := r.Gate(m.ID); ok && gate.IsExclusion() {
			t.Errorf("model %s is seeded but excluded (%s)", m.ID, gate.ExclusionKind)
		}
	}

	// Licenses gate only monetized, hosted and redistributed use (operator stance
	// 2026-11-21); only a safety, technical or hardware-fit reason excludes.
	if len(r.Blocklist()) == 0 {
		t.Error("expected a non-empty blocklist")
	}
	for _, b := range r.Blocklist() {
		switch b.Gate {
		case GateLicense:
			for _, use := range b.GatedUses {
				if use == RightsUseLocalInference {
					t.Errorf("license gate %s restricts normal local use", b.ID)
				}
			}
		case GateExclusion:
			if b.ExclusionKind == "" {
				t.Errorf("exclusion %s has no non-license reason kind", b.ID)
			}
		default:
			t.Errorf("blocklist entry %s has gate %q", b.ID, b.Gate)
		}
	}
	if gate, ok := r.Gate("codeformer"); !ok || gate.IsExclusion() {
		t.Errorf("codeformer should be a license gate, got %+v ok=%v", gate, ok)
	}
	if gate, ok := r.Gate("flux-dev-and-fill"); !ok || gate.ExclusionKind != ExclusionHardwareFit {
		t.Errorf("flux-dev-and-fill should be a hardware-fit exclusion, got %+v ok=%v", gate, ok)
	}
}

// The 2026-11-21 evaluation picked the best normal-use models regardless of
// license: Depth Anything V2 Large for depth_map and RestoreFormer++ for
// face_restore, with the license-clean or runner-up rows still selectable.
func TestBestNormalUseDefaults(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for op, want := range map[string]string{"depth_map": "depth-anything-v2-large", "face_restore": "restoreformer-plus-plus"} {
		def, ok := r.DefaultFor(op)
		if !ok || def.ID != want {
			t.Errorf("default for %s = %q, want %q", op, def.ID, want)
		}
	}
	for _, id := range []string{"depth-anything-v2-small", "depth-anything-v2-large", "codeformer", "restoreformer-plus-plus"} {
		m, ok := r.ByID(id)
		if !ok || !m.Enabled {
			t.Errorf("%s should be an enabled catalog row", id)
			continue
		}
		for _, a := range m.Source.Assets {
			if a.SHA256 == "" && id != "depth-anything-v2-small" {
				t.Errorf("%s asset %s has no pinned sha256", id, a.Filename)
			}
		}
	}
	// Selection prefers the higher tier among GPU-viable candidates, so the
	// defaults must also out-rank their alternatives.
	large, _ := r.ByID("depth-anything-v2-large")
	small, _ := r.ByID("depth-anything-v2-small")
	rf, _ := r.ByID("restoreformer-plus-plus")
	cf, _ := r.ByID("codeformer")
	if large.Tier.rank() <= small.Tier.rank() || rf.Tier.rank() <= cf.Tier.rank() {
		t.Errorf("defaults must out-rank alternatives: large=%s small=%s rf=%s cf=%s", large.Tier, small.Tier, rf.Tier, cf.Tier)
	}
}

// Big-LaMa's pretrained-weight grant is unresolved upstream, so its rights facts
// stay conditional/unknown and monetized use stays refused. That no longer keeps
// it disabled (operator stance 2026-11-21: normal local use ignores the license
// class): the ONNX variant is the enabled object_removal default, while the
// experimental 1024 export stays disabled until its visual qualification.
func TestBigLamaKeepsUnresolvedRightsFactsButServesNormalUse(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, id := range []string{"big-lama", "big-lama-1024-export"} {
		model, ok := r.ByID(id)
		if !ok {
			t.Fatalf("model %q not found", id)
		}
		if model.CapabilityLabels.CommercialUse != CommercialUseConditional {
			t.Errorf("%s commercial_use = %q, want conditional while the upstream weight grant is unresolved", id, model.CapabilityLabels.CommercialUse)
		}
		notes := strings.ToLower(model.CapabilityLabels.CommercialUseNotes)
		if !strings.Contains(notes, "pretrained") || !strings.Contains(notes, "upstream") || !strings.Contains(notes, "issues/366") {
			t.Errorf("%s rights notes = %q, want the unresolved upstream pretrained-weight grant", id, model.CapabilityLabels.CommercialUseNotes)
		}
	}
	lama, _ := r.ByID("big-lama")
	if !lama.Enabled || !lama.IsDefaultFor("object_removal") {
		t.Errorf("big-lama enabled=%t default=%t, want the enabled object_removal default", lama.Enabled, lama.IsDefaultFor("object_removal"))
	}
	if export, _ := r.ByID("big-lama-1024-export"); export.Enabled {
		t.Error("big-lama-1024-export must stay disabled until its visual qualification")
	}
	if migan, _ := r.ByID("mi-gan"); migan.Enabled || migan.IsDefaultFor("object_removal") {
		t.Error("mi-gan needs an unprovisioned IOPaint backend and must not be an enabled default")
	}
}

func TestSwinIRWeightsRequireSeparateRightsConfirmation(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	model, ok := r.ByID("swinir")
	if !ok {
		t.Fatal("model \"swinir\" not found")
	}
	if model.Enabled || model.CapabilityLabels.CommercialUse != CommercialUseConditional {
		t.Errorf("SwinIR rights = enabled:%t commercial_use:%q, want disabled/conditional until pretrained-weight terms are clear", model.Enabled, model.CapabilityLabels.CommercialUse)
	}
	license := strings.ToLower(model.CapabilityLabels.License)
	if !strings.Contains(license, "repository code") || !strings.Contains(license, "pretrained-weight") {
		t.Errorf("SwinIR license = %q, want code and pretrained-weight terms distinguished", model.CapabilityLabels.License)
	}
	notes := strings.ToLower(model.CapabilityLabels.CommercialUseNotes)
	if !strings.Contains(notes, "academic-research-only") || !strings.Contains(notes, "original owners") || !strings.Contains(notes, "huggingface.co/datasets/eugenesiow/div2k") {
		t.Errorf("SwinIR rights notes = %q, want official dataset-rights limitation and unresolved weight grant", model.CapabilityLabels.CommercialUseNotes)
	}
}

// RestoreFormer++'s checkpoint rights stay unresolved (FFHQ training data), so
// its rights record keeps monetized use refused; that no longer keeps it out of
// normal use, where it is the face_restore default.
func TestRestoreFormerPlusPlusServesNormalUseWithUnresolvedRights(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := r.ByID("restoreformer-plus-plus")
	if !ok {
		t.Fatal("RestoreFormer++ model not found")
	}
	if !model.Enabled || model.CapabilityLabels.CommercialUse != CommercialUseConditional {
		t.Errorf("RestoreFormer++ = enabled:%t commercial_use:%q, want enabled/conditional", model.Enabled, model.CapabilityLabels.CommercialUse)
	}
	var rights struct {
		Classification string `json:"classification"`
	}
	if err := json.Unmarshal(model.Rights, &rights); err != nil || rights.Classification == "unrestricted" {
		t.Errorf("RestoreFormer++ rights classification = %q (%v), want a non-granting class until checkpoint rights are clear", rights.Classification, err)
	}
	notes := strings.ToLower(model.CapabilityLabels.CommercialUseNotes)
	for _, want := range []string{"ffhq", "cc-by-nc", "cc-by-nc-sa", "restoreformerplusplus", "nvlabs/ffhq-dataset"} {
		if !strings.Contains(notes, want) {
			t.Errorf("RestoreFormer++ rights notes = %q, want %q", model.CapabilityLabels.CommercialUseNotes, want)
		}
	}
}

func TestDefaultNSFWModelUsesPinnedRunnableONNXArtifact(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	model, ok := r.DefaultFor("nsfw_classify")
	if !ok || model.ID != "adamcodd-vit-nsfw" {
		t.Fatalf("default NSFW model = %+v, want adamcodd-vit-nsfw", model)
	}
	if len(model.Source.Assets) != 1 {
		t.Fatalf("NSFW assets = %d, want one load-smoked ONNX artifact", len(model.Source.Assets))
	}
	asset := model.Source.Assets[0]
	if asset.Filename != "model.onnx" || !strings.Contains(asset.URL, "/resolve/8587de998f441aac03fdd57a85d2e4cb808c7d64/onnx/model.onnx") {
		t.Fatalf("NSFW asset = %+v, want immutable full-precision model.onnx", asset)
	}
	if asset.MinBytes < 340_000_000 || model.SizeMBApprox < 340 {
		t.Fatalf("NSFW artifact sizing = min %d bytes / %d MB, want >=340MB full-precision artifact", asset.MinBytes, model.SizeMBApprox)
	}
}

func TestDisabledExperimentalImageToObjectHasNoSeededDefault(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	model, ok := r.ByID("triposr-vision-to-mesh")
	if !ok || model.Enabled {
		t.Fatalf("TripoSR must remain disabled in the seed until qualification completes: %+v", model)
	}
	if _, ok := r.DefaultFor("image_to_object"); ok {
		t.Fatal("unqualified GPU-only image_to_object must not become a seeded default")
	}
	for _, finding := range r.DoctorCatalog().Findings {
		if finding.Code == "operation_without_installable_enabled_model" && finding.Operation == "image_to_object" {
			if finding.Severity != FindingWarning {
				t.Fatalf("disabled experimental image_to_object availability must be a warning, got %+v", finding)
			}
			return
		}
	}
	t.Fatal("expected catalog doctor to report the disabled experimental operation as intentionally unavailable")
}

func TestLaMaBuildProfileIsClosedAndCannotEnableUnpinnedOutput(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := r.ByID("big-lama-1024-export")
	if !ok || model.Enabled {
		t.Fatalf("expected disabled LaMa export candidate, got %+v", model)
	}
	sourceHash := strings.Repeat("a", 64)
	model.Source.Assets[0].SHA256 = sourceHash
	model.BuildProfile = &BuildProfile{
		Kind: "lama-onnx-fixed", SourceAsset: "lama-source.zip", SourceSHA256: sourceHash,
		CheckpointAsset: "big-lama.safetensors", CheckpointSHA256: "12cb5eb5d515400f262b48d4e7c1994183e170aedb4b86699d9a55b770038e16",
		Resolution: 1024, Opset: 17, OutputFilename: "lama_1024.onnx",
	}
	if err := r.validateModel(model); err != nil {
		t.Fatalf("valid closed build profile rejected: %v", err)
	}
	mutated := model
	bad := *model.BuildProfile
	bad.Kind = "shell"
	mutated.BuildProfile = &bad
	if err := r.validateModel(mutated); err == nil {
		t.Fatal("arbitrary build kind was accepted")
	}
	mutated = model
	mutated.Enabled = true
	if err := r.validateModel(mutated); err == nil {
		t.Fatal("build candidate enabled without pinned output hash")
	}
	mutated = model
	bad = *model.BuildProfile
	bad.SourceSHA256 = strings.Repeat("g", 64)
	mutated.BuildProfile = &bad
	if err := r.validateModel(mutated); err == nil {
		t.Fatal("build profile accepted a 64-character non-hex source digest")
	}
}

func TestEveryDisabledSeedRowSaysWhy(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	disabled := 0
	for _, m := range r.Models() {
		if m.Enabled {
			continue
		}
		disabled++
		if m.Disabled == nil || m.Disabled.Validate() != nil {
			t.Errorf("%s is seeded disabled without a valid reason: %+v", m.ID, m.Disabled)
		}
	}
	if disabled == 0 {
		t.Fatal("expected disabled seed rows to check")
	}
}

func TestSdxlInpaintUsesPinnedFp16RepositorySubsetAndRemainsDisabled(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := r.ByID("sdxl-inpainting-0.1")
	if !ok {
		t.Fatal("catalog is missing the SDXL inpainting candidate")
	}
	if model.Enabled {
		t.Fatal("SDXL inpainting must remain opt-in until visual qualification passes")
	}
	if model.Disabled == nil || model.Disabled.Kind != DisabledCandidate {
		t.Fatalf("SDXL inpainting is a candidate until visually qualified, got %+v", model.Disabled)
	}
	if model.Source.Repo.RepoID != "diffusers/stable-diffusion-xl-1.0-inpainting-0.1" || model.Source.Repo.Revision != "115134f363124c53c7d878647567d04daf26e41e" {
		t.Fatalf("SDXL inpaint source is not pinned: %+v", model.Source.Repo)
	}
	if model.Runtime.Variant != "fp16" {
		t.Fatalf("SDXL inpaint variant=%q, want fp16", model.Runtime.Variant)
	}
	patterns := strings.Join(model.Source.Repo.AllowPatterns, " ")
	if !strings.Contains(patterns, "**/*.fp16.safetensors") {
		t.Fatalf("SDXL inpaint install must select fp16 weights only: %v", model.Source.Repo.AllowPatterns)
	}
	for _, pattern := range model.Source.Repo.AllowPatterns {
		if strings.Contains(pattern, "*.safetensors") && !strings.Contains(pattern, ".fp16.safetensors") {
			t.Fatalf("SDXL inpaint install allows non-fp16 weights: %s", pattern)
		}
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

// TestSeedFetchStrategyContractIsClean is the build-time guard that every weight-backed
// seed row — enabled OR disabled — declares a concrete fetch strategy
// (assets[]/repo/local_path) or is honestly marked source.manual. It is what
// stops a landing-page-only stub from shipping and failing at a user's install.
func TestSeedFetchStrategyContractIsClean(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, model := range r.models {
		if finding := fetchStrategyFinding(model); finding != nil {
			t.Fatalf("seed model %q violates fetch strategy contract: %+v", model.ID, finding)
		}
	}
}

// TestDoctorFlagsLandingPageStub proves DoctorCatalog checks the fetch contract over every row,
// whose only source is a documentation download_url, and that marking it manual
// (or giving it a concrete strategy) clears the finding.
func TestDoctorFlagsLandingPageStub(t *testing.T) {
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
	report := r.DoctorCatalog()
	finding := findFinding(report.Findings, "model_without_fetch_strategy")
	if finding == nil {
		t.Fatalf("disabled stub should yield model_without_fetch_strategy finding: %+v", report.Findings)
	}
	if finding.Severity != FindingWarning {
		t.Errorf("disabled stub finding should be a warning, got %q", finding.Severity)
	}

	// enabled stub → error (and not OK).
	r, _ = Parse([]byte(fmt.Sprintf(base, "true", "")))
	report = r.DoctorCatalog()
	finding = findFinding(report.Findings, "model_without_fetch_strategy")
	if finding == nil || finding.Severity != FindingError {
		t.Fatalf("enabled stub must be an error: %+v", report.Findings)
	}

	// marked manual → clean.
	r, _ = Parse([]byte(fmt.Sprintf(base, "false", `"manual": true, `)))
	report = r.DoctorCatalog()
	if finding := findFinding(report.Findings, "model_without_fetch_strategy"); finding != nil {
		t.Fatalf("manual-marked stub must not get fetch-strategy finding: %+v", finding)
	}
}

func findFinding(findings []CatalogFinding, code string) *CatalogFinding {
	for i := range findings {
		if findings[i].Code == code {
			return &findings[i]
		}
	}
	return nil
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
//   - qwen-image-edit and flux-2-klein-4b ship ENABLED: each has a repo fetch
//     strategy with a PINNED revision and a proven (Ready) family adapter, and was
//     validated end-to-end on a GPU host. flux-2-klein-4b additionally pins its
//     weight files and fetches only the diffusers tree (not the duplicate root
//     single-file checkpoint).
//   - longcat-image-edit-turbo ships DISABLED: it has no fetch strategy yet AND
//     its family adapter is not proven, so force-enabling it fires both
//     `enabled_model_without_assets` and `enabled_edit_model_family_not_ready`.
//
// The shipped seed is clean.
func TestNewEditInstructModelsInstallabilityContract(t *testing.T) {
	enabledProven := []string{"qwen-image-edit", "flux-2-klein-4b"}
	pending := "longcat-image-edit-turbo"

	base, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !base.DoctorCatalog().OK {
		t.Fatalf("shipped seed catalog should be clean: %+v", base.DoctorCatalog().Findings)
	}

	for _, id := range enabledProven {
		m, ok := base.ByID(id)
		if !ok || !m.Enabled {
			t.Fatalf("%s should ship enabled after the e2e proof", id)
		}
		if strings.TrimSpace(m.Source.Repo.Revision) == "" {
			t.Fatalf("%s repo source must pin a revision", id)
		}
		if fam, ok := DiffusersFamilyByName(m.Runtime.Family); !ok || !fam.Ready {
			t.Fatalf("%s family %q must be a proven adapter", id, m.Runtime.Family)
		}
	}

	klein, _ := base.ByID("flux-2-klein-4b")
	if klein.Source.Manual {
		t.Fatal("flux-2-klein-4b must be auto-installable (repo snapshot), not manual")
	}
	if got := klein.Source.Repo.RepoID; got != "black-forest-labs/FLUX.2-klein-4B" {
		t.Fatalf("flux-2-klein-4b must fetch the Apache 4B repo, got %q", got)
	}
	if len(klein.Source.Repo.FileSHA256) == 0 {
		t.Fatal("flux-2-klein-4b must pin its weight file digests")
	}
	for _, pattern := range klein.Source.Repo.AllowPatterns {
		if pattern == "*" || strings.HasSuffix(pattern, ".safetensors") && !strings.Contains(pattern, "/") {
			t.Fatalf("flux-2-klein-4b allow pattern %q would fetch the duplicate root checkpoint", pattern)
		}
	}

	m, ok := base.ByID(pending)
	if !ok {
		t.Fatalf("seed missing %q", pending)
	}
	if m.Enabled {
		t.Fatalf("%s ships enabled but its family adapter is not proven", pending)
	}

	// Force-enabling the pending model fires both invariants.
	r := parseSeedWithEnabled(t, pending)
	report := r.DoctorCatalog()
	if report.OK {
		t.Fatal("DoctorCatalog should not be OK with the pending model enabled")
	}
	var codes []string
	for _, f := range report.Findings {
		if f.ModelID == pending {
			codes = append(codes, f.Code)
		}
	}
	if !containsCode(codes, "enabled_model_without_assets") {
		t.Errorf("%s: expected enabled_model_without_assets, got %v", pending, codes)
	}
	if !containsCode(codes, "enabled_edit_model_family_not_ready") {
		t.Errorf("%s: expected enabled_edit_model_family_not_ready, got %v", pending, codes)
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
    {"id": "bad-model", "operations": ["upscale"], "license": "NC", "reason": "non-commercial", "gate": "license", "gated_uses": ["monetized_use"]}
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
	if gate, ok := r.Gate("bad-model"); !ok || gate.Gate != GateLicense {
		t.Fatalf("bad-model should be license-gated, got %+v ok=%v", gate, ok)
	}
	if !r.IsOperation("upscale") || r.IsOperation("teleport") {
		t.Fatalf("operation membership mismatch")
	}
	// The vocabulary is the SSOT (internal/operations), not declared in the seed;
	// Operations() exposes it verbatim regardless of which ops the seed's models use.
	if got := r.Operations(); !reflect.DeepEqual(got, capability.VocabularyNames()) {
		t.Fatalf("operations = %v, want manifest vocabulary %v", got, capability.VocabularyNames())
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
	firstGate := func(doc map[string]any) map[string]any {
		return doc["blocklist"].([]any)[0].(map[string]any)
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
		{"gate missing", "want license or exclusion", mutate(func(d map[string]any) {
			delete(firstGate(d), "gate")
		})},
		{"license gate restricting normal use", "never normal local inference", mutate(func(d map[string]any) {
			firstGate(d)["gated_uses"] = []any{"monetized_use", "local_inference"}
		})},
		{"license gate without uses", "no gated_uses", mutate(func(d map[string]any) {
			delete(firstGate(d), "gated_uses")
		})},
		{"exclusion with a license-only reason", "license-only reason is a license gate", mutate(func(d map[string]any) {
			firstGate(d)["gate"] = "exclusion"
			delete(firstGate(d), "gated_uses")
		})},
		{"gate without reason", "has no reason", mutate(func(d map[string]any) {
			firstGate(d)["reason"] = ""
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
			name: "commercial no missing notes",
			want: "commercial_use=no but carries no commercial_use_notes",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["capability_labels"] = map[string]any{"commercial_use": "no"}
			}),
		},
		{
			name: "conditional missing notes",
			want: "commercial_use_notes",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["enabled"] = false
				m["disabled"] = map[string]any{"kind": "candidate", "reason": "kept for comparison"}
				m["capability_labels"] = map[string]any{"commercial_use": "conditional"}
			}),
		},
		{
			name: "disabled with no reason",
			want: "no \"disabled\" reason",
			reg:  mutate(func(m map[string]any, _ map[string]any) { m["enabled"] = false }),
		},
		{
			name: "enabled but still carrying a reason",
			want: "still carries a \"disabled\" reason",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["disabled"] = map[string]any{"kind": "runtime", "reason": "stale"}
			}),
		},
		{
			name: "licensing is never a reason to disable",
			want: "licensing is never a reason",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["enabled"] = false
				m["disabled"] = map[string]any{"kind": "licence", "reason": "weights unverified"}
			}),
		},
		{
			name: "multi-line reason",
			want: "one non-empty line",
			reg: mutate(func(m map[string]any, _ map[string]any) {
				m["enabled"] = false
				m["disabled"] = map[string]any{"kind": "runtime", "reason": "a\nb"}
			}),
		},
		{
			name: "exclusion overlap",
			want: "is excluded",
			reg: mutate(func(m map[string]any, doc map[string]any) {
				doc["blocklist"] = append(doc["blocklist"].([]any), map[string]any{
					"id": m["id"], "operations": []any{"upscale"}, "license": "any", "reason": "test overlap",
					"gate": "exclusion", "exclusion_kind": "technical",
				})
			}),
		},
		{
			// A license gate may cover a seeded row, but the row must not claim
			// the rights the gate refuses.
			name: "license gate on a commercial row",
			want: "license-gated but labelled commercial_use=yes",
			reg: mutate(func(m map[string]any, doc map[string]any) {
				doc["blocklist"] = append(doc["blocklist"].([]any), map[string]any{
					"id": m["id"], "operations": []any{"upscale"}, "license": "NC", "reason": "test gate",
					"gate": "license", "gated_uses": []any{"monetized_use"},
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

// FLUX.2 klein is the default generate/transform/edit model wherever a GPU can
// hold one streamed stage, and CPU-only hosts keep their CPU-capable defaults.
// The default comes from tier ranking (klein is the enabled quality-tier
// candidate), not a default_for entry, so it must never reach a host that
// cannot run it.
func TestFluxKleinIsTheGPUDefaultAndCPUHostsKeepCPUDefaults(t *testing.T) {
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	gpuHost := capabilities.Host{OS: "linux", Arch: "amd64", Cores: 16, TotalMemoryBytes: 64 << 30, AvailableMemoryBytes: 32 << 30,
		GPUs: []capabilities.GPU{{Name: "rtx-4070-ti-super", VRAMBytes: 16 << 30, VRAMUsedBytes: 9 << 30}}}
	cpuOnly := capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, TotalMemoryBytes: 32 << 30, AvailableMemoryBytes: 16 << 30}
	for _, operation := range []string{"text_to_image", "image_to_image", "edit_instruct"} {
		gpu, err := registry.Select(SelectRequest{Operation: operation, Host: gpuHost, CapacityManagedGPU: true, RightsEvaluator: allowRightsForTest}, nil)
		if err != nil {
			t.Fatalf("%s on a 16 GB GPU host: %v", operation, err)
		}
		if gpu.Model.ID != "flux-2-klein-4b" {
			t.Errorf("%s GPU default = %s, want flux-2-klein-4b", operation, gpu.Model.ID)
		}
		cpu, err := registry.Select(SelectRequest{Operation: operation, Host: cpuOnly, RightsEvaluator: allowRightsForTest}, nil)
		if err != nil {
			t.Fatalf("%s on a CPU-only host: %v", operation, err)
		}
		if cpu.Model.ID == "flux-2-klein-4b" || !cpu.Model.Hardware.CPUCapable {
			t.Errorf("%s CPU-only default = %s, want a CPU-capable model", operation, cpu.Model.ID)
		}
	}
}
