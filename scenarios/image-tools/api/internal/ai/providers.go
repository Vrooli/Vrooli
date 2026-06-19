package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"image-tools/internal/backends"
	"image-tools/internal/models"
)

// commandRunner executes a resolved command. Injected so tests assert argument
// assembly + availability without invoking absent backend binaries.
type commandRunner func(ctx context.Context, name string, args []string) error

// outputRunner executes a command and returns stdout/stderr text for providers
// whose native result is written to stdout instead of an output file.
type outputRunner func(ctx context.Context, name string, args []string) ([]byte, error)

// lookPathFunc resolves a binary on PATH. Injected for the same reason.
type lookPathFunc func(file string) (string, error)

// pythonModuleChecker checks whether a Python executable can import modules.
type pythonModuleChecker func(ctx context.Context, python string, modules []string) error

// argBuilder assembles the argv (after the program name) for one backend from a
// run request. modelDir is the installed model's directory. It is a pure
// function of its inputs so it can be unit-tested without executing anything.
type argBuilder func(req backends.Request, modelDir string) ([]string, error)

// execProvider is a backends.Provider backed by an external CLI/sidecar program.
// One configured instance exists per standalone backend (stable-diffusion.cpp,
// diffusers, iopaint, realesrgan-ncnn-vulkan, rembg, onnxruntime). Availability
// is "the program resolves on PATH"; model-weight presence is a separate gate
// the engine applies (so it can produce a precise "model not installed" hint),
// keeping providers model-agnostic.
type execProvider struct {
	name       string
	program    string // binary or interpreter resolved on PATH (e.g. "sd", "python3")
	ops        []string
	build      argBuilder
	gpuCapable bool // can this backend actually use the GPU? (CPU-only sidecars: false)
	provision  string
	imports    []string
	lookPath   lookPathFunc
	checkPy    pythonModuleChecker
	run        commandRunner
}

func (p *execProvider) Name() string         { return p.name }
func (p *execProvider) Operations() []string { return append([]string(nil), p.ops...) }
func (p *execProvider) Standalone() bool     { return true } // none of these are ComfyUI
func (p *execProvider) IsCloud() bool        { return false }

// GPUCapable reports whether this backend can run on the GPU. The onnxruntime
// sidecar is bound to CPUExecutionProvider, so it returns false and the selector
// labels its runs local-cpu even on a GPU host (honest tier reporting).
func (p *execProvider) GPUCapable() bool { return p.gpuCapable }

// Available reports whether the backend program is resolvable on PATH.
func (p *execProvider) Available(ctx context.Context) bool {
	return p.Availability(ctx).Available
}

// Availability reports software readiness for doctor/selection. It is limited
// to host software presence; model weights and hardware fit are checked by the
// model lifecycle and capabilities layers.
func (p *execProvider) Availability(ctx context.Context) backends.Availability {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	resolved, err := lookPath(p.program)
	if err == nil {
		if len(p.imports) > 0 {
			check := p.checkPy
			if check == nil {
				check = defaultCheckPythonModules
			}
			if err := check(ctx, resolved, p.imports); err != nil {
				return backends.Availability{
					Available: false,
					Detail:    fmt.Sprintf("%s resolved at %s, but Python imports failed: %v", p.program, resolved, err),
					Provision: p.provision,
				}
			}
			return backends.Availability{
				Available: true,
				Detail:    fmt.Sprintf("%s resolved at %s; Python imports ready: %s", p.program, resolved, strings.Join(p.imports, ",")),
				Provision: p.provision,
			}
		}
		return backends.Availability{
			Available: true,
			Detail:    fmt.Sprintf("%s resolved at %s", p.program, resolved),
			Provision: p.provision,
		}
	}
	return backends.Availability{
		Available: false,
		Detail:    fmt.Sprintf("%s not found on PATH", p.program),
		Provision: p.provision,
	}
}

