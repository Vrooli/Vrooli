package models

import (
	"fmt"
	"strings"

	"image-tools/internal/hfmeta"
)

// Architecture inference (plan capability D, Phase 0). Importing a model needs to
// PROPOSE an architecture from its HuggingFace metadata so the user can confirm
// it (decision D1 — inferred, never silently trusted). The mapping lives here, in
// the architecture SSOT, next to the enum + derivation table it feeds: an
// inferred `sdxl` immediately lights up the proven img2img/inpaint derivations.
//
// Inference is layered by how trustworthy the signal is:
//   - the diffusers pipeline class (model_index.json _class_name) is definitive →
//     high confidence;
//   - model-card tags are strong but author-set → medium;
//   - base_model lineage is a weak hint → low;
//   - nothing recognizable → ArchNone at none confidence, and the import wizard
//     REQUIRES an explicit user selection before install (no silent guess).

// Confidence grades how strong the architecture inference signal was. The import
// flow surfaces it; ConfidenceNone forces a manual pick.
type Confidence string

const (
	// ConfidenceNone means nothing recognizable was found (manual pick required).
	ConfidenceNone Confidence = "none"
	// ConfidenceLow is a weak lineage hint (base_model).
	ConfidenceLow Confidence = "low"
	// ConfidenceMedium is an author-set model-card tag.
	ConfidenceMedium Confidence = "medium"
	// ConfidenceHigh is the definitive diffusers pipeline class.
	ConfidenceHigh Confidence = "high"
)

// InferArchitecture proposes an Architecture for an inspected model source,
// returning the confidence grade and a human-readable evidence string. It is pure
// (no I/O) so it is unit-tested against captured metadata fixtures.
func InferArchitecture(meta hfmeta.Metadata) (Architecture, Confidence, string) {
	if pc := strings.TrimSpace(meta.PipelineClass); pc != "" {
		if a, ok := archFromPipelineClass(pc); ok {
			return a, ConfidenceHigh, fmt.Sprintf("pipeline class %q", pc)
		}
	}
	if a, tag, ok := archFromTags(meta.Tags); ok {
		return a, ConfidenceMedium, fmt.Sprintf("model-card tag %q", tag)
	}
	if a, hit, ok := archFromLineage(meta.BaseModel); ok {
		return a, ConfidenceLow, fmt.Sprintf("base_model lineage %q", hit)
	}
	return ArchNone, ConfidenceNone, "no recognizable pipeline class, tag, or base_model lineage"
}

// archFromPipelineClass maps a diffusers _class_name to an architecture. The
// edit-specialised classes are matched before the generic SD ones so an
// InstructPix2Pix/Qwen-Edit checkpoint is not mis-read as a base SD model.
func archFromPipelineClass(pc string) (Architecture, bool) {
	lc := strings.ToLower(pc)
	switch {
	case strings.Contains(lc, "instructpix2pix"):
		return ArchInstructPix2Pix, true
	case strings.Contains(lc, "qwenimage") && strings.Contains(lc, "edit"):
		return ArchQwenImageEdit, true
	case strings.Contains(lc, "longcat"):
		return ArchLongCatImageEdit, true
	case strings.HasPrefix(lc, "flux"):
		return ArchFlux, true
	case strings.Contains(lc, "stablediffusionxl"):
		return ArchSDXL, true
	case strings.Contains(lc, "stablediffusion"):
		return ArchSD15, true
	}
	return ArchNone, false
}

// archFromTags maps model-card tags to an architecture. SDXL is checked before
// SD1.5 because SDXL repos commonly also carry a generic "stable-diffusion" tag.
func archFromTags(tags []string) (Architecture, string, bool) {
	norm := make([]string, len(tags))
	for i, t := range tags {
		norm[i] = strings.ToLower(strings.TrimSpace(t))
	}
	has := func(needles ...string) (string, bool) {
		for _, t := range norm {
			for _, n := range needles {
				if t == n || strings.Contains(t, n) {
					return t, true
				}
			}
		}
		return "", false
	}
	if t, ok := has("stable-diffusion-xl", "sdxl"); ok {
		return ArchSDXL, t, true
	}
	if t, ok := has("instruct-pix2pix", "instructpix2pix"); ok {
		return ArchInstructPix2Pix, t, true
	}
	if t, ok := has("qwen-image-edit", "qwen-image"); ok {
		return ArchQwenImageEdit, t, true
	}
	if t, ok := has("flux"); ok {
		return ArchFlux, t, true
	}
	if t, ok := has("stable-diffusion-1", "sd-1.5", "sd1.5", "runwayml", "stable-diffusion"); ok {
		return ArchSD15, t, true
	}
	return ArchNone, "", false
}

// archFromLineage maps a base_model lineage string to an architecture (weakest
// signal). SDXL before SD1.5 for the same reason as tags.
func archFromLineage(base string) (Architecture, string, bool) {
	lc := strings.ToLower(strings.TrimSpace(base))
	if lc == "" {
		return ArchNone, "", false
	}
	switch {
	case strings.Contains(lc, "xl") || strings.Contains(lc, "sdxl"):
		return ArchSDXL, base, true
	case strings.Contains(lc, "flux"):
		return ArchFlux, base, true
	case strings.Contains(lc, "stable-diffusion") || strings.Contains(lc, "sd-1") || strings.Contains(lc, "sd1"):
		return ArchSD15, base, true
	}
	return ArchNone, "", false
}
