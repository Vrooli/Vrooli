package models

import (
	"sort"
	"strings"
)

// DiffusersFamily is one registered diffusers execution adapter: a pipeline
// architecture the generic sidecar runner (internal/sidecar/py/image_tools_sidecar/
// _diffusers.py) knows how to load and call. The registry's Runtime.Family on a
// diffusers model must name one of these — that is the single source of truth the
// catalog doctor checks ("enabled diffusers model ⇒ registered family adapter")
// and the Python adapter table mirrors (asserted by the conformance test).
type DiffusersFamily struct {
	// Name is the stable family key referenced by Runtime.Family.
	Name string
	// PipelineClass is the diffusers pipeline class the adapter loads/expects.
	PipelineClass string
	// Ready reports whether the adapter is proven runnable. A family declared but
	// not yet proven (its diffusers pipeline is bleeding-edge / unpinned) is
	// registered with Ready=false so its models stay honestly disabled until an
	// attended run flips them — never a faked-green provider (no-vaporware tenet).
	Ready bool
	// Pending explains, for a not-Ready family, what blocks proving it.
	Pending string
	// Operations is what a model of this family serves by default: the ops its
	// adapter can run (an instruction-edit family serves edit_instruct; a unified
	// family also generates and transforms). `models import` reads it so an
	// imported repo of a known family offers its ops without the operator listing
	// them. text_to_image appears here exactly when the Python adapter generates
	// (asserted by TestDiffusersFamilyAdaptersMirrorPython).
	Operations []string
	// MinRuntime is the diffusers constraint the family needs (for example
	// "diffusers>=0.37.0"); imports copy it so provisioning knows what to pin.
	MinRuntime string
}

// diffusersFamilies is the canonical adapter registry. Keep in lockstep with the
// Python FAMILIES table in _diffusers.py (TestDiffusersFamilyAdaptersMirrorPython
// asserts parity). Adding a model of an existing family is a registry row (or
// `models import <hf-repo>`, which finds the family from the repo's pipeline
// class); a new architecture is one row here + one Python adapter + (once
// proven) Ready=true.
var diffusersFamilies = map[string]DiffusersFamily{
	"instruct-pix2pix": {
		Name:          "instruct-pix2pix",
		PipelineClass: "StableDiffusionInstructPix2PixPipeline",
		Ready:         true,
		Operations:    []string{"edit_instruct"},
		MinRuntime:    "diffusers>=0.21.0",
	},
	"qwen-image-edit-plus": {
		Name:          "qwen-image-edit-plus",
		PipelineClass: "QwenImageEditPlusPipeline",
		Ready:         true,
		Operations:    []string{"edit_instruct"},
		MinRuntime:    "diffusers>=0.36.0",
	},
	"flux-2-klein": {
		Name:          "flux-2-klein",
		PipelineClass: "Flux2KleinPipeline",
		Ready:         true,
		Operations:    []string{"edit_instruct", "image_to_image", "text_to_image"},
		MinRuntime:    "diffusers>=0.37.0",
	},
	"flux-1-kontext": {
		Name:          "flux-1-kontext",
		PipelineClass: "FluxKontextPipeline",
		Ready:         true,
		Operations:    []string{"edit_instruct"},
		MinRuntime:    "diffusers>=0.34.0",
	},
	"longcat-image-edit": {
		Name:          "longcat-image-edit",
		PipelineClass: "LongCatImageEditPipeline",
		Ready:         false,
		Operations:    []string{"edit_instruct"},
		MinRuntime:    "diffusers (LongCatImageEditPipeline; recent/custom build)",
		Pending:       "diffusers LongCatImageEditPipeline needs a recent/custom diffusers; pin the version + prove an attended run before enabling.",
	},
}

// DiffusersFamilyForPipelineClass returns the family whose adapter loads the
// diffusers pipeline class named by a repo's model_index.json (_class_name).
// This is how an imported repo finds its adapter with no operator input.
func DiffusersFamilyForPipelineClass(pipelineClass string) (DiffusersFamily, bool) {
	pc := strings.TrimSpace(pipelineClass)
	for _, f := range DiffusersFamilies() {
		if f.PipelineClass == pc {
			return f, true
		}
	}
	return DiffusersFamily{}, false
}

// DiffusersFamilyByName returns the registered family adapter for name.
func DiffusersFamilyByName(name string) (DiffusersFamily, bool) {
	f, ok := diffusersFamilies[name]
	return f, ok
}

// DiffusersFamilies returns every registered family adapter, name-sorted.
func DiffusersFamilies() []DiffusersFamily {
	out := make([]DiffusersFamily, 0, len(diffusersFamilies))
	for _, f := range diffusersFamilies {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