func defaultCheckPythonModules(ctx context.Context, python string, modules []string) error {
	script := "import " + strings.Join(modules, ", ")
	cmd := exec.CommandContext(ctx, python, "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Execute resolves the model directory, builds the argv, and runs the program.
// The output is written to req.Output.LocalPath by contract; Execute returns
// that path as the result ref.
func (p *execProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q requires a local output path", p.name)
	}
	modelDir := req.ModelDir
	if modelDir == "" {
		modelDir = modelDirFor(req.Model.ID)
	}
	args, err := p.build(req, modelDir)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q build args: %w", p.name, err)
	}
	run := p.run
	if run == nil {
		run = defaultRun
	}
	if err := run(ctx, p.program, args); err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q execution failed: %w", p.name, err)
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name}}, nil
}

func defaultRun(ctx context.Context, name string, args []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %w (%s)", name, args, err, string(out))
	}
	return nil
}

func defaultRunOutput(ctx context.Context, name string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w (%s)", name, args, err, string(out))
	}
	return out, nil
}

// modelDirFor is the on-disk directory a model's weights live under, relative to
// the models root the management layer (IMG-P0-007, Phase 4) populates. The
// arg-builders reference files inside it. Keeping the id-relative segment here
// (not an absolute path) keeps providers path-policy-agnostic.
func modelDirFor(modelID string) string { return "models/" + modelID }

// =============================================================================
// Per-backend argument builders. Each mirrors the documented CLI of a real
// standalone backend; they are pure and unit-tested (arg assembly), while live
// execution is gated on the program + model being installed.
// =============================================================================

func in0(req backends.Request) (string, error) {
	if len(req.InputKeys) == 0 || req.InputKeys[0] == "" {
		return "", fmt.Errorf("missing input image")
	}
	return req.InputKeys[0], nil
}

func maskPath(req backends.Request) (string, error) {
	if len(req.InputKeys) < 2 || req.InputKeys[1] == "" {
		return "", fmt.Errorf("missing mask image")
	}
	return req.InputKeys[1], nil
}

func intParam(req backends.Request, key string, def int) int {
	if v, ok := req.Params[key]; ok {
		if n, err := strconv.Atoi(v); err == nil && n != 0 {
			return n
		}
	}
	return def
}

func strParam(req backends.Request, key string) string { return req.Params[key] }

// buildStableDiffusionCpp assembles a stable-diffusion.cpp (`sd`) invocation for
// text_to_image / image_to_image. Shape: sd -m <model> -p <prompt> [-n <neg>]
// [--cfg-scale x] --steps n -W w -H h [-s seed] [-M img2img -i in --strength x] -o out.
func buildStableDiffusionCpp(req backends.Request, modelDir string) ([]string, error) {
	args := []string{"-m", modelDir, "-o", req.Output.LocalPath, "-p", strParam(req, "prompt")}
	if neg := strParam(req, "negative_prompt"); neg != "" {
		args = append(args, "-n", neg)
	}
	if cfg := strParam(req, "cfg_scale"); cfg != "" {
		args = append(args, "--cfg-scale", cfg)
	}
	args = append(args, "--steps", strconv.Itoa(intParam(req, "steps", 20)))
	args = append(args, "-W", strconv.Itoa(intParam(req, "width", 512)))
	args = append(args, "-H", strconv.Itoa(intParam(req, "height", 512)))
	if seed := strParam(req, "seed"); seed != "" {
		args = append(args, "-s", seed)
	}
	if req.Operation == "image_to_image" {
		in, err := in0(req)
		if err != nil {
			return nil, err
		}
		args = append(args, "-M", "img2img", "-i", in)
		if s := strParam(req, "strength"); s != "" {
			args = append(args, "--strength", s)
		}
	}
	return args, nil
}

