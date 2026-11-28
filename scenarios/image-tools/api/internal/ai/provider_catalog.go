package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	aigateway "image-tools/internal/aigateway"
	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"
)

// providerSpec declares one standalone backend provider, keyed by the registry
// `backend` name so the selector's match-by-backend path lines up. Its techniques
// are the per-operation arg-builders (internal/technique) this backend exposes.
type providerSpec struct {
	name           string
	program        string                // binary/interpreter resolved on PATH
	programAliases []string              // preferred GPU-build commands tried before program (e.g. "sd-gpu")
	techniques     []technique.Technique // per-op technique rows this backend serves
	gpuCapable     bool                  // backend can use the GPU (a per-operation allow-list can narrow it, see gpuQualifiedOps)
	// hostTool is the platform host-tool NAME this backend depends on (the
	// cross-module contract: the scenario declares it in service.json hostTools
	// and the platform defines it in internal/tools/<hostTool>/tool.json). The
	// `provision` message and docs/reference/backends.md are DERIVED from it, so
	// there is one source of truth and no free-text drift.
	hostTool string
	imports  []string
	// gpuProbe selects the runtime probe that proves accelerator use: "torch",
	// "onnx", "sd", or "" (no accelerator claim). Data, not a name comparison in
	// RegisterProviders.
	gpuProbe string
	// gpuQualifiedOps, when set, limits the GPU claim to operations whose CUDA
	// path has been qualified on this backend; every other op runs on CPU even
	// when the backend can reach the GPU. Nil means every served op may use it.
	gpuQualifiedOps []string
	// cpuRetryAfterOOM lets a GPU run that dies of GPU memory exhaustion repeat
	// once on the CPU in the same job (the card is shared, so another service can
	// take the memory a lease promised). Only for backends whose CPU path is a
	// sane substitute: a diffusion model on the CPU is not a retry, it is hours.
	cpuRetryAfterOOM bool
	// warmSidecar keeps a bounded persistent JSONL worker for this backend.
	warmSidecar bool
	// progressScan selects a stdout progress parser ("sd" for stable-diffusion.cpp).
	progressScan string
}

// ops returns the operations this provider serves, in technique-declaration
// order (so the derived host-tool matrix / backends.md stays stable).
func (s providerSpec) ops() []string {
	ops := make([]string, 0, len(s.techniques))
	for _, t := range s.techniques {
		ops = append(ops, t.Op)
	}
	return ops
}

// provision returns the derived remediation message for this provider: the exact
// `vrooli host install <tool>` command. Never hand-written — see deriveProvision.
func (s providerSpec) provision() string { return deriveProvision(s.hostTool) }

// isPython reports whether this backend is served by the Python interpreter (and
// therefore runs from the scenario's private uv venv when one is provisioned).
func (s providerSpec) isPython() bool { return s.program == pythonProgram }

// deriveProvision builds the single canonical provisioning message from a host
// tool name. This is the one place the remediation command is spelled, so the
// runtime error, doctor output, and backends.md cannot drift. Python backends
// bind to the `uv` host tool: installing uv lets image-tools build its private,
// lock-pinned venv (internal/pydeps/requirements.lock) on the next start — there
// is no separate "pip install …" step, so no parallel dependency list to drift.
func deriveProvision(hostTool string) string {
	return fmt.Sprintf("install the %q host tool — run `vrooli host install %s` (see docs/reference/backends.md)", hostTool, hostTool)
}

const llamaCppHostTool = "llama-cpp"

var llamaCppProvision = deriveProvision(llamaCppHostTool)

