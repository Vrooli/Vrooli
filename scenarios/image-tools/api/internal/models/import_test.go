package models

import (
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
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-spicy", Name: "Spicy", Architecture: ArchSDXL})
	if err != nil {
		t.Fatalf("build: %v", err)
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
	if entry.CapabilityLabels.CommercialUse != CommercialUseConditional {
		t.Fatalf("unattested should be conditional, got %q", entry.CapabilityLabels.CommercialUse)
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
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-dreamshaper", Architecture: ArchSD15})
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
		entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-" + string(tc.arch), Architecture: tc.arch})
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
