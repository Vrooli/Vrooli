package models

import "sort"

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
}

// diffusersFamilies is the canonical adapter registry. Keep in lockstep with the
// Python FAMILIES table in _diffusers.py (TestDiffusersFamilyAdaptersMirrorPython
// asserts parity). Adding a model of an existing family is a registry row; a new
// architecture is one row here + one Python adapter + (once proven) Ready=true.
var diffusersFamilies = map[string]DiffusersFamily{
	"instruct-pix2pix": {
		Name:          "instruct-pix2pix",
		PipelineClass: "StableDiffusionInstructPix2PixPipeline",
		Ready:         true,
	},
	"qwen-image-edit-plus": {
		Name:          "qwen-image-edit-plus",
		PipelineClass: "QwenImageEditPlusPipeline",
		Ready:         true,
	},
	"flux-2-klein": {
		Name:          "flux-2-klein",
		PipelineClass: "Flux2KleinPipeline",
		Ready:         false,
		Pending:       "diffusers Flux2KleinPipeline is install-from-git as of 2026-06; pin a released version + prove an attended run before enabling.",
	},
	"longcat-image-edit": {
		Name:          "longcat-image-edit",
		PipelineClass: "LongCatImageEditPipeline",
		Ready:         false,
		Pending:       "diffusers LongCatImageEditPipeline needs a recent/custom diffusers; pin the version + prove an attended run before enabling.",
	},
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
