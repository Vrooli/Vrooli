package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"image-tools/internal/backends"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/technique"
)

// execProvider binds technique dispatch to an independent backend process.
type execProvider struct {
	backendProcess
	techniques technique.Set
	// gpuOps, when non-nil, is the set of operations whose GPU path is
	// qualified (providerSpec.gpuQualifiedOps).
	gpuOps map[string]bool
	// cpuRetry repeats a GPU run that ran out of GPU memory once on the CPU
	// (providerSpec.cpuRetryAfterOOM).
	cpuRetry bool
	// gpuHolders names the processes holding GPU memory when a run runs out of it;
	// nil reads NVML. Tests inject a fixed list.
	gpuHolders func(context.Context) []gpuHolder
}

func (p *execProvider) holders(ctx context.Context) []gpuHolder {
	if p.gpuHolders != nil {
		return p.gpuHolders(ctx)
	}
	return queryGPUHolders(ctx)
}

func opSet(ops []string) map[string]bool {
	if ops == nil {
		return nil
	}
	set := make(map[string]bool, len(ops))
	for _, op := range ops {
		set[op] = true
	}
	return set
}

func (p *execProvider) Name() string                    { return p.name }
func (p *execProvider) Standalone() bool                { return true }
func (p *execProvider) IsCloud() bool                   { return false }
func (p *execProvider) SupportsAdapters(op string) bool { return p.techniques.SupportsAdapters(op) }
func (p *execProvider) GPUCapableFor(op string) bool {
	if p.gpuOps != nil && !p.gpuOps[op] {
		return false
	}
	return p.GPUCapable()
}
func (p *execProvider) Operations() []string {
	ops := make([]string, 0, len(p.techniques))
	for op := range p.techniques {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}

func (p *execProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if p.pythonInterpreter != "" {
		ctx = withPythonRuntimeLibraries(ctx, p.pythonInterpreter)
	}
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q requires a local output path", p.name)
	}
	program, err := p.commandName()
	if err != nil {
		return backends.Result{}, err
	}
	tech, ok := p.techniques[req.Operation]
	if !ok || !tech.Ready || tech.Build == nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q does not serve operation %q", p.name, req.Operation)
	}
	modelDir := req.ModelDir
	if modelDir == "" {
		modelDir = modelDirFor(req.Model.ID)
	}
	attempt := func(req backends.Request) (string, int64, bool, error) {
		args, err := tech.Build(req, modelDir)
		if err != nil {
			return "", 0, false, fmt.Errorf("ai: backend %q build args: %w", p.name, err)
		}
		runCtx := ctx
		if req.GPU && req.GPUUUID != "" && p.SupportsGPUUUID() {
			runCtx = context.WithValue(runCtx, gpuDeviceContextKey{}, req.GPUUUID)
		}
		if p.name == "diffusers" || p.name == "onnxruntime" {
			device := "cpu"
			if req.GPU {
				device = "cuda"
			}
			args = append(args, "--device", device)
		}
		runner, peak, measured, err := p.execute(runCtx, program, args, req.Progress)
		if err != nil {
			return runner, peak, measured, classifySidecarFailure(fmt.Errorf("ai: backend %q execution failed: %w", p.name, err))
		}
		return runner, peak, measured, nil
	}
	runner, peakRSSBytes, peakRSSMeasured, err := attempt(req)
	fallback := map[string]string{}
	if err != nil && p.cpuRetry && req.GPU && !req.GPURequired && req.Model.Hardware.CPUCapable && isGPUOutOfMemory(err) {
		// The lease promised GPU memory another service has since taken. The job is
		// still wanted and the CPU can do it: say so and run it once there.
		gpuErr := err
		req.GPU, req.GPUUUID = false, ""
		runner, peakRSSBytes, peakRSSMeasured, err = attempt(req)
		if err != nil {
			return backends.Result{}, fmt.Errorf("%w (after the GPU run failed for lack of GPU memory: %v)", err, firstLine(gpuErr.Error()))
		}
		fallback = map[string]string{
			"capacity_fallback":        "cpu_after_oom",
			"capacity_fallback_reason": firstLine(gpuErr.Error()),
			MetaExecutionDevice:        "cpu",
			MetaExecutionTier:          backends.TierLocalCPU.String(),
		}
	}
	if err != nil {
		if isGPUOutOfMemory(err) && req.GPU {
			// No fallback recovered it: the memory the lease promised was taken by
			// another service, which says nothing about the model.
			if held := describeGPUHolders(p.holders(ctx)); held != "" {
				err = fmt.Errorf("%w; %s", err, held)
			}
			err = internaljobs.Kinded(internaljobs.FailureCapacity, err)
		}
		return backends.Result{}, err
	}
	meta := map[string]string{"backend": p.name}
	for key, value := range fallback {
		meta[key] = value
	}
	for key, value := range workerInferenceMetadata(ctx) {
		meta[key] = value
	}
	if runner != "" {
		meta["runner"] = runner
	}
	addProcessPeakRSSMetadata(meta, runner, peakRSSBytes, peakRSSMeasured)
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: meta}, nil
}

