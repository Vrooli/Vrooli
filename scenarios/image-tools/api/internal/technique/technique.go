// Package technique owns the *technique* layer of the Operation/Technique/Model
// substrate: the named, proven pipelines that map a model architecture onto a
// single operation, plus the pure arg-builders that assemble each pipeline's
// backend invocation.
//
// Before this package the arg-building lived inside the ~44KB ai/providers.go
// monolith, mixed with backend-process management (exec/availability/host-tool).
// The smell the refactor removed was three buildDiffusers* funcs (a dispatcher
// plus inpaint and edit-instruct builders) for one backend, differing only by
// operation. Here each is a flat Technique row: a name, the op it yields, the
// concrete pipeline class (when meaningful), a Ready/caveat gate, and the pure
// Build function. internal/ai (the backend-process adapter) owns *which* backend
// exposes *which* techniques; it consumes these rows and never re-declares the
// arg shapes. See docs/internal/TECHNIQUE-SUBSTRATE.md.
package technique

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/opparams"
)

// ArgBuilder assembles the argv (after the program name) for one technique from a
// run request. modelDir is the installed model's directory. It is a pure function
// of its inputs so it is unit-testable without executing anything. ArgBuilder is
// the TechniqueBuilder seam: a func value, trivially faked in tests.
type ArgBuilder func(req backends.Request, modelDir string) ([]string, error)

// Technique is one named, proven pipeline that yields a single operation. The
// flat table of these rows replaces the per-backend bespoke build functions.
type Technique struct {
	// Name is the stable identifier of the technique (e.g. "diffusers-inpaint").
	Name string
	// Op is the single operation this technique yields.
	Op string
	// Ready reports whether the technique is proven runnable. An unready technique
	// is structurally un-offerable (no-vaporware, decisions 117/120).
	Ready bool
	// Caveat is the quality note surfaced when this technique is used for a derived
	// (non-native) operation. Empty for native techniques.
	Caveat string
	// Adapters reports whether this technique's Build consumes the conditioning
	// adapter stack (req.Adapters) — i.e. it can apply LoRA/ControlNet/IP-Adapter.
	// Provider/backend selection uses this to route a conditioned request to a
	// backend that can actually honor it (the diffusers sidecar), never to one that
	// would drop or reject the modifiers (stable-diffusion.cpp).
	Adapters bool
	// Build is the pure arg-builder for this technique.
	Build ArgBuilder
}

// SupportsAdapters reports whether the technique serving op consumes the
// conditioning adapter stack. False when no technique serves op.
func (s Set) SupportsAdapters(op string) bool {
	t, ok := s[op]
	return ok && t.Adapters
}

// Set maps an operation to the technique that serves it for one backend. A
// backend-process provider holds one Set and dispatches Execute by req.Operation.
type Set map[string]Technique

// NewSet builds a Set keyed by each technique's Op. A duplicate op is a
// programming error (two techniques for one op on one backend) and panics at
// construction so it surfaces in tests, not at runtime.
func NewSet(techniques ...Technique) Set {
	s := make(Set, len(techniques))
	for _, t := range techniques {
		if _, dup := s[t.Op]; dup {
			panic(fmt.Sprintf("technique: duplicate op %q in set", t.Op))
		}
		s[t.Op] = t
	}
	return s
}

// Single is a convenience for a one-entry Set (common in tests).
func Single(op string, b ArgBuilder) Set {
	return Set{op: {Name: op, Op: op, Ready: true, Build: b}}
}

// =============================================================================
// Request accessors. Pure helpers reading a backends.Request the way a backend
// invocation needs it; shared by every Build function (and the in-process
// builtin providers in internal/ai).
// =============================================================================

// Input0 returns the first input image key, erroring when none is present.
func Input0(req backends.Request) (string, error) {
	if len(req.InputKeys) == 0 || req.InputKeys[0] == "" {
		return "", fmt.Errorf("missing input image")
	}
	return req.InputKeys[0], nil
}

// MaskPath returns the second input (the mask), erroring when none is present.
func MaskPath(req backends.Request) (string, error) {
	if len(req.InputKeys) < 2 || req.InputKeys[1] == "" {
		return "", fmt.Errorf("missing mask image")
	}
	return req.InputKeys[1], nil
}

// ReferencePath returns the third input (the reference image: the source face of
// face_swap), erroring when none is present.
func ReferencePath(req backends.Request) (string, error) {
	if len(req.InputKeys) < 3 || req.InputKeys[2] == "" {
		return "", fmt.Errorf("missing reference image")
	}
	return req.InputKeys[2], nil
}

// IntParam reads an int param, falling back to def when absent or zero/invalid.
func IntParam(req backends.Request, key string, def int) int {
	if v, ok := req.Params[key]; ok {
		if n, err := strconv.Atoi(v); err == nil && n != 0 {
			return n
		}
	}
	return def
}

// StrParam reads a string param (empty when absent).
func StrParam(req backends.Request, key string) string { return req.Params[key] }

