package models

import "sort"

// Architecture is a model's weight lineage/topology — the fact that determines
// which *techniques* are derivable on its weights. It is orthogonal to
// Runtime.Family (the diffusers adapter that LOADS the weights): architecture is
// the weights' lineage, family is the loader. A base SDXL checkpoint and an SDXL
// inpainting checkpoint share architecture `sdxl` but differ in declared ops; the
// architecture is what lets a text-to-image-only checkpoint *derive* img2img /
// inpaint / outpaint / edit-via-img2img with zero per-model code.
//
// This is the Go SSOT; internal/sidecar/py/image_tools_sidecar/_diffusers.py
// mirrors the architecture→technique table (ARCHITECTURES) and the parity test
// TestArchitectureTechniquesMirrorPython asserts lockstep, exactly as families
// are mirrored (decision 117 pattern).
type Architecture string

const (
	// ArchNone is for models whose weights have no derivable-technique lineage:
	// builtin/computed/library models, manual landing-page stubs, and the
	// single-purpose ONNX/ncnn analysis & enhancement weights. They derive nothing.
	ArchNone Architecture = "none"
	// ArchSD15 is the Stable Diffusion 1.5 base architecture.
	ArchSD15 Architecture = "sd15"
	// ArchSDXL is the Stable Diffusion XL 1.0 base architecture.
	ArchSDXL Architecture = "sdxl"
	// ArchFlux is the FLUX.1/FLUX.2 architecture family.
	ArchFlux Architecture = "flux"
	// ArchInstructPix2Pix is the InstructPix2Pix instruction-edit architecture.
	ArchInstructPix2Pix Architecture = "instruct-pix2pix"
	// ArchQwenImageEdit is the Qwen-Image-Edit instruction-edit architecture.
	ArchQwenImageEdit Architecture = "qwen-image-edit"
	// ArchLongCatImageEdit is the LongCat-Image-Edit instruction-edit architecture.
	ArchLongCatImageEdit Architecture = "longcat-image-edit"
)

// architectures is the closed enum the registry validates against.
var architectures = map[Architecture]struct{}{
	ArchNone:             {},
	ArchSD15:             {},
	ArchSDXL:             {},
	ArchFlux:             {},
	ArchInstructPix2Pix:  {},
	ArchQwenImageEdit:    {},
	ArchLongCatImageEdit: {},
}

func (a Architecture) valid() bool {
	_, ok := architectures[a]
	return ok
}

// DerivedTechnique is one architecture-derivable capability: an operation a model
// of this architecture can run through a named technique, the quality caveat that
// derivation carries, and a Ready gate. Ready=false means the technique is
// declared but NOT yet proven runnable on this architecture (an honest
// `derived_pipeline_unproven` state, never a faked capability — decision 120
// extended to derived ops); the attended GPU acceptance run (plan Phase 7) is
// what flips a derivation Ready once it produces real output end-to-end.
type DerivedTechnique struct {
	// Op is the operation this derivation yields (a vocabulary member).
	Op string
	// Technique names the technique (internal/technique) that runs it.
	Technique string
	// Caveat is the quality note surfaced to the user for this derived op.
	Caveat string
	// Ready reports the derivation is proven runnable on this architecture.
	Ready bool
}