// buildDiffusers dispatches a diffusers python-sidecar invocation by operation:
// inpaint (masked regenerate) and edit_instruct (whole-image instruction edit)
// share the diffusers backend but invoke different sidecar modules with
// different argv shapes.
func buildDiffusers(req backends.Request, modelDir string) ([]string, error) {
	switch req.Operation {
	case "inpaint", "outpaint", "background_replace":
		// All three are masked-regenerate ops with the same argv shape: the mask
		// marks the region to synthesize (the hole for inpaint, the new border
		// region for outpaint, the background for background_replace) and the
		// prompt steers it. They share the inpaint sidecar module.
		return buildDiffusersInpaint(req, modelDir)
	case "edit_instruct":
		return buildDiffusersEditInstruct(req, modelDir)
	default:
		return nil, fmt.Errorf("diffusers: unsupported operation %q", req.Operation)
	}
}

// buildDiffusersInpaint assembles a diffusers python-sidecar inpaint invocation.
// Shape: python3 -m image_tools_sidecar.inpaint --model <dir> --image <in>
// --mask <mask> --prompt <p> --out <out>.
func buildDiffusersInpaint(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	mask, err := maskPath(req)
	if err != nil {
		return nil, err
	}
	return []string{
		"-m", "image_tools_sidecar.inpaint",
		"--model", modelDir,
		"--image", in,
		"--mask", mask,
		"--prompt", strParam(req, "prompt"),
		"--out", req.Output.LocalPath,
	}, nil
}

// buildDiffusersEditInstruct assembles a diffusers instruction-edit invocation
// (InstructPix2Pix / Qwen-Image-Edit class). The op is identity-preserving and
// prompt-only: there is no mask, and `prompt` is the natural-language
// instruction ("make it winter", "add sunglasses"). cfg_scale maps to the text
// guidance scale; image_guidance (how faithful to the source) defaults inside
// the sidecar but can be overridden via the `strength` param.
// Shape: python3 -m image_tools_sidecar.edit_instruct --model <dir> --image <in>
// --prompt <instruction> --out <out> [--steps n] [--guidance x] [--image-guidance x] [--seed s].
func buildDiffusersEditInstruct(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	args := []string{
		"-m", "image_tools_sidecar.edit_instruct",
		"--model", modelDir,
		"--image", in,
		"--prompt", strParam(req, "prompt"),
		"--out", req.Output.LocalPath,
		"--steps", strconv.Itoa(intParam(req, "steps", 20)),
	}
	if g := strParam(req, "cfg_scale"); g != "" {
		args = append(args, "--guidance", g)
	}
	if ig := strParam(req, "strength"); ig != "" {
		args = append(args, "--image-guidance", ig)
	}
	if seed := strParam(req, "seed"); seed != "" {
		args = append(args, "--seed", seed)
	}
	return args, nil
}