// LoRAArgs returns the repeatable `--lora <path>:<scale>` flags for the LoRA
// adapters in the request's conditioning stack (Phase 4). Each adapter's single
// weight file is resolved inside its installed dir (ResolvedAdapter.Dir, filled by
// the engine). ControlNet / IP-Adapter adapters are handled by later phases and
// skipped here. The wire format mirrors the Python parser
// (_adapters.parse_lora_spec); the parity test asserts they agree. It errors when
// a LoRA adapter declares no resolvable weight file (fail closed, never silently
// drop a requested modifier).
func LoRAArgs(req backends.Request) ([]string, error) {
	var args []string
	for _, a := range req.Adapters {
		if a.Kind != adapters.KindLoRA {
			continue
		}
		path := adapterWeightFile(a.Dir)
		if path == "" {
			return nil, fmt.Errorf("technique: lora adapter %q has no weight file in %q", a.ID, a.Dir)
		}
		args = append(args, "--lora", path+":"+strconv.FormatFloat(a.Scale, 'g', -1, 64))
	}
	return args, nil
}

// ControlNetArgs returns the repeatable `--controlnet <dir>:<scale>:<image>` flags
// for the ControlNet adapters in the request's conditioning stack (Phase 5). The
// dir is the adapter's installed diffusers repo (loaded with
// ControlNetModel.from_pretrained); the image is the conditioning map — already
// preprocessed (the auto-preprocess path runs the adapter's declared preprocessor
// op as a Look step BEFORE the conditioned generate, so by execution time the
// control image is a concrete map; a pre-made map is accepted the same way). Both
// the dir and the image are colon-free local paths, so the Python parser
// (_adapters.parse_controlnet_spec) splits the scale + image off the right with
// rsplit. It fails closed when a ControlNet has no installed dir or no conditioning
// image — a structural conditioner with no control map cannot run.
func ControlNetArgs(req backends.Request) ([]string, error) {
	var args []string
	for _, a := range req.Adapters {
		if a.Kind != adapters.KindControlNet {
			continue
		}
		if a.Dir == "" {
			return nil, fmt.Errorf("technique: controlnet adapter %q has no installed dir", a.ID)
		}
		if a.ConditioningImageKey == "" {
			return nil, fmt.Errorf("technique: controlnet adapter %q has no conditioning image (preprocess the input first, or pass a pre-made control map)", a.ID)
		}
		args = append(args, "--controlnet", a.Dir+":"+strconv.FormatFloat(a.Scale, 'g', -1, 64)+":"+a.ConditioningImageKey)
	}
	return args, nil
}

// IPAdapterArgs returns the repeatable `--ip-adapter <weightfile>:<scale>:<ref>`
// flags for the IP-Adapter adapters in the request's conditioning stack (Phase 6).
// The weight file is the single .safetensors inside the adapter dir (loaded with
// pipe.load_ip_adapter); the reference image is mandatory (it is the adapter's
// prompt). Like ControlNetArgs the trailing scale + image are colon-free so the
// Python parser splits them off the right. Fails closed on a missing weight file or
// reference image.
func IPAdapterArgs(req backends.Request) ([]string, error) {
	var args []string
	for _, a := range req.Adapters {
		if a.Kind != adapters.KindIPAdapter {
			continue
		}
		path := adapterWeightFile(a.Dir)
		if path == "" {
			return nil, fmt.Errorf("technique: ip-adapter %q has no weight file in %q", a.ID, a.Dir)
		}
		if a.ConditioningImageKey == "" {
			return nil, fmt.Errorf("technique: ip-adapter %q requires a reference image", a.ID)
		}
		args = append(args, "--ip-adapter", path+":"+strconv.FormatFloat(a.Scale, 'g', -1, 64)+":"+a.ConditioningImageKey)
	}
	return args, nil
}

// conditioningArgs appends LoRA, ControlNet, and IP-Adapter flags in the canonical
// order (the resolver already sorted the stack LoRA→ControlNet→IP-Adapter; this
// keeps the argv grouped by kind regardless). Shared by the diffusers generate /
// transform builders so a conditioned request threads its whole stack through.
func conditioningArgs(req backends.Request) ([]string, error) {
	lora, err := LoRAArgs(req)
	if err != nil {
		return nil, err
	}
	cn, err := ControlNetArgs(req)
	if err != nil {
		return nil, err
	}
	ip, err := IPAdapterArgs(req)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(lora)+len(cn)+len(ip))
	out = append(out, lora...)
	out = append(out, cn...)
	out = append(out, ip...)
	return out, nil
}

// adapterWeightFile resolves the adapter's own weight file inside its installed
// directory (a LoRA / IP-Adapter ships one .safetensors; some are .bin/.pt/.ckpt).
// Empty when the dir holds none.
//
// Two layouts are supported: a flat single-file install (LoRA) and a repo
// snapshot that also carries the CLIP image encoder in an image_encoder/ subdir
// (an IP-Adapter needs the encoder to embed its reference image). The walk skips
// any image_encoder/ directory — those weights are the encoder, never the
// adapter — and prefers .safetensors over the other formats, so the IP-Adapter
// weight is picked rather than the encoder's model.safetensors.
func adapterWeightFile(dir string) string {
	if dir == "" {
		return ""
	}
	// Fast path: a flat top-level single-file install.
	for _, pattern := range []string{"*.safetensors", "*.ckpt", "*.bin", "*.pt"} {
		if matches, _ := filepath.Glob(filepath.Join(dir, pattern)); len(matches) > 0 {
			return matches[0]
		}
	}
	// Repo-snapshot layout: walk for the weight, skipping the image encoder.
	rank := map[string]int{".safetensors": 4, ".ckpt": 3, ".bin": 2, ".pt": 1}
	var best string
	bestRank := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "image_encoder" {
				return filepath.SkipDir
			}
			return nil
		}
		if r := rank[strings.ToLower(filepath.Ext(path))]; r > bestRank {
			best, bestRank = path, r
		}
		return nil
	})
	return best
}

