package models

import (
	"context"
	"sort"
	"strings"
	"testing"

	"image-tools/internal/hfmeta"
)

func TestProposeImport_DiffusersRepoSDXL(t *testing.T) {
	meta := hfmeta.Metadata{
		Source:        "stabilityai/stable-diffusion-xl-base-1.0",
		RepoID:        "stabilityai/stable-diffusion-xl-base-1.0",
		Revision:      "462165984030d82259a11f4367a4eed129e94a7b",
		Layout:        hfmeta.LayoutDiffusersRepo,
		PipelineClass: "StableDiffusionXLPipeline",
		License:       "openrail++",
		Files:         []hfmeta.FileInfo{{Path: "model_index.json", Size: 600}},
	}
	p := ProposeImport(meta)
	if p.Architecture != ArchSDXL || p.Confidence != ConfidenceHigh {
		t.Fatalf("arch/conf = %q/%q", p.Architecture, p.Confidence)
	}
	if p.Entry.Backend != BackendDiffusers {
		t.Fatalf("backend = %q, want diffusers", p.Entry.Backend)
	}
	if p.Entry.Source.Repo.RepoID != meta.RepoID || p.Entry.Source.Repo.Revision != meta.Revision {
		t.Fatalf("repo source = %+v", p.Entry.Source.Repo)
	}
	if p.Entry.CapabilityLabels.Provenance != ProvenanceUserImported {
		t.Fatalf("provenance = %q", p.Entry.CapabilityLabels.Provenance)
	}
	// Offers text_to_image native + derived img2img/inpaint (proven for sdxl).
	gotOps := map[string]bool{}
	for _, eo := range p.EffectiveOps {
		if eo.Offerable() {
			gotOps[eo.Op] = true
		}
	}
	for _, want := range []string{"text_to_image", "image_to_image", "inpaint"} {
		if !gotOps[want] {
			t.Errorf("expected offerable op %q in %v", want, gotOps)
		}
	}
}

func TestBuildImportEntry_SingleFileURL(t *testing.T) {
	meta := hfmeta.Metadata{
		Source: "https://example.test/checkpoints/spicy-realism-xl.safetensors",
		Layout: hfmeta.LayoutSingleFile,
		Files:  []hfmeta.FileInfo{{Path: "spicy-realism-xl.safetensors"}},
	}
	// Licensing never blocks a local import: unattested is registered with a
	// conditional label that keeps hosted use refused; attesting lifts it.
	unattested, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-spicy", Name: "Spicy", Architecture: ArchSDXL})
	if err != nil {
		t.Fatalf("unattested import must register for local use: %v", err)
	}
	if unattested.CapabilityLabels.CommercialUse != CommercialUseConditional || unattested.CapabilityLabels.License != "unverified" {
		t.Fatalf("unattested labels = %+v", unattested.CapabilityLabels)
	}
	if !unattested.Enabled {
		t.Fatal("an import with a runnable backend is selectable as soon as it installs")
	}
	if err := RequireLocalUsePermitted(context.Background(), nil, unattested); err != nil {
		t.Fatalf("local use of an unattested import: %v", err)
	}
	if err := RequireUsePermitted(context.Background(), nil, unattested, RightsUseMonetizedUse); err == nil {
		t.Fatal("monetized use of an unattested import must stay refused")
	}
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-spicy", Name: "Spicy", Architecture: ArchSDXL, AttestCommercialRights: true})
	if err != nil {
		t.Fatalf("attested build: %v", err)
	}
	if entry.Backend != "stable-diffusion.cpp" {
		t.Fatalf("backend = %q", entry.Backend)
	}
	if len(entry.Source.Assets) != 1 || entry.Source.Assets[0].URL != meta.Source {
		t.Fatalf("assets = %+v", entry.Source.Assets)
	}
	if entry.Source.Assets[0].Kind != ArtifactSafetensors {
		t.Fatalf("asset kind = %q", entry.Source.Assets[0].Kind)
	}
	if entry.CapabilityLabels.CommercialUse != CommercialUseYes {
		t.Fatalf("attested should be commercial-clean, got %q", entry.CapabilityLabels.CommercialUse)
	}
}

