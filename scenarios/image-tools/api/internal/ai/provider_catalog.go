package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	gpuCapable     bool                  // backend can use the GPU (false for the CPU-only onnx sidecar)
	// hostTool is the platform host-tool NAME this backend depends on (the
	// cross-module contract: the scenario declares it in service.json hostTools
	// and the platform defines it in internal/tools/<hostTool>/tool.json). The
	// `provision` message and docs/reference/backends.md are DERIVED from it, so
	// there is one source of truth and no free-text drift.
	hostTool string
	imports  []string
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
		out = append(out, HostToolBinding{Provider: s.name, HostTool: s.hostTool, Ops: s.ops()})
	}
	out = append(out, HostToolBinding{Provider: "llama.cpp", HostTool: llamaCppHostTool, Ops: []string{"caption"}}, HostToolBinding{Provider: "tesseract", HostTool: "tesseract", Ops: []string{"ocr"}})
	return out
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
			name: "stable-diffusion.cpp", program: "sd", programAliases: []string{"sd-gpu"}, gpuCapable: true, hostTool: "sd",
			techniques: []technique.Technique{
				nativeTechnique("sd-txt2img", "text_to_image", technique.StableDiffusionCpp),
				nativeTechnique("sd-img2img", "image_to_image", technique.StableDiffusionCpp),
			},
		},
		{
			name: "diffusers", program: pythonProgram, gpuCapable: true, hostTool: "uv", imports: []string{"diffusers", "torch", "PIL"},
			techniques: []technique.Technique{
				conditioningTechnique("diffusers-txt2img", "text_to_image", technique.DiffusersText2Image),
				conditioningTechnique("diffusers-img2img", "image_to_image", technique.DiffusersImg2Img),
				conditioningTechnique("diffusers-inpaint", "inpaint", technique.DiffusersInpaint),
				conditioningTechnique("diffusers-outpaint", "outpaint", technique.DiffusersInpaint),
				conditioningTechnique("diffusers-background-replace", "background_replace", technique.DiffusersInpaint),
				nativeTechnique("diffusers-edit-instruct", "edit_instruct", technique.DiffusersEditInstruct),
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
			name: "onnxruntime", program: pythonProgram, gpuCapable: false, hostTool: "uv", imports: []string{"onnxruntime", "PIL", "numpy"},
			techniques: []technique.Technique{
				nativeTechnique("onnx-denoise", "denoise", technique.OnnxSidecar),
				nativeTechnique("onnx-deblur", "deblur", technique.OnnxSidecar),
				nativeTechnique("onnx-background-removal", "background_removal", technique.OnnxSidecar),
				nativeTechnique("onnx-object-removal", "object_removal", technique.OnnxSidecar),
				nativeTechnique("onnx-colorize", "colorize", technique.OnnxSidecar),
				nativeTechnique("onnx-depth-map", "depth_map", technique.OnnxSidecar),
				nativeTechnique("moge-scene-geometry", "scene_geometry", technique.OnnxSidecar),
				nativeTechnique("onnx-object-detection", "object_detection", technique.OnnxSidecar),
				nativeTechnique("onnx-segment", "segment", technique.OnnxSidecar),
				nativeTechnique("onnx-tagging", "tagging", technique.OnnxSidecar),
				nativeTechnique("onnx-nsfw-classify", "nsfw_classify", technique.OnnxSidecar),
				nativeTechnique("onnx-embedding", "embedding", technique.OnnxSidecar),
			},
		},
		{
			name: "python-sidecar", program: pythonProgram, gpuCapable: false, hostTool: "uv", imports: []string{"onnxruntime", "PIL", "numpy"},
			techniques: []technique.Technique{
				nativeTechnique("python-sidecar-colorize", "colorize", technique.PythonSidecar),
			},
		},
		{
			name: "python-sidecar", program: pythonProgram, gpuCapable: false, hostTool: "uv", imports: []string{"torch", "basicsr", "facexlib", "PIL", "numpy"},
			techniques: []technique.Technique{
				nativeTechnique("python-sidecar-face-restore", "face_restore", technique.PythonSidecar),
				nativeTechnique("python-sidecar-old-photo-restore", "old_photo_restore", technique.PythonSidecar),
			},
		},
		{
			name: "sam2-transformers", program: pythonProgram, gpuCapable: true, hostTool: "uv", imports: []string{"torch", "transformers", "PIL"},
			techniques: []technique.Technique{
				nativeTechnique("sam2-transformers-segment", "segment", technique.SAM2Sidecar),
			},
		},
		{
			name: "triposr", program: pythonProgram, gpuCapable: true, hostTool: "uv", imports: []string{"torch", "einops", "trimesh", "skimage", "yaml"},
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
	for _, s := range providerSpecs() {
		p := &execProvider{backendProcess: backendProcess{name: s.name, program: s.program, programAliases: s.programAliases, gpuCapable: s.gpuCapable, provision: s.provision(), imports: s.imports, lookPath: lookPath, run: run}, techniques: technique.NewSet(s.techniques...)}
		if s.isPython() {
			p.pythonInterpreter = pythonInterpreter
			p.probe.TTL = 30 * time.Second
			if s.gpuCapable {
				p.gpuProbe = torchProbe
			}
		}
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
	// BYOK cloud provider (last tier). The provider is registered unconditionally
	// so selection and readiness remain explicit, but all remote transport,
	// credentials, model policy, receipts, and retries belong to AI Gateway.
	if err := reg.Register(newAIGatewayProvider(newAIGatewayMediaClient())); err != nil {
		return fmt.Errorf("ai: register provider %q: %w", models.BackendOpenRouter, err)
	}
	return nil
}