// =============================================================================
// Per-technique argument builders. Each mirrors the documented CLI of a real
// standalone backend; they are pure and unit-tested (arg assembly), while live
// execution is gated on the program + model being installed.
// =============================================================================

// sdModelArg resolves the weights argument for sd-cli. The image-tools model
// install dir holds a single-file checkpoint (.safetensors/.gguf/.ckpt), but
// sd-cli's `-m` expects that FILE — handed a bare directory it tries to load a
// diffusers-layout tree and fails. Resolve the single-file checkpoint inside the
// dir; fall back to the path as-is when none is found (already a file, or a
// diffusers-layout dir sd-cli can load directly).
func sdModelArg(modelDir string) string {
	for _, pattern := range []string{"*.safetensors", "*.gguf", "*.ckpt"} {
		if matches, _ := filepath.Glob(filepath.Join(modelDir, pattern)); len(matches) > 0 {
			return matches[0]
		}
	}
	return modelDir
}

// StableDiffusionCpp assembles a stable-diffusion.cpp `sd-cli` invocation for
// text_to_image / image_to_image. Shape: sd-cli -m <model> -p <prompt> [-n <neg>]
// [--cfg-scale x] --steps n -W w -H h [-s seed] [-i in --strength x] -o out.
// image_to_image is selected by passing -i <init>; the run mode stays the default
// img_gen.
func StableDiffusionCpp(req backends.Request, modelDir string) ([]string, error) {
	// Conditioning adapters (LoRA/ControlNet/IP-Adapter) run on the diffusers
	// sidecar, not sd.cpp. Fail closed rather than silently dropping a requested
	// modifier — the resolver should route a conditioned request to a diffusers
	// model, but if one reaches here it must not run unconditioned.
	if len(req.Adapters) > 0 {
		return nil, fmt.Errorf("stable-diffusion.cpp does not support conditioning adapters; use a diffusers-backed model for LoRA/ControlNet/IP-Adapter")
	}
	args := []string{"-m", sdModelArg(modelDir), "-o", req.Output.LocalPath, "-p", StrParam(req, "prompt")}
	if !req.GPU {
		// A GPU-capable sd launcher can also serve CPU work. Explicitly pin
		// compute AND parameter placement; offload-to-cpu alone retains GPU compute.
		args = append(args, "--backend", "cpu", "--params-backend", "cpu")
		// sd-cli otherwise sizes its compute pool to every physical core, and
		// that explicit count overrides OMP_NUM_THREADS, so the admitted CPU
		// budget must be bound through argv.
		if req.CPUThreads > 0 {
			args = append(args, "-t", strconv.Itoa(req.CPUThreads))
		}
	}
	if neg := StrParam(req, "negative_prompt"); neg != "" {
		args = append(args, "-n", neg)
	}
	if cfg := StrParam(req, "cfg_scale"); cfg != "" {
		args = append(args, "--cfg-scale", cfg)
	}
	args = append(args, "--steps", strconv.Itoa(IntParam(req, "steps", 20)))
	args = append(args, "-W", strconv.Itoa(IntParam(req, "width", 512)))
	args = append(args, "-H", strconv.Itoa(IntParam(req, "height", 512)))
	if seed := StrParam(req, "seed"); seed != "" {
		args = append(args, "-s", seed)
	}
	if req.Operation == "image_to_image" {
		in, err := Input0(req)
		if err != nil {
			return nil, err
		}
		args = append(args, "-i", in)
		if s := StrParam(req, "strength"); s != "" {
			args = append(args, "--strength", s)
		}
	}
	return args, nil
}