func TestBuildImportEntry_SingleFileHFRepoResolveURL(t *testing.T) {
	meta := hfmeta.Metadata{
		Source:   "Lykon/checkpoint",
		RepoID:   "Lykon/checkpoint",
		Revision: "abc123",
		Layout:   hfmeta.LayoutSingleFile,
		Files:    []hfmeta.FileInfo{{Path: "README.md"}, {Path: "dreamshaper.safetensors"}},
	}
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-dreamshaper", Architecture: ArchSD15, AttestCommercialRights: true})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	want := "https://huggingface.co/Lykon/checkpoint/resolve/abc123/dreamshaper.safetensors"
	if len(entry.Source.Assets) != 1 || entry.Source.Assets[0].URL != want {
		t.Fatalf("asset url = %+v, want %s", entry.Source.Assets, want)
	}
}

func TestBuildImportEntry_AttestationFlipsCommercial(t *testing.T) {
	meta := hfmeta.Metadata{Source: "Org/Repo", RepoID: "Org/Repo", Revision: "r1", Layout: hfmeta.LayoutDiffusersRepo, PipelineClass: "StableDiffusionXLPipeline", Files: []hfmeta.FileInfo{{Path: "model_index.json"}}}
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "x", AttestCommercialRights: true})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if entry.CapabilityLabels.CommercialUse != CommercialUseYes {
		t.Fatalf("attested should be yes, got %q", entry.CapabilityLabels.CommercialUse)
	}
}

// TestBuildImportEntry_RunnableAndMergeable proves the import-flow gap fix: an
// imported entry carries runnable hardware bounds (so it passes the registry's
// "cpu_capable or gpu_required" invariant) and the seed registry accepts it via
// WithCustom — i.e. an imported model is actually resolvable for generation, not
// merely registered.
func TestBuildImportEntry_RunnableAndMergeable(t *testing.T) {
	for _, tc := range []struct {
		arch    Architecture
		gpuOnly bool
	}{
		{ArchSD15, false},
		{ArchSDXL, true},
		{ArchFlux, true},
	} {
		meta := hfmeta.Metadata{Source: "Org/Repo", RepoID: "Org/Repo", Revision: "r1", Layout: hfmeta.LayoutDiffusersRepo, Files: []hfmeta.FileInfo{{Path: "model_index.json"}}}
		entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-" + string(tc.arch), Architecture: tc.arch, AttestCommercialRights: true})
		if err != nil {
			t.Fatalf("%s build: %v", tc.arch, err)
		}
		if !entry.Hardware.CPUCapable && !entry.Hardware.GPURequired {
			t.Fatalf("%s: imported entry is neither cpu_capable nor gpu_required (un-runnable)", tc.arch)
		}
		if tc.gpuOnly && !entry.Hardware.GPURequired {
			t.Fatalf("%s: large architecture should be GPU-required", tc.arch)
		}
		reg, err := Load()
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		merged, err := reg.WithCustom([]Model{entry})
		if err != nil {
			t.Fatalf("%s: WithCustom rejected a valid imported entry: %v", tc.arch, err)
		}
		if _, ok := merged.ByID(entry.ID); !ok {
			t.Fatalf("%s: imported entry not resolvable by id after merge", tc.arch)
		}
		if got := merged.ForOperation("text_to_image"); !containsModelID(got, entry.ID) {
			t.Fatalf("%s: imported entry not joined to text_to_image candidates", tc.arch)
		}
	}
}

func containsModelID(ms []Model, id string) bool {
	for _, m := range ms {
		if m.ID == id {
			return true
		}
	}
	return false
}