// buildIopaint assembles an iopaint object-removal invocation.
// Shape: iopaint run --model <dir> --device <cpu|cuda> --image <in> --mask <mask> --output <out>.
func buildIopaint(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	mask, err := maskPath(req)
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

// buildRealesrgan assembles a realesrgan-ncnn-vulkan upscale invocation.
// Shape: realesrgan-ncnn-vulkan -i <in> -o <out> -s <scale> -m <model-dir> -n <model-id>.
func buildRealesrgan(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	return []string{
		"-i", in,
		"-o", req.Output.LocalPath,
		"-s", strconv.Itoa(intParam(req, "scale", 4)),
		"-m", modelDir,
		"-n", req.Model.ID,
	}, nil
}

// buildRembg assembles a rembg background-removal invocation.
// Shape: rembg i -m <model-id> <in> <out>.
func buildRembg(req backends.Request, _ string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	return []string{"i", "-m", req.Model.ID, in, req.Output.LocalPath}, nil
}

// buildLlamaCppCaption assembles a llama.cpp multimodal caption invocation.
// Shape: llama-mtmd-cli -m <gguf> --mmproj <gguf> --image <in> -p <prompt> -n <tokens>.
func buildLlamaCppCaption(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	model, mmproj, err := llamaCppModelPaths(req, modelDir)
	if err != nil {
		return nil, err
	}
	prompt := strParam(req, "prompt")
	if prompt == "" {
		prompt = "Describe this image concisely."
	}
	args := []string{
		"-m", model,
		"--mmproj", mmproj,
		"--image", in,
		"-p", prompt,
		"-n", strconv.Itoa(intParam(req, "max_tokens", 96)),
	}
	if temp := strParam(req, "temperature"); temp != "" {
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

// buildOnnxSidecar assembles an invocation of the in-repo onnxruntime python
// sidecar (sidecar/image_tools_sidecar). The sidecar is the CPU-tractable,
// provisionable backend: onnxruntime + Pillow run real ONNX weights with no
// GPU. One dispatch per supported op:
//
//	background_removal: python3 -m image_tools_sidecar.bg_removal --model <onnx> --image <in> --out <out>
//	denoise:            python3 -m image_tools_sidecar.denoise    --model <onnx> --image <in> --out <out>
//
// The model argument is the resolved ONNX weight file inside the model dir
// (from the registry asset list), not the directory, so the sidecar loads it
// directly.
func buildOnnxSidecar(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	model := onnxModelPath(req, modelDir)
	var module string
	switch req.Operation {
	case "background_removal":
		module = "image_tools_sidecar.bg_removal"
	case "denoise":
		module = "image_tools_sidecar.denoise"
	case "colorize":
		module = "image_tools_sidecar.colorize"
	case "depth_map":
		module = "image_tools_sidecar.depth"
	default:
		return nil, fmt.Errorf("onnxruntime sidecar: unsupported operation %q", req.Operation)
	}
	return []string{
		"-m", module,
		"--model", model,
		"--image", in,
		"--out", req.Output.LocalPath,
	}, nil
}

// onnxModelPath resolves the ONNX weight file the sidecar should load: the first
// registry asset of kind ONNX (else the first asset) under the model dir. Falls
// back to the model dir itself when the model declares no assets.
func onnxModelPath(req backends.Request, modelDir string) string {
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

// providerSpec declares one standalone backend provider, keyed by the registry
// `backend` name so the selector's match-by-backend path lines up.
type providerSpec struct {
	name       string
	program    string // binary/interpreter resolved on PATH
	ops        []string
	build      argBuilder
	gpuCapable bool // backend can use the GPU (false for the CPU-only onnx sidecar)
	provision  string
	imports    []string
}

const llamaCppProvision = "install llama.cpp's multimodal runner (llama-mtmd-cli or compatible llama-cli) through Scenario Dependency Analyzer; see docs/reference/backends.md"

type llamaCppProvider struct {
	lookPath lookPathFunc
	run      outputRunner
}

func (p *llamaCppProvider) Name() string                       { return "llama.cpp" }
func (p *llamaCppProvider) Operations() []string               { return []string{"caption"} }
func (p *llamaCppProvider) Standalone() bool                   { return true }
func (p *llamaCppProvider) IsCloud() bool                      { return false }
func (p *llamaCppProvider) GPUCapable() bool                   { return true }
func (p *llamaCppProvider) Available(ctx context.Context) bool { return p.Availability(ctx).Available }
func (p *llamaCppProvider) Availability(context.Context) backends.Availability {
	program, err := p.resolveProgram()
	if err != nil {
		return backends.Availability{
			Available: false,
			Detail:    err.Error(),
			Provision: llamaCppProvision,
		}
	}
	return backends.Availability{
		Available: true,
		Detail:    fmt.Sprintf("llama.cpp multimodal runner resolved at %s", program),
		Provision: llamaCppProvision,
	}
}

func (p *llamaCppProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q requires a local output path", p.Name())
	}
	modelDir := req.ModelDir
	if modelDir == "" {
		modelDir = modelDirFor(req.Model.ID)
	}
	args, err := buildLlamaCppCaption(req, modelDir)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q build args: %w", p.Name(), err)
	}
	program, err := p.resolveProgram()
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q unavailable: %w", p.Name(), err)
	}
	run := p.run
	if run == nil {
		run = defaultRunOutput
	}
	out, err := run(ctx, program, args)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q execution failed: %w", p.Name(), err)
	}
	caption := strings.TrimSpace(string(out))
	if caption == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q produced an empty caption", p.Name())
	}
	payload := map[string]string{
		"backend": p.Name(),
		"model":   req.Model.ID,
		"caption": caption,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: encode caption output: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(req.Output.LocalPath, data, 0o644); err != nil {
		return backends.Result{}, fmt.Errorf("ai: write caption output: %w", err)
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.Name()}}, nil
}