// DiffusersInpaint assembles a diffusers python-sidecar inpaint invocation. It
// serves inpaint / outpaint / background_replace: all three are masked-regenerate
// ops with the same argv shape (the mask marks the region to synthesize, the
// prompt steers it), sharing the inpaint sidecar module. The concrete inpaint
// pipeline class is resolved from the model's registry architecture (passed via
// --architecture: sd15 → StableDiffusionInpaintPipeline, sdxl →
// StableDiffusionXLInpaintPipeline), so a base text2image checkpoint inpaints via
// the standard pipeline — the derived capability. `strength` is the masked-region
// change knob; the rest mirror the generation params.
// Shape: python3 -m image_tools_sidecar.inpaint --model <dir> --architecture <arch>
// --image <in> --mask <mask> --prompt <p> --out <out> [--strength x] [--steps n]
// [--guidance x] [--negative-prompt p] [--seed s].
func DiffusersInpaint(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	mask, err := MaskPath(req)
	if err != nil {
		return nil, err
	}
	// A unified instruction-edit family (FLUX.2 [klein], ...) has no inpaint
	// pipeline of its own: it serves the masked ops by editing a crop around the
	// selection and pasting it back through the feathered mask.
	if family := strings.TrimSpace(req.Model.Runtime.Family); family != "" {
		return diffusersMaskedEditArgs(req, modelDir, family, in, mask)
	}
	arch := strings.TrimSpace(string(req.Model.Architecture))
	if arch == "" {
		return nil, fmt.Errorf("diffusers inpaint: model %q has no architecture (cannot resolve inpaint pipeline)", req.Model.ID)
	}
	args := []string{
		"-m", "image_tools_sidecar.inpaint",
		"--model", modelDir,
		"--model-id", req.Model.ID,
		"--architecture", arch,
		"--image", in,
		"--mask", mask,
		"--prompt", StrParam(req, "prompt"),
		"--out", req.Output.LocalPath,
	}
	for _, asset := range req.Model.Source.Assets {
		if asset.Filename == "" || asset.SHA256 == "" {
			continue
		}
		args = append(args, "--model-sha256", asset.SHA256)
		break
	}
	if variant := strings.TrimSpace(req.Model.Runtime.Variant); variant != "" {
		args = append(args, "--variant", variant)
	}
	if revision := strings.TrimSpace(req.Model.Source.Repo.Revision); revision != "" {
		args = append(args, "--model-revision", revision)
	}
	inpaint, err := opparams.DecodeInpaintParams(req.Params)
	if err != nil {
		return nil, err
	}
	if inpaint.CropToMask != nil && *inpaint.CropToMask {
		args = append(args, "--crop-to-mask")
		if inpaint.ContextPixels != nil && *inpaint.ContextPixels > 0 {
			args = append(args, "--crop-context-pixels", strconv.Itoa(*inpaint.ContextPixels))
		}
		if inpaint.MaxEdgePx != nil && *inpaint.MaxEdgePx > 0 {
			args = append(args, "--crop-max-edge-px", strconv.Itoa(*inpaint.MaxEdgePx))
		}
		if inpaint.NeutralizeMaskedPixels != nil && *inpaint.NeutralizeMaskedPixels {
			args = append(args, "--neutralize-masked-pixels")
		}
	}
	lora, err := LoRAArgs(req)
	if err != nil {
		return nil, err
	}
	args = append(args, lora...)
	if s := StrParam(req, "strength"); s != "" {
		args = append(args, "--strength", s)
	}
	if s := IntParam(req, "steps", 0); s > 0 {
		args = append(args, "--steps", strconv.Itoa(s))
	}
	if g := StrParam(req, "cfg_scale"); g != "" {
		args = append(args, "--guidance", g)
	}
	if np := StrParam(req, "negative_prompt"); np != "" {
		args = append(args, "--negative-prompt", np)
	}
	if seed := StrParam(req, "seed"); seed != "" {
		args = append(args, "--seed", seed)
	}
	return args, nil
}

// diffusersMaskedEditArgs assembles the selected-area edit for an instruction
// family: the sidecar crops to the mask plus context, runs the family edit with
// the prompt as the instruction, and blends the result back through the grown,
// feathered mask so everything outside it stays the source's own pixels.
// Shape: python3 -m image_tools_sidecar.masked_edit --model <dir> --family <fam>
// --image <in> --mask <mask> --prompt <instruction> --out <out> [--grow-px n]
// [--feather-px n] [--crop-context-pixels n] [--crop-max-edge-px n] [--full-frame]
// [--steps n] [--guidance x] [--negative-prompt p] [--seed s].
func diffusersMaskedEditArgs(req backends.Request, modelDir, family, in, mask string) ([]string, error) {
	if len(req.Adapters) > 0 {
		return nil, fmt.Errorf("diffusers %s: conditioning adapters are not supported on family %q (model %q)", req.Operation, family, req.Model.ID)
	}
	args := []string{
		"-m", "image_tools_sidecar.masked_edit",
		"--model", modelDir,
		"--model-id", req.Model.ID,
		"--family", family,
		"--image", in,
		"--mask", mask,
		"--prompt", StrParam(req, "prompt"),
		"--out", req.Output.LocalPath,
	}
	p, err := opparams.DecodeInpaintParams(req.Params)
	if err != nil {
		return nil, err
	}
	if p.GrowPx != nil {
		args = append(args, "--grow-px", strconv.Itoa(*p.GrowPx))
	}
	if p.FeatherPx != nil {
		args = append(args, "--feather-px", strconv.Itoa(*p.FeatherPx))
	}
	if p.CropToMask != nil && !*p.CropToMask {
		args = append(args, "--full-frame")
	}
	if p.ContextPixels != nil && *p.ContextPixels > 0 {
		args = append(args, "--crop-context-pixels", strconv.Itoa(*p.ContextPixels))
	}
	if p.MaxEdgePx != nil && *p.MaxEdgePx > 0 {
		args = append(args, "--crop-max-edge-px", strconv.Itoa(*p.MaxEdgePx))
	}
	if s := IntParam(req, "steps", 0); s > 0 {
		args = append(args, "--steps", strconv.Itoa(s))
	}
	if g := StrParam(req, "cfg_scale"); g != "" {
		args = append(args, "--guidance", g)
	}
	if ig := StrParam(req, "strength"); ig != "" {
		args = append(args, "--image-guidance", ig)
	}
	if np := StrParam(req, "negative_prompt"); np != "" {
		args = append(args, "--negative-prompt", np)
	}
	if seed := StrParam(req, "seed"); seed != "" {
		args = append(args, "--seed", seed)
	}
	return args, nil
}