func TestBuildImportEntry_Errors(t *testing.T) {
	repo := hfmeta.Metadata{Source: "Org/Repo", RepoID: "Org/Repo", Layout: hfmeta.LayoutDiffusersRepo, Files: []hfmeta.FileInfo{{Path: "model_index.json"}}}
	if _, err := BuildImportEntry(repo, ImportConfirm{}); err == nil {
		t.Error("missing id must error")
	}
	// Unknown architecture (no pipeline class, no tags) + no confirm → error.
	if _, err := BuildImportEntry(repo, ImportConfirm{ID: "x"}); err == nil {
		t.Error("unresolved architecture must error")
	}
}

func diffusersRepo(pipelineClass string, files ...string) hfmeta.Metadata {
	meta := hfmeta.Metadata{
		Source: "acme/some-model", RepoID: "acme/some-model", Revision: "abc123",
		Layout: hfmeta.LayoutDiffusersRepo, PipelineClass: pipelineClass, License: "apache-2.0",
	}
	for _, f := range files {
		meta.Files = append(meta.Files, hfmeta.FileInfo{Path: f, Size: 1 << 20})
	}
	return meta
}

// TestImportOfAKnownFamilyIsRunnableWithNoOperatorInput is the shortest path to a
// new HF diffusers model: the repo's pipeline class finds its adapter, the adapter
// supplies the operations and runtime, the duplicate root checkpoint is not
// fetched, and the entry is selectable as soon as it installs.
func TestImportOfAKnownFamilyIsRunnableWithNoOperatorInput(t *testing.T) {
	meta := diffusersRepo("Flux2KleinPipeline",
		"model_index.json", "flux2-klein-9b.safetensors", "transformer/diffusion_pytorch_model.safetensors", "text_encoder/model.safetensors", "README.md")
	p := ProposeImport(meta)
	if p.Family != "flux-2-klein" || len(p.Needs) != 0 {
		t.Fatalf("family = %q, needs = %v", p.Family, p.Needs)
	}
	e := p.Entry
	if e.Runtime.Family != "flux-2-klein" || e.Runtime.PipelineClass != "Flux2KleinPipeline" || e.Runtime.MinRuntime != "diffusers>=0.37.0" {
		t.Fatalf("runtime = %+v", e.Runtime)
	}
	if got := strings.Join(e.Operations, ","); got != "edit_instruct,image_to_image,text_to_image" {
		t.Fatalf("operations = %s, want the family's", got)
	}
	if !e.Enabled || e.Backend != BackendDiffusers {
		t.Fatalf("enabled=%t backend=%q", e.Enabled, e.Backend)
	}
	want := []string{"model_index.json", "transformer/diffusion_pytorch_model.safetensors", "text_encoder/model.safetensors", "README.md"}
	if got := e.Source.Repo.AllowPatterns; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("allow patterns = %v, want the component files without the root checkpoint", got)
	}

	// Operator-listed operations win over the family's.
	built, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-flux", Operations: []string{"text_to_image"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(built.Operations, ",") != "text_to_image" || built.Runtime.Family != "flux-2-klein" {
		t.Fatalf("operations = %v, runtime = %+v", built.Operations, built.Runtime)
	}
}