func (p *llamaCppProvider) resolveProgram() (string, error) {
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	for _, candidate := range []string{"llama-mtmd-cli", "llama-cli"} {
		resolved, err := lookPath(candidate)
		if err == nil {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("llama-mtmd-cli/llama-cli not found on PATH")
}

func providerSpecs() []providerSpec {
	return []providerSpec{
		{name: "stable-diffusion.cpp", program: "sd", ops: []string{"text_to_image", "image_to_image"}, build: buildStableDiffusionCpp, gpuCapable: true, provision: "install stable-diffusion.cpp's sd binary via Scenario Dependency Analyzer; see docs/reference/backends.md"},
		{name: "diffusers", program: "python3", ops: []string{"inpaint", "outpaint", "background_replace", "edit_instruct"}, build: buildDiffusers, gpuCapable: true, provision: "install the embedded Python sidecar runtime dependencies via Scenario Dependency Analyzer; see docs/reference/backends.md", imports: []string{"diffusers", "torch", "PIL"}},
		{name: "iopaint", program: "iopaint", ops: []string{"object_removal"}, build: buildIopaint, gpuCapable: true, provision: "install the iopaint CLI via Scenario Dependency Analyzer; see docs/reference/backends.md"},
		{name: "realesrgan-ncnn-vulkan", program: "realesrgan-ncnn-vulkan", ops: []string{"upscale"}, build: buildRealesrgan, gpuCapable: true, provision: "install the realesrgan-ncnn-vulkan binary via Scenario Dependency Analyzer; see docs/reference/backends.md"},
		{name: "rembg", program: "rembg", ops: []string{"background_removal"}, build: buildRembg, gpuCapable: false, provision: "install the rembg CLI via Scenario Dependency Analyzer; see docs/reference/backends.md"},
		{name: "onnxruntime", program: "python3", ops: []string{"denoise", "background_removal", "colorize", "depth_map"}, build: buildOnnxSidecar, gpuCapable: false, provision: "ensure python3 plus onnxruntime/Pillow/numpy for the embedded sidecar; see docs/reference/backends.md", imports: []string{"onnxruntime", "PIL", "numpy"}},
	}
}

// RegisterProviders registers the standalone AI backends on reg. Pass nil
// lookPath/run to use the real os/exec implementations.
func RegisterProviders(reg *backends.Registry, lookPath lookPathFunc, run commandRunner) error {
	for _, s := range providerSpecs() {
		p := &execProvider{name: s.name, program: s.program, ops: s.ops, build: s.build, gpuCapable: s.gpuCapable, provision: s.provision, imports: s.imports, lookPath: lookPath, run: run}
		if err := reg.Register(p); err != nil {
			return fmt.Errorf("ai: register provider %q: %w", s.name, err)
		}
	}
	if err := reg.Register(&llamaCppProvider{lookPath: lookPath}); err != nil {
		return fmt.Errorf("ai: register provider %q: %w", "llama.cpp", err)
	}
	// In-process builtin providers (deterministic, no PATH/weights dependency).
	for _, p := range builtinProviderSpecs() {
		if err := reg.Register(p); err != nil {
			return fmt.Errorf("ai: register builtin provider %q: %w", p.Name(), err)
		}
	}
	return nil
}