// DiffusersEditInstruct assembles a diffusers instruction-edit invocation
// (InstructPix2Pix / Qwen-Image-Edit class). The op is identity-preserving and
// prompt-only: there is no mask, and `prompt` is the natural-language instruction
// ("make it winter"). cfg_scale maps to the text guidance scale; image_guidance
// (faithfulness to the source) defaults inside the sidecar but can be overridden
// via the `strength` param. The concrete pipeline class is selected by the
// model's registry runtime family (passed via --family), not hardcoded — so the
// diffusers backend runs every registered edit family.
// Shape: python3 -m image_tools_sidecar.edit_instruct --model <dir> --family <fam>
// --image <in> --prompt <instruction> --out <out> [--width n] [--height n]
// [--steps n] [--guidance x] [--image-guidance x] [--negative-prompt p] [--seed s].
func DiffusersEditInstruct(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	family := strings.TrimSpace(req.Model.Runtime.Family)
	if family == "" {
		return nil, fmt.Errorf("diffusers edit_instruct: model %q has no runtime.family (not runnable)", req.Model.ID)
	}
	args := []string{
		"-m", "image_tools_sidecar.edit_instruct",
		"--model", modelDir,
		"--family", family,
		"--image", in,
		"--prompt", StrParam(req, "prompt"),
		"--out", req.Output.LocalPath,
	}
	// An explicit output size is threaded for families whose pipeline takes one
	// (FLUX.2 [klein]); without it the family keeps its reference-derived size.
	if w := IntParam(req, "width", 0); w > 0 {
		args = append(args, "--width", strconv.Itoa(w))
	}
	if h := IntParam(req, "height", 0); h > 0 {
		args = append(args, "--height", strconv.Itoa(h))
	}
	// Pass --steps only when explicitly requested so each family applies its own
	// default (e.g. 20 for InstructPix2Pix, 40 for Qwen-Image-Edit, 4 for klein).
	if s := IntParam(req, "steps", 0); s > 0 {
		args = append(args, "--steps", strconv.Itoa(s))
	}
	if g := StrParam(req, "cfg_scale"); g != "" {
		args = append(args, "--guidance", g)
	}
	if ig := StrParam(req, "strength"); ig != "" {
		args = append(args, "--image-guidance", ig)
	}
	if np := StrParam(req, "negative_prompt"); np != "" {
		args = append(args, "--negative-prompt", np)
	}
	if seed := StrParam(req, "seed"); seed != "" {
		args = append(args, "--seed", seed)
	}
	return args, nil
}

// diffusersPipelineArgs selects how the generate/transform sidecar resolves the
// pipeline. A model naming a runtime.family is loaded by that family adapter (a
// unified pipeline the base-checkpoint AutoPipeline cannot resolve); every other
// diffusers model resolves its pipeline class from the architecture. The
// conditioning adapter stack is proven only on base checkpoints, so a family
// request carrying adapters is refused here rather than dropped downstream.
func diffusersPipelineArgs(req backends.Request, op string) ([]string, error) {
	if family := strings.TrimSpace(req.Model.Runtime.Family); family != "" {
		if len(req.Adapters) > 0 {
			return nil, fmt.Errorf("diffusers %s: conditioning adapters are not supported on family %q (model %q)", op, family, req.Model.ID)
		}
		return []string{"--family", family}, nil
	}
	arch := strings.TrimSpace(string(req.Model.Architecture))
	if arch == "" {
		return nil, fmt.Errorf("diffusers %s: model %q has no architecture (cannot resolve pipeline)", op, req.Model.ID)
	}
	args := []string{"--architecture", arch}
	// A repo that ships only reduced-precision files (name.fp16.safetensors) loads
	// through diffusers' variant selector; the registry declares it, never the caller.
	if variant := strings.TrimSpace(req.Model.Runtime.Variant); variant != "" {
		args = append(args, "--variant", variant)
	}
	return args, nil
}

// DiffusersText2Image assembles a diffusers python-sidecar text2image generate
// invocation. It serves text_to_image on the diffusers backend for two model
// shapes, chosen by diffusersPipelineArgs:
//   - a base checkpoint (diffusers-REPO weights sd.cpp cannot load; a single-file
//     checkpoint generates via stable-diffusion.cpp instead), whose pipeline class
//     is resolved from the architecture (--architecture sd15/sdxl);
//   - a unified family model (runtime.family, e.g. FLUX.2 [klein]), which its
//     family adapter loads for every op it declares (--family).
//
// Shape: python3 -m image_tools_sidecar.text_to_image --model <dir>
// (--architecture <arch> | --family <fam>) --prompt <p> --out <out> [--width n]
// [--height n] [--steps n] [--guidance x] [--negative-prompt p] [--seed s].
func DiffusersText2Image(req backends.Request, modelDir string) ([]string, error) {
	pipeline, err := diffusersPipelineArgs(req, "text_to_image")
	if err != nil {
		return nil, err
	}
	args := []string{
		"-m", "image_tools_sidecar.text_to_image",
		"--model", modelDir,
	}
	args = append(args, pipeline...)
	args = append(args,
		"--prompt", StrParam(req, "prompt"),
		"--out", req.Output.LocalPath,
	)
	cond, err := conditioningArgs(req)
	if err != nil {
		return nil, err
	}
	args = append(args, cond...)
	if w := IntParam(req, "width", 0); w > 0 {
		args = append(args, "--width", strconv.Itoa(w))
	}
	if h := IntParam(req, "height", 0); h > 0 {
		args = append(args, "--height", strconv.Itoa(h))
	}
	if s := IntParam(req, "steps", 0); s > 0 {
		args = append(args, "--steps", strconv.Itoa(s))
	}
	if g := StrParam(req, "cfg_scale"); g != "" {
		args = append(args, "--guidance", g)
	}
	if np := StrParam(req, "negative_prompt"); np != "" {
		args = append(args, "--negative-prompt", np)
	}
	if seed := StrParam(req, "seed"); seed != "" {
		args = append(args, "--seed", seed)
	}
	return args, nil
}