// gpuOutOfMemoryMarkers are the lower-cased fragments ONNX Runtime and CUDA put in
// an error when a device allocation fails.
var gpuOutOfMemoryMarkers = []string{
	"out of memory",
	"failed to allocate memory for requested buffer",
	"cudaerrormemoryallocation",
	"cudnn_status_alloc_failed",
	"cublas_status_alloc_failed",
	"bfcarena::allocaterawinternal",
}

// isGPUOutOfMemory reports whether err is a GPU memory exhaustion rather than a
// fault of the input or the model: only that is worth repeating on the CPU.
func isGPUOutOfMemory(err error) bool {
	text := strings.ToLower(err.Error())
	for _, marker := range gpuOutOfMemoryMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// firstLine trims an error to its first line, capped, for a meta value.
func firstLine(text string) string {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	if len(text) > 240 {
		text = text[:240]
	}
	return text
}

func addProcessPeakRSSMetadata(meta map[string]string, runner string, bytes int64, measured bool) {
	if measured && bytes > 0 {
		meta["process_peak_rss_bytes"] = strconv.FormatInt(bytes, 10)
		meta["process_peak_rss_source"] = "child_wait4_rusage_maxrss"
		return
	}
	meta["process_peak_rss_status"] = "unavailable"
	if runner == "warm" || runner == "operation-scoped-worker" {
		meta["process_peak_rss_reason"] = "persistent warm worker does not isolate a per-job peak RSS"
	} else {
		meta["process_peak_rss_reason"] = "backend runtime did not expose an attributable completed-child peak RSS"
	}
}

func modelDirFor(modelID string) string { return "models/" + modelID }

type llamaCppProvider struct {
	lookPath lookPathFunc
	run      outputRunner
}

func (p *llamaCppProvider) Name() string         { return "llama.cpp" }
func (p *llamaCppProvider) Operations() []string { return []string{"caption"} }
func (p *llamaCppProvider) Standalone() bool     { return true }
func (p *llamaCppProvider) IsCloud() bool        { return false }

// The governed llama-cpp host tool currently installs the official CPU bundle.
// Keep caption on an explicit CPU fallback until a GPU build is installed and
// runtime-probed; command availability alone is not GPU qualification.
func (p *llamaCppProvider) GPUCapable() bool                   { return false }
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
	var peakRSS *childPeakRSSCollector
	if run == nil {
		run = defaultRunOutput
		peakRSS = &childPeakRSSCollector{}
		ctx = context.WithValue(ctx, childPeakRSSKey{}, peakRSS)
	}
	out, err := run(ctx, program, args)
	if err != nil {
		return backends.Result{}, classifySidecarFailure(fmt.Errorf("ai: backend %q execution failed: %w", p.Name(), err))
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
	meta := map[string]string{"backend": p.Name()}
	bytes, measured := int64(0), false
	if peakRSS != nil {
		bytes, measured = peakRSS.result()
	}
	addProcessPeakRSSMetadata(meta, "", bytes, measured)
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: meta}, nil
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
