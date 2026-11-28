package adapters

import (
	"testing"

	"image-tools/internal/hfmeta"
	"image-tools/internal/models"
	"image-tools/internal/safety"
)

func TestInferKind(t *testing.T) {
	cases := []struct {
		name string
		meta hfmeta.Metadata
		want Kind
	}{
		{"lora tag", hfmeta.Metadata{Tags: []string{"stable-diffusion", "lora"}}, KindLoRA},
		{"controlnet tag", hfmeta.Metadata{Tags: []string{"controlnet"}}, KindControlNet},
		{"ip-adapter tag", hfmeta.Metadata{Tags: []string{"ip_adapter"}}, KindIPAdapter},
		{"ip-adapter filename", hfmeta.Metadata{Files: []hfmeta.FileInfo{{Path: "ip-adapter_sd15.safetensors"}}}, KindIPAdapter},
		{"lora filename", hfmeta.Metadata{Files: []hfmeta.FileInfo{{Path: "my_lora.safetensors"}}}, KindLoRA},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := InferKind(tt.meta); got != tt.want {
				t.Fatalf("InferKind=%q want %q", got, tt.want)
			}
		})
	}
	if got, _ := InferKind(hfmeta.Metadata{Tags: []string{"misc"}}); got != "" {
		t.Fatalf("unrecognized source should infer empty kind, got %q", got)
	}
}

func TestBuildImportEntrySingleFileLoRA(t *testing.T) {
	meta := hfmeta.Metadata{
		Source:    "someuser/cool-lora",
		RepoID:    "someuser/cool-lora",
		Revision:  "abc123",
		Layout:    hfmeta.LayoutSingleFile,
		Tags:      []string{"lora"},
		BaseModel: "stabilityai/stable-diffusion-xl-base-1.0",
		Files:     []hfmeta.FileInfo{{Path: "pytorch_lora_weights.safetensors", Size: 200 << 20}},
	}
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-cool-lora", Architecture: models.ArchSDXL})
	if err != nil {
		t.Fatalf("build entry: %v", err)
	}
	if entry.Kind != KindLoRA {
		t.Fatalf("kind=%q want lora", entry.Kind)
	}
	if entry.Architecture != models.ArchSDXL {
		t.Fatalf("arch=%q want sdxl", entry.Architecture)
	}
	if entry.Ready {
		t.Fatal("imported adapter must not be Ready")
	}
	if entry.CapabilityLabels.Provenance != models.ProvenanceUserImported {
		t.Fatalf("expected user-imported provenance, got %q", entry.CapabilityLabels.Provenance)
	}
	if entry.CapabilityLabels.CommercialUse != models.CommercialUseConditional {
		t.Fatalf("unattested import should be conditional, got %q", entry.CapabilityLabels.CommercialUse)
	}
	if len(entry.Source.Assets) != 1 {
		t.Fatalf("expected one asset, got %+v", entry.Source)
	}
	// The assembled entry must pass structural validation.
	if err := validateAdapter(entry); err != nil {
		t.Fatalf("assembled entry invalid: %v", err)
	}
}

func TestBuildImportEntryIPAdapterIsHighWeightAndAttestable(t *testing.T) {
	meta := hfmeta.Metadata{
		Source: "h94/IP-Adapter", RepoID: "h94/IP-Adapter", Revision: "deadbeef",
		Layout: hfmeta.LayoutSingleFile, Tags: []string{"ip-adapter"},
		BaseModel: "runwayml/stable-diffusion-v1-5",
		Files:     []hfmeta.FileInfo{{Path: "ip-adapter_sd15.safetensors", Size: 45 << 20}},
	}
	entry, err := BuildImportEntry(meta, ImportConfirm{ID: "imported-ipa", Architecture: models.ArchSD15, AttestCommercialRights: true})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if entry.Kind != KindIPAdapter {
		t.Fatalf("kind=%q want ip-adapter", entry.Kind)
	}
	if entry.Weight != safety.WeightHigh {
		t.Fatalf("ip-adapter weight=%q want high", entry.Weight)
	}
	if entry.CapabilityLabels.CommercialUse != models.CommercialUseYes {
		t.Fatalf("attested import should be commercial yes, got %q", entry.CapabilityLabels.CommercialUse)
	}
}

func TestBuildImportEntryFailsClosed(t *testing.T) {
	// No kind inferable and none confirmed.
	if _, err := BuildImportEntry(hfmeta.Metadata{Tags: []string{"misc"}, Layout: hfmeta.LayoutSingleFile}, ImportConfirm{ID: "x", Architecture: models.ArchSD15}); err == nil {
		t.Fatal("expected failure when kind is unresolved")
	}
	// Kind ok but architecture unresolved.
	if _, err := BuildImportEntry(hfmeta.Metadata{Tags: []string{"lora"}, Layout: hfmeta.LayoutSingleFile}, ImportConfirm{ID: "x"}); err == nil {
		t.Fatal("expected failure when architecture is unresolved")
	}
	// Missing id.
	if _, err := BuildImportEntry(hfmeta.Metadata{Tags: []string{"lora"}}, ImportConfirm{Architecture: models.ArchSD15}); err == nil {
		t.Fatal("expected failure when id is missing")
	}
}