// DiffusersImg2Img assembles a diffusers python-sidecar img2img transform
// invocation on the diffusers backend, for the same two model shapes as
// DiffusersText2Image: a base checkpoint's architecture-derived img2img
// (`strength` = how far the output may drift from the init image; a single-file
// checkpoint takes the stable-diffusion.cpp sd-img2img path instead), or a
// unified family model that transforms with the init image as its reference.
// Shape: python3 -m image_tools_sidecar.image_to_image --model <dir>
// (--architecture <arch> | --family <fam>) --image <in> --prompt <p> --out <out>
// [--strength x] [--steps n] [--guidance x] [--negative-prompt p] [--seed s].
func DiffusersImg2Img(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	pipeline, err := diffusersPipelineArgs(req, "image_to_image")
	if err != nil {
		return nil, err
	}
	args := []string{
		"-m", "image_tools_sidecar.image_to_image",
		"--model", modelDir,
	}
	args = append(args, pipeline...)
	args = append(args,
		"--image", in,
		"--prompt", StrParam(req, "prompt"),
		"--out", req.Output.LocalPath,
	)
	cond, err := conditioningArgs(req)
	if err != nil {
		return nil, err
	}
	args = append(args, cond...)
	if s := StrParam(req, "strength"); s != "" {
		args = append(args, "--strength", s)
	}
	if s := IntParam(req, "steps", 0); s > 0 {
		args = append(args, "--steps", strconv.Itoa(s))
	}
	if g := StrParam(req, "cfg_scale"); g != "" {
		args = append(args, "--guidance", g)
	}
	if np := StrParam(req, "negative_prompt"); np != "" {
		args = append(args, "--negative-prompt", np)
	}
	if seed := StrParam(req, "seed"); seed != "" {
		args = append(args, "--seed", seed)
	}
	return args, nil
}

// Iopaint assembles an iopaint object-removal invocation.
// Shape: iopaint run --model <dir> --device <cpu|cuda> --image <in> --mask <mask> --output <out>.
func Iopaint(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	mask, err := MaskPath(req)
	if err != nil {
		return nil, err
	}
	device := "cpu"
	if req.GPU {
		device = "cuda"
	}
	return []string{
		"run",
		"--model", modelDir,
		"--device", device,
		"--image", in,
		"--mask", mask,
		"--output", req.Output.LocalPath,
	}, nil
}

// realesrganModel picks the bundled model. A 4x upscale uses realesrgan-x4plus:
// on the guinea-pig-v1 photo (4x of a 756x1008 downscale against the real
// 3024x4032 original) its LPIPS is 0.187 against 0.263 for bicubic, while
// realesr-animevideov3 reaches only 0.246 and realesrgan-x4plus-anime 0.302. It
// costs about 27 s on the leased GPU against 5 s. realesr-animevideov3 stays for
// the variable scales (2x, 3x) that x4plus cannot do, and for denoise.
func realesrganModel(operation string, scale int) string {
	if operation == "upscale" && scale == 4 {
		return "realesrgan-x4plus"
	}
	return "realesr-animevideov3"
}

// Realesrgan assembles a realesrgan-ncnn-vulkan invocation. The prebuilt release
// ships its own `models/` directory (realesr-animevideov3 at -s 2/3/4 and
// realesrgan-x4plus), which the binary resolves relative to its own location — so
// we pass a bundled model NAME via -n and let the tool find the weights, rather
// than a -m model dir or the image-tools model id. realesr-animevideov3 is the
// variable-scale general model used for both upscale and denoise.
// Shape: realesrgan-ncnn-vulkan -i <in> -o <out> -s <scale> -n <bundled-model>.
func Realesrgan(req backends.Request, _ string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	scale := IntParam(req, "scale", 4)
	if scale < 2 || scale > 4 {
		scale = 4
	}
	args := []string{
		"-i", in,
		"-o", req.Output.LocalPath,
		"-s", strconv.Itoa(scale),
		"-n", realesrganModel(req.Operation, scale),
	}
	// The tool opens the first Vulkan GPU unless told otherwise, so the device
	// must follow the admission: the leased GPU, or the software device on CPU.
	device, err := NcnnDeviceArgs(req.GPU, req.GPUUUID)
	if err != nil {
		return nil, err
	}
	return append(args, device...), nil
}

