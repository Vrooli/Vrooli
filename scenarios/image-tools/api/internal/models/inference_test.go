package models

import (
	"testing"

	"image-tools/internal/hfmeta"
)

func TestInferArchitecture(t *testing.T) {
	cases := []struct {
		name     string
		meta     hfmeta.Metadata
		wantArch Architecture
		wantConf Confidence
	}{
		{
			name:     "sdxl from pipeline class",
			meta:     hfmeta.Metadata{PipelineClass: "StableDiffusionXLPipeline", Tags: []string{"stable-diffusion"}},
			wantArch: ArchSDXL, wantConf: ConfidenceHigh,
		},
		{
			name:     "sd15 from pipeline class",
			meta:     hfmeta.Metadata{PipelineClass: "StableDiffusionPipeline"},
			wantArch: ArchSD15, wantConf: ConfidenceHigh,
		},
		{
			name:     "instruct-pix2pix class beats generic SD substring",
			meta:     hfmeta.Metadata{PipelineClass: "StableDiffusionInstructPix2PixPipeline"},
			wantArch: ArchInstructPix2Pix, wantConf: ConfidenceHigh,
		},
		{
			name:     "qwen edit from class",
			meta:     hfmeta.Metadata{PipelineClass: "QwenImageEditPlusPipeline"},
			wantArch: ArchQwenImageEdit, wantConf: ConfidenceHigh,
		},
		{
			name:     "flux from class prefix",
			meta:     hfmeta.Metadata{PipelineClass: "FluxPipeline"},
			wantArch: ArchFlux, wantConf: ConfidenceHigh,
		},
		{
			name:     "sdxl from tag when no class",
			meta:     hfmeta.Metadata{Tags: []string{"text-to-image", "stable-diffusion-xl"}},
			wantArch: ArchSDXL, wantConf: ConfidenceMedium,
		},
		{
			name:     "sdxl tag wins over generic stable-diffusion tag",
			meta:     hfmeta.Metadata{Tags: []string{"stable-diffusion", "sdxl"}},
			wantArch: ArchSDXL, wantConf: ConfidenceMedium,
		},
		{
			name:     "sd15 from tag",
			meta:     hfmeta.Metadata{Tags: []string{"stable-diffusion", "text-to-image"}},
			wantArch: ArchSD15, wantConf: ConfidenceMedium,
		},
		{
			name:     "lineage low confidence",
			meta:     hfmeta.Metadata{BaseModel: "stabilityai/stable-diffusion-xl-base-1.0"},
			wantArch: ArchSDXL, wantConf: ConfidenceLow,
		},
		{
			name:     "nothing recognizable",
			meta:     hfmeta.Metadata{Tags: []string{"some-random-tag"}},
			wantArch: ArchNone, wantConf: ConfidenceNone,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arch, conf, evidence := InferArchitecture(tc.meta)
			if arch != tc.wantArch {
				t.Errorf("arch = %q, want %q", arch, tc.wantArch)
			}
			if conf != tc.wantConf {
				t.Errorf("confidence = %q, want %q", conf, tc.wantConf)
			}
			if evidence == "" {
				t.Error("evidence must never be empty")
			}
		})
	}
}
