package photographic

import (
	"encoding/json"
	"strings"
	"testing"

	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestAuthoredCleanPlateRequiresExactEvidenceWithoutInventedModelIdentity(t *testing.T) {
	manifest := &preparationv1.PreparedPhotograph{
		Source: &preparationv1.SourceRaster{
			Original: &preparationv1.Artifact{Sha256: strings.Repeat("a", 64)},
			Oriented: &preparationv1.Artifact{Sha256: strings.Repeat("b", 64)},
		},
		Background: &preparationv1.Background{
			RemovalMask:      &preparationv1.Artifact{Sha256: strings.Repeat("c", 64)},
			CleanPlate:       &preparationv1.Artifact{Sha256: strings.Repeat("d", 64)},
			RepairProvenance: &preparationv1.Artifact{Path: "repair.json"},
		},
	}
	prompt := "Remove the phone and its shadow.\nPreserve camera framing and table texture."
	base := repairDoc{
		Schema: "vrooli.image-tools.authored-clean-plate/v1", SourceSHA256: manifest.Source.Oriented.Sha256,
		MaskSHA256: manifest.Background.RemovalMask.Sha256, OutputSHA256: manifest.Background.CleanPlate.Sha256,
		Tool: "image_gen.imagegen", ToolOutputReference: "generated-images:retained-tool-output",
		Prompt: prompt, PromptSHA256: digest([]byte(prompt)), PixelScope: "full_frame_generated",
	}
	check := func(p repairDoc) error {
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		return validateRepairProvenance(manifest, map[string][]byte{"repair.json": raw})
	}
	if err := check(base); err != nil {
		t.Fatalf("explicit authored repair must not require an invented model/checkpoint: %v", err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*repairDoc)
	}{
		{"original-instead-of-oriented-source", func(p *repairDoc) { p.SourceSHA256 = manifest.Source.Original.Sha256 }},
		{"different-mask", func(p *repairDoc) { p.MaskSHA256 = strings.Repeat("e", 64) }},
		{"different-output", func(p *repairDoc) { p.OutputSHA256 = strings.Repeat("e", 64) }},
		{"changed-prompt", func(p *repairDoc) { p.Prompt += " Added direction." }},
		{"missing-prompt", func(p *repairDoc) { p.Prompt = ""; p.PromptSHA256 = digest(nil) }},
		{"oversized-prompt", func(p *repairDoc) {
			p.Prompt = strings.Repeat("x", (16<<10)+1)
			p.PromptSHA256 = digest([]byte(p.Prompt))
		}},
		{"missing-tool", func(p *repairDoc) { p.Tool = "" }},
		{"missing-tool-output", func(p *repairDoc) { p.ToolOutputReference = "" }},
		{"false-outside-mask-guarantee", func(p *repairDoc) { p.PixelScope = "outside_mask_exact" }},
		{"local-model-identity", func(p *repairDoc) { p.ModelID = "big-lama"; p.ModelRevision = "invented-revision" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			if err := check(p); err == nil {
				t.Fatal("contradictory or incomplete authored repair evidence was accepted")
			}
		})
	}
}

func TestConvertRetainsAuthoredRepairAndExactPlate(t *testing.T) {
	b, err := openBundle(photographicFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	m := b.manifest
	prompt := "Remove the photographed device and restore the table."
	proof, err := json.Marshal(repairDoc{
		Schema: "vrooli.image-tools.authored-clean-plate/v1", SourceSHA256: m.Source.Oriented.Sha256,
		MaskSHA256: m.Background.RemovalMask.Sha256, OutputSHA256: m.Background.CleanPlate.Sha256,
		Tool: "image_gen.imagegen", ToolOutputReference: "generated-images:retained-fixture",
		Prompt: prompt, PromptSHA256: digest([]byte(prompt)), PixelScope: "full_frame_generated",
	})
	if err != nil {
		t.Fatal(err)
	}
	b.files[m.Background.RepairProvenance.Path] = proof
	m.Background.RepairProvenance.Sha256 = digest(proof)
	m.Background.RepairProvenance.SizeBytes = int64(len(proof))
	b.files["manifest.json"], err = protojson.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	converted, err := Convert(zipBytes(t, b.files))
	if err != nil {
		t.Fatalf("valid authored clean plate should use the normal protected-bundle conversion: %v", err)
	}
	if converted.Report.RepairProvenanceSHA256 != digest(proof) || digest(converted.Texture) != m.Background.CleanPlate.Sha256 || !converted.Report.DeviceMaskExcluded {
		t.Fatal("authored repair identity, exact plate or removed-device reconstruction was lost")
	}
}