// Rembg assembles a rembg invocation. background_removal writes an RGBA cutout.
// background_replace asks rembg to composite the cutout over a caller-supplied
// background_color (R,G,B or R,G,B,A) so replacement requests do not silently
// degrade to removal-only output.
// Shape: rembg i -m <model-id> [--bgcolor r,g,b,a] <in> <out>.
func Rembg(req backends.Request, _ string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	args := []string{"i", "-m", req.Model.ID}
	if req.Operation == "background_replace" {
		color := strings.TrimSpace(StrParam(req, "background_color"))
		if color == "" {
			return nil, fmt.Errorf("rembg background_replace requires background_color")
		}
		args = append(args, "--bgcolor", color)
	}
	args = append(args, in, req.Output.LocalPath)
	return args, nil
}

// LlamaCppCaption assembles a llama.cpp multimodal caption invocation.
// Shape: llama-mtmd-cli -m <gguf> --mmproj <gguf> --image <in> -p <prompt> -n <tokens>.
func LlamaCppCaption(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	model, mmproj, err := llamaCppModelPaths(req, modelDir)
	if err != nil {
		return nil, err
	}
	prompt := StrParam(req, "prompt")
	if prompt == "" {
		prompt = "Describe this image concisely."
	}
	args := []string{
		"--verbosity", "0",
		"-m", model,
		"--mmproj", mmproj,
		"--image", in,
		"-p", prompt,
		"-n", strconv.Itoa(IntParam(req, "max_tokens", 96)),
	}
	if temp := StrParam(req, "temperature"); temp != "" {
		args = append(args, "--temp", temp)
	}
	return args, nil
}

func llamaCppModelPaths(req backends.Request, modelDir string) (model string, mmproj string, err error) {
	for _, a := range req.Model.Source.Assets {
		if a.Kind != models.ArtifactGGUF || a.Filename == "" {
			continue
		}
		path := filepath.Join(modelDir, a.Filename)
		if strings.Contains(strings.ToLower(a.Filename), "mmproj") {
			if mmproj == "" {
				mmproj = path
			}
			continue
		}
		if model == "" {
			model = path
		}
	}
	if model == "" || mmproj == "" {
		return "", "", fmt.Errorf("llama.cpp caption requires one text GGUF and one mmproj GGUF asset")
	}
	return model, mmproj, nil
}

// OnnxSidecar assembles an invocation of the in-repo onnxruntime python sidecar.
// The sidecar supports explicit CPU fallback and an admitted CUDA execution path.
// One dispatch per supported op. The model
// argument is the resolved ONNX weight file inside the model dir (from the
// registry asset list), not the directory, so the sidecar loads it directly.
func OnnxSidecar(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	model := OnnxModelPath(req, modelDir)
	var module string
	switch req.Operation {
	case "background_removal":
		module = "image_tools_sidecar.bg_removal"
	case "object_removal":
		module = "image_tools_sidecar.lama"
	case "denoise":
		module = "image_tools_sidecar.denoise"
	case "deblur":
		module = "image_tools_sidecar.deblur"
	case "colorize":
		module = "image_tools_sidecar.colorize"
	case "depth_map":
		module = "image_tools_sidecar.depth"
	case "face_restore":
		// The model dir carries the restorer and its YuNet face detector; the
		// sidecar tells them apart by ONNX signature.
		module = "image_tools_sidecar.face_restore"
		model = modelDir
	case "scene_geometry":
		module = "image_tools_sidecar.geometry"
	case "object_detection":
		module = "image_tools_sidecar.detect"
	case "segment":
		module = "image_tools_sidecar.segment"
		model = modelDir
	case "tagging":
		module = "image_tools_sidecar.tagging"
	case "nsfw_classify":
		module = "image_tools_sidecar.nsfw"
	case "embedding":
		module = "image_tools_sidecar.embedding"
	case "face_swap":
		// The model dir carries the swapper, the ArcFace embedder, the YuNet
		// detector and the XSeg occluder; the sidecar tells them apart by ONNX
		// signature.
		module = "image_tools_sidecar.face_swap"
		model = modelDir
	default:
		return nil, fmt.Errorf("onnxruntime sidecar: unsupported operation %q", req.Operation)
	}
	args := []string{
		"-m", module,
		"--model", model,
		"--image", in,
		"--out", req.Output.LocalPath,
		"--device", map[bool]string{true: "cuda", false: "cpu"}[req.GPU],
	}
	if req.Operation == "object_removal" {
		mask, err := MaskPath(req)
		if err != nil {
			return nil, err
		}
		args = append(args, "--mask", mask)
		removal, err := opparams.DecodeObjectRemovalParams(req.Params)
		if err != nil {
			return nil, err
		}
		if removal.BackgroundRepair != nil {
			raw, err := json.Marshal(removal.BackgroundRepair)
			if err != nil {
				return nil, fmt.Errorf("object_removal: encode background_repair: %w", err)
			}
			args = append(args, "--params", string(raw))
		}
	}
	if req.Operation == "face_swap" {
		reference, err := ReferencePath(req)
		if err != nil {
			return nil, err
		}
		swap, err := opparams.DecodeFaceSwapParams(req.Params)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(swap)
		if err != nil {
			return nil, fmt.Errorf("face_swap: encode params: %w", err)
		}
		args = append(args, "--reference", reference, "--params", string(raw))
	}
	if req.Operation == "segment" {
		seg, err := opparams.DecodeSegmentParams(req.Params)
		if err != nil {
			return nil, err
		}
		if seg.Segmentation != nil {
			raw, err := json.Marshal(seg.Segmentation)
			if err != nil {
				return nil, fmt.Errorf("segment: encode prompts: %w", err)
			}
			args = append(args, "--prompts", string(raw))
		}
	}
	if req.Operation == "scene_geometry" {
		args = append(args, "--model-id", req.Model.ID)
		geom, err := opparams.DecodeSceneGeometryParams(req.Params)
		if err != nil {
			return nil, err
		}
		if geom.SupportRoi != nil {
			raw, err := json.Marshal(geom.SupportRoi)
			if err != nil {
				return nil, fmt.Errorf("scene_geometry: encode support_roi: %w", err)
			}
			args = append(args, "--support-roi", string(raw))
		}
	}
	return args, nil
}