func TestImportNamesWhatIsMissingInPlainWords(t *testing.T) {
	t.Run("a family whose adapter is not proven stays disabled and says why", func(t *testing.T) {
		p := ProposeImport(diffusersRepo("LongCatImageEditPipeline", "model_index.json", "transformer/a.safetensors"))
		if p.Entry.Enabled || len(p.Needs) != 1 || !strings.Contains(p.Needs[0], "not proven runnable yet") {
			t.Fatalf("enabled=%t needs=%v", p.Entry.Enabled, p.Needs)
		}
	})
	t.Run("a pipeline class with no adapter says where the adapter goes", func(t *testing.T) {
		p := ProposeImport(diffusersRepo("FluxNextPipeline", "model_index.json", "transformer/a.safetensors"))
		if p.Family != "" || len(p.Needs) != 1 {
			t.Fatalf("family=%q needs=%v", p.Family, p.Needs)
		}
		for _, want := range []string{`"FluxNextPipeline"`, "families.go", "_diffusers.py"} {
			if !strings.Contains(p.Needs[0], want) {
				t.Errorf("need %q lacks %q", p.Needs[0], want)
			}
		}
		// Registering it would only produce a model whose first job fails.
		if _, err := BuildImportEntry(diffusersRepo("FluxNextPipeline", "model_index.json", "transformer/a.safetensors"), ImportConfirm{ID: "imported-next"}); err == nil || !strings.Contains(err.Error(), "families.go") {
			t.Fatalf("an import with no adapter must be refused with the missing piece named, got %v", err)
		}
	})
	t.Run("an unrecognisable source asks for the architecture", func(t *testing.T) {
		p := ProposeImport(diffusersRepo("MysteryPipeline", "model_index.json"))
		if len(p.Needs) == 0 || !strings.Contains(p.Needs[0], "--architecture") || !strings.Contains(p.Needs[0], "sdxl") {
			t.Fatalf("needs = %v", p.Needs)
		}
	})
	t.Run("a base SDXL repo needs no adapter", func(t *testing.T) {
		p := ProposeImport(diffusersRepo("StableDiffusionXLPipeline", "model_index.json", "unet/a.safetensors"))
		if p.Family != "" || len(p.Needs) != 0 || !p.Entry.Enabled {
			t.Fatalf("family=%q needs=%v enabled=%t", p.Family, p.Needs, p.Entry.Enabled)
		}
	})
}

// TestCompleteCustomMakesTheShortestRegistrationComplete covers adding an ONNX
// model by id, operation, backend and URL: everything else is completed and
// recorded, nothing is refused.
func TestCompleteCustomMakesTheShortestRegistrationComplete(t *testing.T) {
	m := CompleteCustom(Model{
		ID: "my-depth", Operations: []string{"depth_map"}, Backend: "onnxruntime",
		Source: Source{DownloadURL: "https://example.test/files/depth_v3.onnx?download=1"},
	})
	if m.Name != "my-depth" || m.Tier != TierNiceToHave || !m.Hardware.CPUCapable {
		t.Fatalf("name/tier/hardware = %q/%q/%+v", m.Name, m.Tier, m.Hardware)
	}
	if len(m.Source.Assets) != 1 || m.Source.Assets[0].Filename != "depth_v3.onnx" || m.Source.Assets[0].Kind != ArtifactONNX {
		t.Fatalf("assets = %+v", m.Source.Assets)
	}
	l := m.CapabilityLabels
	if l.License != "unverified" || l.CommercialUse != CommercialUseConditional || l.Provenance != ProvenanceUserImported || l.CommercialUseNotes == "" {
		t.Fatalf("labels = %+v", l)
	}
	// What the operator stated is kept.
	kept := CompleteCustom(Model{
		ID: "mine", Name: "Mine", Tier: TierDefault, Hardware: Hardware{GPURequired: true},
		CapabilityLabels: CapabilityLabels{License: "MIT", CommercialUse: CommercialUseYes},
		Source:           Source{LocalPath: "/x/y.onnx"},
	})
	if kept.Name != "Mine" || kept.Tier != TierDefault || !kept.Hardware.GPURequired || kept.CapabilityLabels.License != "MIT" || kept.CapabilityLabels.CommercialUse != CommercialUseYes || len(kept.Source.Assets) != 0 {
		t.Fatalf("stated facts were overwritten: %+v", kept)
	}
	if _, ok := assetFromURL("ftp://x/y.onnx"); ok {
		t.Fatal("only http(s) URLs name a fetchable asset")
	}
}

