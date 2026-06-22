package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

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

// onnxProviderChecker reports the ONNX Runtime execution providers importable
// from the configured Python executable.
type onnxProviderChecker func(ctx context.Context, python string) ([]string, error)

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
	gpuCapable bool // can this backend (the TYPE) use the GPU? (CPU-only sidecars: false)
	provision  string
	imports    []string
	lookPath   lookPathFunc
	checkPy    pythonModuleChecker
	checkONNX  onnxProviderChecker
	run        commandRunner
	warm       warmRunner
	// gpuProbe, when set, reports whether the INSTALLED binary actually has a GPU
	// compute backend compiled in (a prebuilt release may be CPU-only even though
	// the backend type can use a GPU). gpuCapable gates the type; gpuProbe gates
	// the install. Result is cached via gpuProbeOnce so selection stays cheap.
	gpuProbe      func(ctx context.Context, program string) bool
	gpuProbeOnce  sync.Once
	gpuProbeValue bool
	// progressScan, when set, parses a single line/fragment of the backend's
	// stdout/stderr into execution progress. Returning ok=true streams a
	// Request.Progress update; the long-running run no longer sits at a static
	// percent. Only backends that emit parseable progress (sd-cli) set it.
	progressScan func(line string) (frac float64, message string, ok bool)
	// stream, when set, runs the command while feeding each output fragment to a
	// callback (for progressScan). Defaults to defaultStreamRun. Injected in tests.
	stream streamRunner
}

func (p *execProvider) Name() string         { return p.name }
func (p *execProvider) Operations() []string { return append([]string(nil), p.ops...) }
func (p *execProvider) Standalone() bool     { return true } // none of these are ComfyUI
func (p *execProvider) IsCloud() bool        { return false }

