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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"
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

// pythonProgram is the sentinel program string that marks a backend as Python-
// served. Such backends run ONLY through the scenario's private uv venv
// interpreter (execProvider.pythonInterpreter); they never resolve a bare
// "python3" off the host PATH.
const pythonProgram = "python3"

// execProvider is a backends.Provider backed by an external CLI/sidecar program.
// One configured instance exists per standalone backend (stable-diffusion.cpp,
// diffusers, iopaint, realesrgan-ncnn-vulkan, rembg, onnxruntime). It owns the
// backend *process* (availability probe, exec, host-tool resolution) and
// dispatches arg-building to the technique.Set it was registered with — it never
// re-declares the arg shapes (those live in internal/technique). Availability is
// "the program resolves on PATH"; model-weight presence is a separate gate the
// engine applies, keeping providers model-agnostic.
type execProvider struct {
	name    string
	program string // binary or interpreter resolved on PATH (e.g. "sd", "python3")
	// pythonInterpreter, when set, is the ABSOLUTE path to this scenario's private
	// uv-built venv interpreter (<scenario-data>/pyenv/bin/python). Python backends
	// (program==pythonProgram) invoke it directly, so their heavy deps (torch/
	// diffusers/transformers/onnxruntime) come from the lock-pinned venv and cannot
	// be perturbed by other Python on the box. There is deliberately NO bare-
	// "python3" PATH fallback: running AI ops against the shared host interpreter is
	// exactly the cross-contamination this isolation seam exists to prevent. Empty ⇒
	// the venv is not provisioned yet and the backend reports unavailable (surfaced
	// before use via doctor/health/ready_state), never silently runs unisolated.
	pythonInterpreter string
	// programAliases are preferred program names tried (in order) on PATH before
	// program. They let an optional GPU build install under a distinct command
	// (e.g. "sd-gpu") and be picked up automatically without colliding with the
	// base CPU launcher's command name. Empty for backends with no GPU variant.
	programAliases []string
	// techniques maps an operation to the technique that serves it on this backend.
	// Execute dispatches by req.Operation; an op with no technique is unsupported.
	techniques technique.Set
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

func (p *execProvider) Name() string     { return p.name }
func (p *execProvider) Standalone() bool { return true } // none of these are ComfyUI
func (p *execProvider) IsCloud() bool    { return false }

// Operations returns the ops this backend serves, sorted for determinism.
func (p *execProvider) Operations() []string {
	ops := make([]string, 0, len(p.techniques))
	for op := range p.techniques {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}

// programName returns the program this provider should invoke: the first of its
// programAliases that resolves on PATH (a GPU build like "sd-gpu"), else the
// base program ("sd"). Resolution is cheap (a PATH lookup) and done per call so
// installing a GPU build at runtime is picked up without a restart.
func (p *execProvider) programName() string {
	// Python backends invoke the scenario's private venv interpreter by absolute
	// path — and ONLY that. There is no bare-"python3" PATH fallback: that would
	// reintroduce the host-Python contamination this isolation seam prevents. An
	// empty interpreter means the venv is not provisioned yet, and Availability/
	// Execute report the backend unavailable (with remediation) rather than
	// silently running unisolated.
	if p.program == pythonProgram {
		return p.pythonInterpreter
	}
	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	for _, alias := range p.programAliases {
		if _, err := lookPath(alias); err == nil {
			return alias
		}
	}
	return p.program
}

// GPUCapable reports whether this backend can run on the GPU. The onnxruntime
// sidecar is bound to CPUExecutionProvider, so it returns false and the selector
// labels its runs local-cpu even on a GPU host (honest tier reporting).
//
// gpuCapable is the static capability of the backend TYPE. When a gpuProbe is
// set, the INSTALLED binary is also consulted (cached): a prebuilt
// stable-diffusion.cpp release with no CUDA/Vulkan backend compiled in runs on
// the CPU regardless of a GPU on the host, so claiming local-gpu would be a lie
// (the user sees "Running on your GPU" while the binary reports VRAM 0.00MB).
// SupportsAdapters reports whether this backend can apply the conditioning
// adapter stack for op (LoRA/ControlNet/IP-Adapter). It delegates to the op's
// technique, so only backends whose builder consumes req.Adapters (the diffusers
// sidecar) advertise support — backend selection routes conditioned requests
// here. Satisfies the backends.adapterCapableProvider optional capability.
func (p *execProvider) SupportsAdapters(op string) bool {
	return p.techniques.SupportsAdapters(op)
}

func (p *execProvider) GPUCapable() bool {
	if !p.gpuCapable {
		return false
	}
	if p.gpuProbe == nil {
		return true
	}
	p.gpuProbeOnce.Do(func() {
		program := p.programName()
		if p.lookPath != nil {
			if resolved, err := p.lookPath(program); err == nil {
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
	program := p.programName()
	// A Python backend with no resolved interpreter means its isolated venv is not
	// provisioned yet (uv missing, or the background build hasn't finished). Report
	// unavailable with the provisioning remediation instead of probing a bare PATH
	// python3 — the isolation contract has no host-interpreter fallback.
	if p.program == pythonProgram && program == "" {
		return backends.Availability{
			Available: false,
			Detail:    "image-tools Python venv not provisioned yet (the isolated uv interpreter is unavailable); AI ops never run against the host python3",
			Provision: p.provision,
		}
	}
	resolved, err := lookPath(program)
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
			Detail:    fmt.Sprintf("%s resolved at %s", program, resolved),
			Provision: p.provision,
		}
	}
	return backends.Availability{
		Available: false,
		Detail:    fmt.Sprintf("%s not found on PATH", program),
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

// Execute resolves the model directory, dispatches to the operation's technique
// arg-builder, and runs the program. The output is written to
// req.Output.LocalPath by contract; Execute returns that path as the result ref.
func (p *execProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q requires a local output path", p.name)
	}
	modelDir := req.ModelDir
	if modelDir == "" {
		modelDir = modelDirFor(req.Model.ID)
	}
	// Resolve the program and reject an unprovisioned Python backend before building
	// args or running anything — a Python backend with no venv interpreter must not
	// fall through to the host python3 (the isolation contract).
	program := p.programName()
	if p.program == pythonProgram && program == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q unavailable: image-tools Python venv not provisioned — %s", p.name, p.provision)
	}
	tech, ok := p.techniques[req.Operation]
	if !ok {
		return backends.Result{}, fmt.Errorf("ai: backend %q does not serve operation %q", p.name, req.Operation)
	}
	args, err := tech.Build(req, modelDir)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q build args: %w", p.name, err)
	}
	if p.warm != nil {
		if err := p.warm.Run(ctx, program, args); err == nil {
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
		if err := stream(ctx, program, args, onLine); err != nil {
			return backends.Result{}, fmt.Errorf("ai: backend %q execution failed: %w", p.name, err)
		}
		return backends.Result{OutputRef: req.Output.LocalPath, Meta: map[string]string{"backend": p.name, "runner": "stream"}}, nil
	}
	run := p.run
	if run == nil {
		run = defaultRun
	}
	if err := run(ctx, program, args); err != nil {
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
	args, err := technique.LlamaCppCaption(req, modelDir)
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
				nativeTechnique("onnx-colorize", "colorize", technique.OnnxSidecar),
				nativeTechnique("onnx-depth-map", "depth_map", technique.OnnxSidecar),
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
	}
}

// RegisterProviders registers the standalone AI backends on reg. Pass nil
// lookPath/run to use the real os/exec implementations.
//
// pythonInterpreter is the absolute path to the scenario's private uv venv
// interpreter; every Python backend invokes it directly (the isolation seam —
// see internal/pyenv and main.go boot wiring). Pass "" when the venv is not
// provisioned (uv absent or the background build is still running): Python
// backends then report unavailable (no bare-"python3" PATH fallback), surfaced
// before use via doctor/health/ready_state.
func RegisterProviders(reg *backends.Registry, lookPath lookPathFunc, run commandRunner, pythonInterpreter string) error {
	for _, s := range providerSpecs() {
		p := &execProvider{name: s.name, program: s.program, programAliases: s.programAliases, techniques: technique.NewSet(s.techniques...), gpuCapable: s.gpuCapable, provision: s.provision(), imports: s.imports, lookPath: lookPath, run: run}
		if s.isPython() {
			p.pythonInterpreter = pythonInterpreter
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