// architectureTechniques is the canonical architecture→technique derivation
// table (Go SSOT; mirrored in _diffusers.py ARCHITECTURES). Base txt2img
// checkpoints (sd15/sdxl/flux) derive the image-conditioned ops; the
// instruction-edit architectures are already specialised and derive nothing
// extra. A row is Ready=false until an attended GPU run proves that pipeline on
// that architecture produces real output; flipping it is a reviewed change here +
// in the _diffusers.py mirror (parity test) + the golden matrix. img2img (sd.cpp
// -i) and inpaint (diffusers Stable Diffusion[XL]InpaintPipeline) are proven for
// sd15 AND sdxl; outpaint (needs canvas-expansion, delivered via the
// outpaint-extend Look) and edit-via-img2img stay unproven. Unproven rows are
// wired + inspectable but never offered for selection (no vaporware).
var architectureTechniques = map[Architecture][]DerivedTechnique{
	ArchSD15: {
		{Op: "image_to_image", Technique: "sd-img2img", Ready: true, Caveat: "derived: img2img on a base checkpoint is a first-class use; results follow the checkpoint's style"},
		{Op: "inpaint", Technique: "diffusers-inpaint", Ready: true, Caveat: "derived: a base checkpoint inpaints via the standard pipeline; a dedicated *-inpainting model blends masked edges more cleanly"},
		{Op: "outpaint", Technique: "diffusers-outpaint", Caveat: "derived: outpaint = expand-canvas + inpaint on a base checkpoint; new borders may need a refine pass"},
		{Op: "edit_instruct", Technique: "edit-via-img2img", Caveat: "derived: probabilistic edit via low-strength img2img; not identity-preserving like a true instruction-edit model"},
	},
	ArchSDXL: {
		{Op: "image_to_image", Technique: "sd-img2img", Ready: true, Caveat: "derived: img2img on a base checkpoint is a first-class use; results follow the checkpoint's style"},
		{Op: "inpaint", Technique: "diffusers-inpaint", Ready: true, Caveat: "derived: a base checkpoint inpaints via the standard pipeline; a dedicated *-inpainting model blends masked edges more cleanly"},
		{Op: "outpaint", Technique: "diffusers-outpaint", Caveat: "derived: outpaint = expand-canvas + inpaint on a base checkpoint; new borders may need a refine pass"},
		{Op: "edit_instruct", Technique: "edit-via-img2img", Caveat: "derived: probabilistic edit via low-strength img2img; not identity-preserving like a true instruction-edit model"},
	},
	ArchFlux: {
		{Op: "image_to_image", Technique: "sd-img2img", Caveat: "derived: img2img on a base checkpoint is a first-class use; results follow the checkpoint's style"},
		{Op: "edit_instruct", Technique: "edit-via-img2img", Caveat: "derived: probabilistic edit via low-strength img2img; not identity-preserving like a true instruction-edit model"},
	},
}

// DerivableTechniques returns the derivation rows for an architecture, sorted by
// op for determinism. Unknown / ArchNone architectures derive nothing.
func DerivableTechniques(arch Architecture) []DerivedTechnique {
	rows := architectureTechniques[arch]
	out := make([]DerivedTechnique, len(rows))
	copy(out, rows)
	sort.Slice(out, func(i, j int) bool { return out[i].Op < out[j].Op })
	return out
}

// EffectiveOp is one entry in a model's derived effective op set: the operation,
// whether the model serves it natively (declared) or via a derived technique, the
// technique name, the quality caveat, and whether it is offerable (native, or
// derived AND proven runnable).
type EffectiveOp struct {
	Op        string
	Support   string // "native" | "derived"
	Technique string // technique name (empty for native ops, which the selector resolves)
	Caveat    string
	Ready     bool // derived-and-proven; always true for native
}

// Offerable reports whether this op can actually be selected/run today: native
// ops always, derived ops only once their technique is proven (Ready). A derived
// op that is not yet Ready is reported (for honest doctor/picker surfacing) but
// is never offered for execution.
func (e EffectiveOp) Offerable() bool { return e.Support == "native" || e.Ready }

// EffectiveOps returns the model's full effective op set:
//
//	declaredOps (native)  ∪  derivable(architecture) (derived, each with a caveat)
//
// Native ops win a collision (a model that declares image_to_image keeps it
// native even if its architecture would also derive it). The result is
// deterministic (native ops in declared order, then derived ops op-sorted).
func (m Model) EffectiveOps() []EffectiveOp {
	native := make(map[string]struct{}, len(m.Operations))
	out := make([]EffectiveOp, 0, len(m.Operations))
	for _, op := range m.Operations {
		native[op] = struct{}{}
		out = append(out, EffectiveOp{Op: op, Support: "native", Ready: true})
	}
	for _, d := range DerivableTechniques(m.Architecture) {
		if _, isNative := native[d.Op]; isNative {
			continue
		}
		out = append(out, EffectiveOp{
			Op:        d.Op,
			Support:   "derived",
			Technique: d.Technique,
			Caveat:    d.Caveat,
			Ready:     d.Ready,
		})
	}
	return out
}

// DerivesOperation reports whether the model serves op through a PROVEN derived
// technique (it is not declared native, but its architecture derives it and that
// derivation is Ready). Callers use this to attach the derived caveat.
func (m Model) DerivesOperation(op string) (DerivedTechnique, bool) {
	for _, eo := range m.EffectiveOps() {
		if eo.Op == op && eo.Support == "derived" && eo.Ready {
			return DerivedTechnique{Op: eo.Op, Technique: eo.Technique, Caveat: eo.Caveat, Ready: eo.Ready}, true
		}
	}
	return DerivedTechnique{}, false
}