// GPUCapable reports whether this backend can run on the GPU. The onnxruntime
// sidecar is bound to CPUExecutionProvider, so it returns false and the selector
// labels its runs local-cpu even on a GPU host (honest tier reporting).
//
// gpuCapable is the static capability of the backend TYPE. When a gpuProbe is
// set, the INSTALLED binary is also consulted (cached): a prebuilt
// stable-diffusion.cpp release with no CUDA/Vulkan backend compiled in runs on
// the CPU regardless of a GPU on the host, so claiming local-gpu would be a lie
// (the user sees "Running on your GPU" while the binary reports VRAM 0.00MB).
func (p *execProvider) GPUCapable() bool {
	if !p.gpuCapable {
		return false
	}
	if p.gpuProbe == nil {
		return true
	}
	p.gpuProbeOnce.Do(func() {
		program := p.program
		if p.lookPath != nil {
			if resolved, err := p.lookPath(p.program); err == nil {
				program = resolved
			} else {
				return // not installed → leave gpuProbeValue false
			}
		}
		p.gpuProbeValue = p.gpuProbe(context.Background(), program)
	})
	return p.gpuProbeValue
}

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
			if p.name == "onnxruntime" {
				providers, err := p.onnxRuntimeProviders(ctx, resolved)
				if err != nil {
					return backends.Availability{
						Available: false,
						Detail:    fmt.Sprintf("%s resolved at %s, but ONNX Runtime provider probe failed: %v", p.program, resolved, err),
						Provision: p.provision,
					}
				}
				return backends.Availability{
					Available: onnxProviderAvailable(providers, "CPUExecutionProvider"),
					Detail:    p.onnxRuntimeProviderDetail(resolved, providers),
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

func (p *execProvider) onnxRuntimeProviders(ctx context.Context, python string) ([]string, error) {
	check := p.checkONNX
	if check == nil {
		check = defaultCheckONNXRuntimeProviders
	}
	return check(ctx, python)
}

func (p *execProvider) onnxRuntimeProviderDetail(python string, providers []string) string {
	detail := fmt.Sprintf("%s resolved at %s; Python imports ready: %s; ONNX Runtime providers: %s", p.program, python, strings.Join(p.imports, ","), strings.Join(providers, ","))
	if !onnxProviderAvailable(providers, "CPUExecutionProvider") {
		return detail + "; CPUExecutionProvider missing"
	}
	if !onnxProviderAvailable(providers, "CUDAExecutionProvider") {
		return detail + "; CUDAExecutionProvider unavailable, sidecar remains CPU-only"
	}
	return detail + "; CUDAExecutionProvider available, but this registered sidecar remains CPU-labeled until a GPU-capable ONNX provider row is promoted"
}

func onnxProviderAvailable(providers []string, want string) bool {
	for _, p := range providers {
		if p == want {
			return true
		}
	}
	return false
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

func defaultCheckONNXRuntimeProviders(ctx context.Context, python string) ([]string, error) {
	script := "import json, onnxruntime as ort; print(json.dumps(ort.get_available_providers()))"
	cmd := exec.CommandContext(ctx, python, "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	var providers []string
	if err := json.Unmarshal(out, &providers); err != nil {
		return nil, fmt.Errorf("decode onnxruntime providers: %w", err)
	}
	return providers, nil
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
	if p.warm != nil {
		if err := p.warm.Run(ctx, p.program, args); err == nil {
			return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name, "runner": "warm"}}, nil
		}
	}
	// Stream when the caller wants progress and this backend can parse its own
	// output. Otherwise fall back to the simple run-to-completion path so the
	// long sampling loop no longer leaves the job frozen at a static percent.
	if req.Progress != nil && p.progressScan != nil {
		stream := p.stream
		if stream == nil {
			stream = defaultStreamRun
		}
		onLine := func(line string) {
			if frac, msg, ok := p.progressScan(line); ok {
				req.Progress(frac, msg)
			}
		}
		if err := stream(ctx, p.program, args, onLine); err != nil {
			return backends.Result{}, fmt.Errorf("ai: backend %q execution failed: %w", p.name, err)
		}
		return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name, "runner": "stream"}}, nil
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

// streamRunner runs a command, invoking onLine for each output fragment as it is
// produced (split on \r and \n, since progress bars redraw with \r). It still
// returns a non-nil error carrying captured output on a non-zero exit, matching
// defaultRun's error shape.
type streamRunner func(ctx context.Context, name string, args []string, onLine func(line string)) error

// defaultStreamRun is the real streamRunner. It merges stdout+stderr, scans
// fragments live (so per-step progress surfaces immediately rather than buffering
// until a newline), and accumulates everything for the failure message.
func defaultStreamRun(ctx context.Context, name string, args []string, onLine func(line string)) error {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%s: stdout pipe: %w", name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("%s: stderr pipe: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: start: %w", name, err)
	}

	var (
		mu  sync.Mutex
		buf strings.Builder
		wg  sync.WaitGroup
	)
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		sc.Split(scanLinesOrCR)
		for sc.Scan() {
			frag := sc.Text()
			mu.Lock()
			buf.WriteString(frag)
			buf.WriteByte('\n')
			mu.Unlock()
			if onLine != nil {
				onLine(frag)
			}
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s %v: %w (%s)", name, args, err, buf.String())
	}
	return nil
}

// scanLinesOrCR is a bufio.SplitFunc that breaks on either \n or \r, so a
// carriage-return-redrawn progress bar (e.g. "|====| 3/20\r") yields a fragment
// per redraw instead of buffering the whole line until the next \n.
func scanLinesOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// sdSamplerLine matches the stable-diffusion.cpp SAMPLING progress bar, which is
// the long phase users wait on. It is distinguished from the model-load /
// vae-decode byte bars (e.g. "|####| 196/196 - 2.28GB/s") by the per-iteration
// "s/it" or "it/s" rate suffix; only the sampler reports iterations.
var sdSamplerLine = regexp.MustCompile(`\|[#=> ]*\|\s*(\d+)/(\d+)\s*-\s*[0-9.]+\s*(?:s/it|it/s)`)

// scanStableDiffusionProgress parses one sd-cli output fragment into sampling
// progress. Returns ok=false for any non-sampler line so the caller streams a
// Progress update only on real forward motion.
func scanStableDiffusionProgress(line string) (frac float64, message string, ok bool) {
	m := sdSamplerLine.FindStringSubmatch(line)
	if m == nil {
		return 0, "", false
	}
	cur, err1 := strconv.Atoi(m[1])
	total, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || total <= 0 {
		return 0, "", false
	}
	if cur > total {
		cur = total
	}
	return float64(cur) / float64(total), fmt.Sprintf("sampling step %d/%d", cur, total), true
}

// probeStableDiffusionGPU reports whether the installed sd-cli has a GPU compute
// backend compiled in. stable-diffusion.cpp validates the --backend device
// assignment BEFORE loading the model, so pointing it at a non-existent model is
// a ~millisecond probe: a CPU-only build prints "backend 'cuda0' was not found"
// and exits; a GPU build gets past backend config to the model-load failure
// (which never mentions the device). A short timeout bounds the worst case.
func probeStableDiffusionGPU(ctx context.Context, program string) bool {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	missing := filepath.Join(os.TempDir(), "imgtools-gpu-probe-nonexistent.safetensors")
	out := filepath.Join(os.TempDir(), "imgtools-gpu-probe.png")
	for _, dev := range []string{"cuda0", "vulkan0"} {
		cmd := exec.CommandContext(ctx, program, "--backend", dev, "-M", "img_gen",
			"-m", missing, "-o", out, "-p", "probe", "--steps", "1")
		combined, _ := cmd.CombinedOutput()
		if !strings.Contains(string(combined), fmt.Sprintf("backend '%s' was not found", dev)) {
			return true // device backend is compiled in
		}
	}
	return false
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

// buildStableDiffusionCpp assembles a stable-diffusion.cpp `sd-cli` invocation
// for text_to_image / image_to_image. Shape: sd-cli -m <model> -p <prompt>
// [-n <neg>] [--cfg-scale x] --steps n -W w -H h [-s seed] [-i in --strength x]
// -o out. image_to_image is selected by passing -i <init>; the run mode stays
// the default img_gen (the new -M flag's values are img_gen|vid_gen|upscale|
// convert|metadata, so the old `-M img2img` is gone).
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

func buildStableDiffusionCpp(req backends.Request, modelDir string) ([]string, error) {
	args := []string{"-m", sdModelArg(modelDir), "-o", req.Output.LocalPath, "-p", strParam(req, "prompt")}
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
		args = append(args, "-i", in)
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

// buildRealesrgan assembles a realesrgan-ncnn-vulkan invocation. The prebuilt
// release ships its own `models/` directory (realesr-animevideov3 at -s 2/3/4
// and realesrgan-x4plus), which the binary resolves relative to its own
// location — so we pass a bundled model NAME via -n and let the tool find the
// weights, rather than a -m model dir or the image-tools model id (neither of
// which holds the ncnn .param/.bin files). realesr-animevideov3 is the
// variable-scale general model used for both upscale and denoise.
// Shape: realesrgan-ncnn-vulkan -i <in> -o <out> -s <scale> -n <bundled-model>.
func buildRealesrgan(req backends.Request, _ string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	scale := intParam(req, "scale", 4)
	if scale < 2 || scale > 4 {
		scale = 4
	}
	return []string{
		"-i", in,
		"-o", req.Output.LocalPath,
		"-s", strconv.Itoa(scale),
		"-n", "realesr-animevideov3",
	}, nil
}

// buildRembg assembles a rembg invocation. background_removal writes an RGBA
// cutout. background_replace asks rembg to composite the cutout over a caller-
// supplied background_color (R,G,B or R,G,B,A) so replacement requests do not
// silently degrade to removal-only output.
// Shape: rembg i -m <model-id> [--bgcolor r,g,b,a] <in> <out>.
func buildRembg(req backends.Request, _ string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	args := []string{"i", "-m", req.Model.ID}
	if req.Operation == "background_replace" {
		color := strings.TrimSpace(strParam(req, "background_color"))
		if color == "" {
			return nil, fmt.Errorf("rembg background_replace requires background_color")
		}
		args = append(args, "--bgcolor", color)
	}
	args = append(args, in, req.Output.LocalPath)
	return args, nil
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
//	deblur:             python3 -m image_tools_sidecar.deblur     --model <onnx> --image <in> --out <out>
//	object_detection:   python3 -m image_tools_sidecar.detect     --model <onnx> --image <in> --out <out>
//	segment:            python3 -m image_tools_sidecar.segment    --model <dir|onnx> --image <in> --out <out>
//	tagging:            python3 -m image_tools_sidecar.tagging    --model <onnx> --image <in> --out <out>
//	nsfw_classify:      python3 -m image_tools_sidecar.nsfw       --model <onnx> --image <in> --out <out>
//	embedding:          python3 -m image_tools_sidecar.embedding  --model <onnx> --image <in> --out <out>
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
	case "deblur":
		module = "image_tools_sidecar.deblur"
	case "colorize":
		module = "image_tools_sidecar.colorize"
	case "depth_map":
		module = "image_tools_sidecar.depth"
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

// buildPythonSidecar assembles invocations for enabled catalog entries whose
// native backend family is `python-sidecar`. These are heavier Python modules
// than the ONNX CPU floor, but they still use the embedded sidecar package and
// the same host-provisioned Python runtime seam.
func buildPythonSidecar(req backends.Request, modelDir string) ([]string, error) {
	in, err := in0(req)
	if err != nil {
		return nil, err
	}
	var module string
	switch req.Operation {
	case "colorize":
		module = "image_tools_sidecar.colorize"
	case "face_restore", "old_photo_restore":
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
	// hostTool is the platform host-tool NAME this backend depends on (the
	// cross-module contract: the scenario declares it in service.json hostTools
	// and the platform defines it in internal/tools/<hostTool>/tool.json). The
	// `provision` message and docs/reference/backends.md are DERIVED from it, so
	// there is one source of truth and no free-text drift.
	hostTool string
	// pipDeps are the Python packages a python-backed provider also needs beyond
	// the host `python` tool itself (surfaced honestly; not auto-fetched).
	pipDeps []string
	imports []string
}

// provision returns the derived remediation message for this provider: the exact
// `vrooli host install <tool>` command plus any pip dependencies. Never
// hand-written — see deriveProvision.
func (s providerSpec) provision() string { return deriveProvision(s.hostTool, s.pipDeps) }

// deriveProvision builds the single canonical provisioning message from a host
// tool name (+ optional pip deps). This is the one place the remediation command
// is spelled, so the runtime error, doctor output, and backends.md cannot drift.
func deriveProvision(hostTool string, pipDeps []string) string {
	msg := fmt.Sprintf("install the %q host tool — run `vrooli host install %s` (see docs/reference/backends.md)", hostTool, hostTool)
	if len(pipDeps) > 0 {
		msg += fmt.Sprintf("; this Python backend also needs pip packages: %s", strings.Join(pipDeps, ", "))
	}
	return msg
}

const llamaCppHostTool = "llama-cpp"

var llamaCppProvision = deriveProvision(llamaCppHostTool, nil)

// HostToolBinding records which platform host tool (and pip deps) a backend
// provider needs. It is the structured contract the conformance tests and the
// generated backends.md consume. Built from providerSpecs() so it cannot drift
// from the registered providers.
type HostToolBinding struct {
	Provider string
	HostTool string
	Ops      []string
	PipDeps  []string
}

// HostToolForProvider returns the host tool a backend provider depends on,
// matched by provider name. Reports false for in-process / cloud providers that
// need no host tool.
func HostToolForProvider(provider string) (string, bool) {
	for _, b := range HostToolBindings() {
		if b.Provider == provider && b.HostTool != "" {
			return b.HostTool, true
		}
	}
	return "", false
}

// HostToolForOperation returns the host tool of the backend that serves an
// operation, matched against the provider bindings. Reports false when no
// host-tool-backed provider serves the op.
func HostToolForOperation(operation string) (string, bool) {
	for _, b := range HostToolBindings() {
		for _, op := range b.Ops {
			if op == operation && b.HostTool != "" {
				return b.HostTool, true
			}
		}
	}
	return "", false
}

// HostToolBindings returns the provider→host-tool bindings for every standalone
// backend that depends on a host tool (including llama.cpp). In-process
// providers (builtin/computed/library-go) need no host tool and are omitted.
func HostToolBindings() []HostToolBinding {
	specs := providerSpecs()
	out := make([]HostToolBinding, 0, len(specs)+1)
	for _, s := range specs {
		out = append(out, HostToolBinding{Provider: s.name, HostTool: s.hostTool, Ops: append([]string(nil), s.ops...), PipDeps: append([]string(nil), s.pipDeps...)})
	}
	out = append(out, HostToolBinding{Provider: "llama.cpp", HostTool: llamaCppHostTool, Ops: []string{"caption"}})
	return out
}

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
		{name: "stable-diffusion.cpp", program: "sd", ops: []string{"text_to_image", "image_to_image"}, build: buildStableDiffusionCpp, gpuCapable: true, hostTool: "sd"},
		{name: "diffusers", program: "python3", ops: []string{"inpaint", "outpaint", "background_replace", "edit_instruct"}, build: buildDiffusers, gpuCapable: true, hostTool: "python", pipDeps: []string{"diffusers", "torch", "Pillow"}, imports: []string{"diffusers", "torch", "PIL"}},
		{name: "iopaint", program: "iopaint", ops: []string{"object_removal"}, build: buildIopaint, gpuCapable: true, hostTool: "iopaint"},
		{name: "realesrgan-ncnn-vulkan", program: "realesrgan-ncnn-vulkan", ops: []string{"upscale", "denoise"}, build: buildRealesrgan, gpuCapable: true, hostTool: "realesrgan-ncnn-vulkan"},
		{name: "rembg", program: "rembg", ops: []string{"background_removal", "background_replace"}, build: buildRembg, gpuCapable: false, hostTool: "rembg"},
		{name: "onnxruntime", program: "python3", ops: []string{"denoise", "deblur", "background_removal", "colorize", "depth_map", "object_detection", "segment", "tagging", "nsfw_classify", "embedding"}, build: buildOnnxSidecar, gpuCapable: false, hostTool: "python", pipDeps: []string{"onnxruntime", "Pillow", "numpy"}, imports: []string{"onnxruntime", "PIL", "numpy"}},
		{name: "python-sidecar", program: "python3", ops: []string{"colorize"}, build: buildPythonSidecar, gpuCapable: false, hostTool: "python", pipDeps: []string{"onnxruntime", "Pillow", "numpy"}, imports: []string{"onnxruntime", "PIL", "numpy"}},
		{name: "python-sidecar", program: "python3", ops: []string{"face_restore", "old_photo_restore"}, build: buildPythonSidecar, gpuCapable: false, hostTool: "python", pipDeps: []string{"torch", "basicsr", "facexlib", "Pillow", "numpy"}, imports: []string{"torch", "basicsr", "facexlib", "PIL", "numpy"}},
	}
}

// RegisterProviders registers the standalone AI backends on reg. Pass nil
// lookPath/run to use the real os/exec implementations.
func RegisterProviders(reg *backends.Registry, lookPath lookPathFunc, run commandRunner) error {
	for _, s := range providerSpecs() {
		p := &execProvider{name: s.name, program: s.program, ops: s.ops, build: s.build, gpuCapable: s.gpuCapable, provision: s.provision(), imports: s.imports, lookPath: lookPath, run: run}
		if s.name == "onnxruntime" && run == nil {
			p.warm = newWarmSidecarRunner()
		}
		if s.name == "stable-diffusion.cpp" {
			// The installed sd-cli may be a CPU-only prebuilt; probe the real binary
			// so the tier is honest, and parse its sampler bar so the long run shows
			// live progress instead of a frozen percent.
			p.gpuProbe = probeStableDiffusionGPU
			p.progressScan = scanStableDiffusionProgress
		}
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