// HostToolBinding records which platform host tool (and pip deps) a backend
// provider needs. It is the structured contract the conformance tests and the
// generated backends.md consume. Built from providerSpecs() so it cannot drift
// from the registered providers.
type HostToolBinding struct {
	Provider string
	HostTool string
	Ops      []string
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

// HostToolForOperation returns the host tool an operator must ensure for an
// operation, matched against the provider bindings. An operation several backends
// serve (caption: llama.cpp and the Florence-2 worker) resolves to the external
// tool, because "uv" is the scenario's own runtime and is always provisioned; the
// first binding wins among equals. Reports false when no host-tool-backed provider
// serves the op.
func HostToolForOperation(operation string) (string, bool) {
	fallback := ""
	for _, b := range HostToolBindings() {
		for _, op := range b.Ops {
			if op != operation || b.HostTool == "" {
				continue
			}
			if b.HostTool != "uv" {
				return b.HostTool, true
			}
			if fallback == "" {
				fallback = b.HostTool
			}
		}
	}
	return fallback, fallback != ""
}

// HostToolBindings returns the provider→host-tool bindings for every standalone
// backend that depends on a host tool (including llama.cpp). In-process
// providers (builtin/computed/library-go) need no host tool and are omitted.
// HostToolBindings is the projection of the two tables that name host tools: the
// provider specs (a tool per inference backend) and the tool-only bindings below.
func HostToolBindings() []HostToolBinding {
	specs := providerSpecs()
	out := make([]HostToolBinding, 0, len(specs)+len(hostToolOnlyBindings))
	for _, s := range specs {
		out = append(out, HostToolBinding{Provider: s.name, HostTool: s.hostTool, Ops: s.ops()})
	}
	return append(out, hostToolOnlyBindings...)
}

// hostToolOnlyBindings cover operations that a host tool serves directly, with no
// inference technique row: the captioning runtime and the OCR engine are
// installed as tools and invoked by their own code path, so they have no
// providerSpec. They are data here rather than appends inside the projection so
// that adding one is a row, and so the conformance test cannot miss it.
var hostToolOnlyBindings = []HostToolBinding{
	{Provider: "llama.cpp", HostTool: llamaCppHostTool, Ops: []string{"caption"}},
	{Provider: "tesseract", HostTool: "tesseract", Ops: []string{"ocr"}},
}

// nativeTechnique is a thin constructor for a proven, native technique row: a
// stable name, the op it yields, and the pure arg-builder. PipelineClass/Caveat
// stay empty (no derived-op quality note); a later phase populates those for
// architecture-derived techniques.
func nativeTechnique(name, op string, build technique.ArgBuilder) technique.Technique {
	return technique.Technique{Name: name, Op: op, Ready: true, Build: build}
}

// conditioningTechnique is a native technique whose builder consumes the
// conditioning adapter stack (LoRA/ControlNet/IP-Adapter). Marking it Adapters
// lets backend selection route a conditioned request to this backend (the
// diffusers sidecar) rather than the model's default non-conditioning backend.
func conditioningTechnique(name, op string, build technique.ArgBuilder) technique.Technique {
	t := nativeTechnique(name, op, build)
	t.Adapters = true
	return t
}

func providerSpecs() []providerSpec {
	return []providerSpec{
		{
			name: "stable-diffusion.cpp", program: "sd", programAliases: []string{"sd-gpu"}, gpuCapable: true, hostTool: "sd", gpuProbe: "sd", progressScan: "sd",
			techniques: []technique.Technique{
				nativeTechnique("sd-txt2img", "text_to_image", technique.StableDiffusionCpp),
				nativeTechnique("sd-img2img", "image_to_image", technique.StableDiffusionCpp),
			},
		},
		{
			name: "diffusers", program: pythonProgram, gpuCapable: true, hostTool: "uv", gpuProbe: "torch", imports: []string{"diffusers", "torch", "PIL"},
			techniques: []technique.Technique{
				conditioningTechnique("diffusers-txt2img", "text_to_image", technique.DiffusersText2Image),
				conditioningTechnique("diffusers-img2img", "image_to_image", technique.DiffusersImg2Img),
				conditioningTechnique("diffusers-inpaint", "inpaint", technique.DiffusersInpaint),
				conditioningTechnique("diffusers-outpaint", "outpaint", technique.DiffusersInpaint),
				conditioningTechnique("diffusers-background-replace", "background_replace", technique.DiffusersInpaint),
				nativeTechnique("diffusers-edit-instruct", "edit_instruct", technique.DiffusersEditInstruct),
				// Garment replace is a masked inpaint driven by a prompt; the model
				// row declares which weights back it. See DECISIONS.md 2026-07-04 for
				// the capability policy. The dedicated face swappers are onnxruntime
				// models; a reference-editing family (FLUX.2 [klein]) also serves
				// face_swap as a general-model route, picked only by naming it.
				nativeTechnique("diffusers-face-swap", "face_swap", technique.DiffusersFaceSwapInstruct),
				conditioningTechnique("diffusers-garment-replace", "garment_replace", technique.DiffusersInpaint),
			},
		},
		{
			name: "iopaint", program: "iopaint", gpuCapable: true, hostTool: "iopaint",
			techniques: []technique.Technique{
				nativeTechnique("iopaint-object-removal", "object_removal", technique.Iopaint),
			},
		},
		{
			name: "realesrgan-ncnn-vulkan", program: "realesrgan-ncnn-vulkan", gpuCapable: true, hostTool: "realesrgan-ncnn-vulkan",
			techniques: []technique.Technique{
				nativeTechnique("realesrgan-upscale", "upscale", technique.Realesrgan),
				nativeTechnique("realesrgan-denoise", "denoise", technique.Realesrgan),
			},
		},
		{
			name: "rembg", program: "rembg", gpuCapable: false, hostTool: "rembg",
			techniques: []technique.Technique{
				nativeTechnique("rembg-background-removal", "background_removal", technique.Rembg),
				nativeTechnique("rembg-background-replace", "background_replace", technique.Rembg),
			},
		},
		{
			name: "onnxruntime", program: pythonProgram, gpuCapable: true, hostTool: "uv", gpuProbe: "onnx", warmSidecar: true, cpuRetryAfterOOM: true, imports: []string{"onnxruntime", "PIL", "numpy"},
			// CUDA qualified per op: background_removal and segment by the speedup
			// e2e; depth_map (Depth Anything V2 Small/Large) and face_restore
			// (RestoreFormer++/CodeFormer) by the 2026-11-21 licence-gated model
			// evaluation (same sidecar modules on CUDAExecutionProvider, 2.3GB VRAM).
			// colorize and denoise join them on the 2026-11-23 warm-session measurement
			// (same sidecar module, CPU 4 threads vs CUDA, steady state): ddcolor-large
			// 2.4-2.9 s vs 0.95-1.2 s, ddcolor-tiny 1.9-2.8 s vs 1.35-1.6 s, SCUNet on a
			// 768x1024 photo 18-20 s vs 1.0 s (it needs about 6 GB; a GPU run that runs
			// out of memory repeats on the CPU, cpuRetryAfterOOM). The small classifiers
			// (nsfw_classify, age_estimate, embedding, object_detection) gain 0.02-0.2 s,
			// less than a GPU lease and a held warm residency cost, so they stay on the
			// CPU. ocr stays on the CPU too: its detector pyramid needs 1.8 GB buffers and
			// failed on CUDA with 3-7 GB free. deblur stays: the NAFNet export is int8 and
			// ONNX Runtime's CUDA provider has no DequantizeLinear kernel for it.
			gpuQualifiedOps: []string{"background_removal", "segment", "depth_map", "face_restore", "face_swap", "colorize", "denoise"},
			techniques: []technique.Technique{
				nativeTechnique("onnx-denoise", "denoise", technique.OnnxSidecar),
				nativeTechnique("onnx-deblur", "deblur", technique.OnnxSidecar),
				nativeTechnique("onnx-background-removal", "background_removal", technique.OnnxSidecar),
				nativeTechnique("onnx-object-removal", "object_removal", technique.OnnxSidecar),
				nativeTechnique("onnx-colorize", "colorize", technique.OnnxSidecar),
				nativeTechnique("onnx-depth-map", "depth_map", technique.OnnxSidecar),
				nativeTechnique("onnx-face-restore", "face_restore", technique.OnnxSidecar),
				nativeTechnique("onnx-face-swap", "face_swap", technique.OnnxSidecar),
				nativeTechnique("onnx-pose-extract", "pose_extract", technique.OnnxSidecar),
				nativeTechnique("moge-scene-geometry", "scene_geometry", technique.OnnxSidecar),
				nativeTechnique("onnx-object-detection", "object_detection", technique.OnnxSidecar),
				nativeTechnique("onnx-segment", "segment", technique.OnnxSidecar),
				nativeTechnique("onnx-tagging", "tagging", technique.OnnxSidecar),
				nativeTechnique("onnx-nsfw-classify", "nsfw_classify", technique.OnnxSidecar),
				nativeTechnique("onnx-ocr", "ocr", technique.OnnxSidecar),
				nativeTechnique("onnx-age-estimate", "age_estimate", technique.OnnxSidecar),
				nativeTechnique("onnx-embedding", "embedding", technique.OnnxSidecar),
			},
		},
		{
			// One provider row for the python-sidecar backend. It previously appeared
			// twice with the same name (one row per dependency set), which made the
			// generated host-tool matrix ambiguous and split one backend's ops across
			// two registrations.
			name: "python-sidecar", program: pythonProgram, gpuCapable: false, hostTool: "uv",
			imports: []string{"onnxruntime", "torch", "basicsr", "facexlib", "PIL", "numpy"},
			techniques: []technique.Technique{
				nativeTechnique("python-sidecar-colorize", "colorize", technique.PythonSidecar),
				nativeTechnique("python-sidecar-old-photo-restore", "old_photo_restore", technique.PythonSidecar),
			},
		},
		{
			name: "sam2-transformers", program: pythonProgram, gpuCapable: true, hostTool: "uv", gpuProbe: "torch", warmSidecar: true, imports: []string{"torch", "transformers", "PIL"},
			techniques: []technique.Technique{
				nativeTechnique("sam2-transformers-segment", "segment", technique.SAM2Sidecar),
			},
		},
		{
			// SegFormer human/clothes parsers: a phrase ("hair", "the jacket") or a point
			// becomes the pixels of a parser label. Same torch + transformers runtime and
			// warm worker as SAM 2; a separate backend because the technique differs.
			name: "segformer-transformers", program: pythonProgram, gpuCapable: true, hostTool: "uv", gpuProbe: "torch", warmSidecar: true, imports: []string{"torch", "transformers", "PIL", "scipy"},
			techniques: []technique.Technique{
				nativeTechnique("segformer-semantic-segment", "segment", technique.SemanticSegmentSidecar),
			},
		},
		{
			// Florence-2: one pass yields a short caption, a paragraph caption and
			// labelled boxes. Transformers checkpoint (no remote code), warm worker like
			// SAM 2; the caption and detection techniques share the florence module.
			name: "florence-transformers", program: pythonProgram, gpuCapable: true, hostTool: "uv", gpuProbe: "torch", warmSidecar: true, imports: []string{"torch", "transformers", "PIL"},
			techniques: []technique.Technique{
				nativeTechnique("florence-caption", "caption", technique.FlorenceSidecar),
				nativeTechnique("florence-object-detection", "object_detection", technique.FlorenceSidecar),
			},
		},
		{
			name: "triposr", program: pythonProgram, gpuCapable: true, hostTool: "uv", gpuProbe: "torch", imports: []string{"torch", "einops", "trimesh", "skimage", "yaml"},
			techniques: []technique.Technique{
				nativeTechnique("triposr-image-to-object", "image_to_object", technique.TripoSRSidecar),
			},
		},
	}
}

// RegisterProviders registers the standalone AI backends on reg. Pass nil
// lookPath/run to use the real os/exec implementations.
//
// pythonInterpreter is the absolute path to the scenario's private uv venv
// interpreter; every Python backend invokes it directly (the isolation seam —
// see github.com/vrooli/pyenv-go and main.go boot wiring). Pass "" when the venv is not
// provisioned (uv absent or the background build is still running): Python
// backends then report unavailable (no bare-"python3" PATH fallback), surfaced
// before use via doctor/health/ready_state.
func RegisterProviders(reg *backends.Registry, lookPath lookPathFunc, run commandRunner, pythonInterpreter string) error {
	// Hardware presence alone does not prove the scenario's Torch wheel can use
	// CUDA. Python providers share a bounded, short-lived cached runtime probe
	// so native environment convergence can recover without another restart.
	var torchProbeMu sync.Mutex
	var torchProbeUntil time.Time
	var torchCUDA bool
	torchProbe := func(ctx context.Context, program string) bool {
		torchProbeMu.Lock()
		defer torchProbeMu.Unlock()
		if !time.Now().Before(torchProbeUntil) {
			torchCUDA = probeTorchCUDA(ctx, program)
			torchProbeUntil = time.Now().Add(30 * time.Second)
		}
		return torchCUDA
	}
	var onnxProbeMu sync.Mutex
	var onnxProbeUntil time.Time
	var onnxCUDA bool
	onnxProbe := func(ctx context.Context, program string) bool {
		onnxProbeMu.Lock()
		defer onnxProbeMu.Unlock()
		if !time.Now().Before(onnxProbeUntil) {
			onnxCUDA = probeONNXCUDA(ctx, program)
			onnxProbeUntil = time.Now().Add(30 * time.Second)
		}
		return onnxCUDA
	}
	for _, s := range providerSpecs() {
		p := &execProvider{backendProcess: backendProcess{name: s.name, program: s.program, programAliases: s.programAliases, gpuCapable: s.gpuCapable, provision: s.provision(), imports: s.imports, lookPath: lookPath, run: run}, techniques: technique.NewSet(s.techniques...), gpuOps: opSet(s.gpuQualifiedOps), cpuRetry: s.cpuRetryAfterOOM}
		if s.isPython() {
			p.pythonInterpreter = pythonInterpreter
			p.probe.TTL = 30 * time.Second
		}
		// Provider-specific behaviour is declared in the spec row, not selected by
		// comparing the provider name, so adding a backend is a data change.
		switch s.gpuProbe {
		case "torch":
			p.gpuProbe = torchProbe
		case "onnx":
			p.gpuProbe = onnxProbe
		case "sd":
			// The installed sd-cli may be a CPU-only prebuilt; probe the real binary
			// so the tier is honest.
			p.gpuProbe = probeStableDiffusionGPU
		}
		if s.warmSidecar && run == nil {
			p.warm = newWarmSidecarRunner()
		}
		if s.progressScan == "sd" {
			// Parse the sampler bar so a long run shows live progress instead of a
			// frozen percent.
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
	// BYOK cloud provider (last tier). The provider is registered unconditionally
	// so selection and readiness remain explicit, but all remote transport,
	// credentials, model policy, receipts, and retries belong to AI Gateway.
	vision := aigateway.NewVisionClient()
	// General vision models served by the gateway: the local rung (Ollama behind a
	// gateway role) and, below, the hosted rung that shares the BYOK provider.
	if err := reg.Register(aigateway.NewVisionProvider(vision)); err != nil {
		return fmt.Errorf("ai: register provider %q: %w", models.BackendGatewayVision, err)
	}
	if err := reg.Register(aigateway.NewProvider(aigateway.NewMediaClient(), vision)); err != nil {
		return fmt.Errorf("ai: register provider %q: %w", models.BackendOpenRouter, err)
	}
	return nil
}
