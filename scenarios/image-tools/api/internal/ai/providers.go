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
	"image-tools/internal/technique"
)

// execProvider binds technique dispatch to an independent backend process.
type execProvider struct {
	backendProcess
	techniques technique.Set
}

func (p *execProvider) Name() string                    { return p.name }
func (p *execProvider) Standalone() bool                { return true }
func (p *execProvider) IsCloud() bool                   { return false }
func (p *execProvider) SupportsAdapters(op string) bool { return p.techniques.SupportsAdapters(op) }
func (p *execProvider) Operations() []string {
	ops := make([]string, 0, len(p.techniques))
	for op := range p.techniques {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}

func (p *execProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
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
	args, err := tech.Build(req, modelDir)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q build args: %w", p.name, err)
	}
	if req.GPU && req.GPUUUID != "" && p.SupportsGPUUUID() {
		ctx = context.WithValue(ctx, gpuDeviceContextKey{}, req.GPUUUID)
	}
	if p.name == "diffusers" {
		device := "cpu"
		if req.GPU {
			device = "cuda"
		}
		args = append(args, "--device", device)
	}
	runner, peakRSSBytes, peakRSSMeasured, err := p.execute(ctx, program, args, req.Progress)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: backend %q execution failed: %w", p.name, err)
	}
	meta := map[string]string{"backend": p.name}
	if runner != "" {
		meta["runner"] = runner
	}
	addProcessPeakRSSMetadata(meta, runner, peakRSSBytes, peakRSSMeasured)
	return backends.Result{OutputRef: req.Output.LocalPath, Meta: meta}, nil
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
	var peakRSS *childPeakRSSCollector
	if run == nil {
		run = defaultRunOutput
		peakRSS = &childPeakRSSCollector{}
		ctx = context.WithValue(ctx, childPeakRSSKey{}, peakRSS)
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