// PythonSidecar assembles invocations for enabled catalog entries whose native
// backend family is `python-sidecar`. These are heavier Python modules than the
// ONNX CPU floor, but they still use the embedded sidecar package and the same
// host-provisioned Python runtime seam.
func PythonSidecar(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	var module string
	switch req.Operation {
	case "colorize":
		module = "image_tools_sidecar.colorize"
	case "old_photo_restore":
		module = "image_tools_sidecar.restore"
	default:
		return nil, fmt.Errorf("python-sidecar: unsupported operation %q", req.Operation)
	}
	return []string{
		"-m", module,
		"--model", modelDir,
		"--image", in,
		"--out", req.Output.LocalPath,
	}, nil
}

// SAM2Sidecar assembles the isolated Transformers SAM 2 image adapter. The
// model folder contains pinned HF config and weights; processor defaults are
// explicit in the sidecar because upstream processor metadata is sub-kilobyte.
// A text prompt selects the grounded adapter instead (the model folder then
// holds detector/ and segmenter/).
func SAM2Sidecar(req backends.Request, modelDir string) ([]string, error) {
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	device := "cpu"
	if req.GPU {
		device = "cuda"
	}
	seg, err := opparams.DecodeSegmentParams(req.Params)
	if err != nil {
		return nil, err
	}
	// A text phrase is resolved by the grounded model (detector + SAM 2.1); a
	// point or box prompt goes straight to SAM 2.1.
	module := "image_tools_sidecar.sam2"
	if seg.Segmentation != nil && seg.Segmentation.Text != nil {
		module = "image_tools_sidecar.grounded_segment"
	}
	args := []string{"-m", module, "--model", modelDir, "--image", in, "--out", req.Output.LocalPath, "--device", device}
	if seg.Segmentation != nil {
		raw, err := json.Marshal(seg.Segmentation)
		if err != nil {
			return nil, fmt.Errorf("segment: encode prompts: %w", err)
		}
		args = append(args, "--prompts", string(raw))
	}
	return args, nil
}

// TripoSRSidecar invokes the pinned single-image object reconstruction adapter.
// The optional second input is an explicit foreground mask; the module never
// downloads or invokes an automatic background remover.
func TripoSRSidecar(req backends.Request, modelDir string) ([]string, error) {
	if strings.TrimSpace(req.SourceProfileDir) == "" {
		return nil, fmt.Errorf("image_to_object requires an engine-resolved managed TripoSR source profile")
	}
	in, err := Input0(req)
	if err != nil {
		return nil, err
	}
	device := "cpu"
	if req.GPU {
		device = "cuda"
	}
	object, err := opparams.DecodeImageToObjectParams(req.Params)
	if err != nil {
		return nil, err
	}
	resolution := 256
	if object.MeshResolution != nil {
		resolution = *object.MeshResolution
	}
	if resolution != 128 && resolution != 256 {
		return nil, fmt.Errorf("image_to_object mesh resolution must be 128 or 256")
	}
	args := []string{"-m", "image_tools_sidecar.image_to_object", "--model", modelDir, "--source-dir", req.SourceProfileDir, "--image", in, "--out", req.Output.LocalPath, "--device", device, "--mesh-resolution", strconv.Itoa(resolution)}
	if len(req.InputKeys) > 1 && req.InputKeys[1] != "" {
		args = append(args, "--mask", req.InputKeys[1])
	}
	return args, nil
}

// OnnxModelPath resolves the ONNX weight file the sidecar should load: the first
// registry asset of kind ONNX (else the first asset) under the model dir. Falls
// back to the model dir itself when the model declares no assets.
func OnnxModelPath(req backends.Request, modelDir string) string {
	// Closed build profiles own a derived runtime artifact that is not one of the
	// downloaded source assets. Resolve only the validated LaMa profile output;
	// otherwise the source archive/checkpoint could be handed to ONNX Runtime.
	if req.Model.ID == "big-lama-1024-export" && req.Model.BuildProfile != nil &&
		req.Model.BuildProfile.Kind == "lama-onnx-fixed" && req.Model.BuildProfile.OutputFilename == "lama_1024.onnx" {
		return filepath.Join(modelDir, req.Model.BuildProfile.OutputFilename)
	}
	assets := req.Model.Source.Assets
	for _, a := range assets {
		if a.Kind == models.ArtifactONNX && a.Filename != "" {
			return filepath.Join(modelDir, a.Filename)
		}
	}
	if len(assets) > 0 && assets[0].Filename != "" {
		return filepath.Join(modelDir, assets[0].Filename)
	}
	return modelDir
}