// TestImportFetchesOneCopyOfTheWeights proves a repo that lists a root checkpoint,
// alternate formats and both precisions imports as one component set at one
// precision, with the size the user will actually download.
func TestImportFetchesOneCopyOfTheWeights(t *testing.T) {
	mb := int64(1 << 20)
	repo := func(files map[string]int64) hfmeta.Metadata {
		meta := diffusersRepo("StableDiffusionXLPipeline")
		var order []string
		for p := range files {
			order = append(order, p)
		}
		sort.Strings(order)
		for _, p := range order {
			meta.Files = append(meta.Files, hfmeta.FileInfo{Path: p, Size: files[p] * mb})
		}
		return meta
	}
	t.Run("fp16 everywhere keeps only fp16 safetensors", func(t *testing.T) {
		meta := repo(map[string]int64{
			"model_index.json": 0, "sdxl.safetensors": 6000, "sdxl.ckpt": 6000,
			"unet/config.json": 0, "unet/diffusion_pytorch_model.safetensors": 10000, "unet/diffusion_pytorch_model.fp16.safetensors": 5000, "unet/diffusion_pytorch_model.bin": 10000,
			"vae/diffusion_pytorch_model.safetensors": 300, "vae/diffusion_pytorch_model.fp16.safetensors": 150, "vae/diffusion_pytorch_model.onnx": 300,
		})
		p := planRepoFetch(meta)
		if p.Variant != "fp16" || p.Bytes != 5150*mb {
			t.Fatalf("variant=%q bytes=%d MB", p.Variant, p.Bytes/mb)
		}
		want := "model_index.json,unet/config.json,unet/diffusion_pytorch_model.fp16.safetensors,vae/diffusion_pytorch_model.fp16.safetensors"
		if got := strings.Join(p.Files, ","); got != want {
			t.Fatalf("files = %s", got)
		}
		e := ProposeImport(meta).Entry
		if e.Runtime.Variant != "fp16" || e.SizeMBApprox != 5150 {
			t.Fatalf("entry variant=%q size=%d MB", e.Runtime.Variant, e.SizeMBApprox)
		}
	})
	t.Run("fp16 missing from one folder falls back to full precision", func(t *testing.T) {
		p := planRepoFetch(repo(map[string]int64{
			"unet/diffusion_pytorch_model.safetensors": 10000, "unet/diffusion_pytorch_model.fp16.safetensors": 5000,
			"text_encoder/model.safetensors": 500,
		}))
		if p.Variant != "" || p.Bytes != 10500*mb {
			t.Fatalf("variant=%q bytes=%d MB", p.Variant, p.Bytes/mb)
		}
	})
	t.Run("a bin-only repo keeps its bin weights", func(t *testing.T) {
		p := planRepoFetch(repo(map[string]int64{"unet/diffusion_pytorch_model.bin": 3000, "model_index.json": 0}))
		if len(p.Files) != 2 {
			t.Fatalf("files = %v", p.Files)
		}
	})
	t.Run("a flat listing fetches the whole repo", func(t *testing.T) {
		if p := planRepoFetch(repo(map[string]int64{"model.safetensors": 100})); p.Files != nil {
			t.Fatalf("files = %v", p.Files)
		}
	})
}

// TestImportOfAnUnprovenFamilyStaysOffAndSaysWhy: an adapter that is registered
// but not proven keeps the import disabled, and the entry carries the family's
// own reason so its verdict explains it.
func TestImportOfAnUnprovenFamilyStaysOffAndSaysWhy(t *testing.T) {
	e := ProposeImport(diffusersRepo("LongCatImageEditPipeline")).Entry
	if e.Enabled {
		t.Fatalf("an unproven family import must start disabled: %+v", e)
	}
	fam, _ := DiffusersFamilyByName("longcat-image-edit")
	if e.Disabled == nil || e.Disabled.Kind != DisabledRuntime || e.Disabled.Reason != fam.Pending {
		t.Fatalf("disabled = %+v, want runtime with the family's pending reason", e.Disabled)
	}
	if ok := ProposeImport(diffusersRepo("Flux2KleinPipeline")).Entry; !ok.Enabled || ok.Disabled != nil {
		t.Fatalf("a proven family import must be enabled with no hold: %+v", ok)
	}
}
