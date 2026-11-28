package ai

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"image-tools/internal/modelenv"
	"image-tools/internal/models"
	"image-tools/internal/sidecar"
)

type (
	gpuDeviceContextKey  struct{}
	cpuThreadsContextKey struct{}
	pythonLibrariesKey   struct{}
)

func withPythonRuntimeLibraries(ctx context.Context, python string) context.Context {
	return context.WithValue(ctx, pythonLibrariesKey{}, sidecar.RuntimeLibraries(python))
}

func pythonProbeCommand(ctx context.Context, python string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Env = modelenv.Apply(nil, modelenv.ModeInference)
	return withLibraryPath(cmd, sidecar.RuntimeLibraries(python))
}

func probeTorchCUDA(ctx context.Context, python string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if python == "" {
		return false
	}
	cmd := exec.CommandContext(ctx, python, "-c", "import torch; print('cuda-ready' if torch.cuda.is_available() else 'cpu-only')")
	cmd.Env = modelenv.Apply(nil, modelenv.ModeInference)
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "cuda-ready"
}

func probeONNXCUDA(ctx context.Context, python string) bool {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if python == "" {
		return false
	}
	code := "import torch, onnxruntime as ort; print('cuda-ready' if torch.cuda.is_available() and 'CUDAExecutionProvider' in ort.get_available_providers() else 'cpu-only')"
	out, err := pythonProbeCommand(ctx, python, "-c", code).Output()
	return err == nil && strings.TrimSpace(string(out)) == "cuda-ready"
}

// SupportsGPUUUID names backends that can bind execution to the exact physical
// device the capacity ledger leased. CUDA backends honor per-process visible
// devices. The ncnn-vulkan backend does not read CUDA variables; it is qualified
// separately because technique.NcnnDeviceArgs maps the leased UUID to the
// Vulkan device index and passes it as `-g`.
func (p *execProvider) SupportsGPUUUID() bool {
	if models.IsTransformersBackend(p.name) {
		return true
	}
	switch p.name {
	case "diffusers", "iopaint", "onnxruntime", "triposr", "realesrgan-ncnn-vulkan":
		return true
	default:
		return false
	}
}

// admittedCPUThreads returns the whole-thread CPU budget Admit reserved for the
// current job, or zero when the context carries no CPU admission.
func admittedCPUThreads(ctx context.Context) int {
	threads, _ := ctx.Value(cpuThreadsContextKey{}).(int)
	return max(threads, 0)
}

// inferenceCommand builds every job-time process: CUDA/CPU admission bindings
// plus the model containment environment (cache variables pinned under the
// managed root, hubs offline, non-loopback network refused). No inference
// process is ever started without the containment overlay.
func inferenceCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	uuid, _ := ctx.Value(gpuDeviceContextKey{}).(string)
	threads := admittedCPUThreads(ctx)
	pythonLibraries, _ := ctx.Value(pythonLibrariesKey{}).([]string)
	// Never mutate the serving process environment: concurrent jobs and probes
	// must keep their own bindings. CUDA enumerates this one UUID as logical 0.
	overrides := map[string]string{"CUDA_VISIBLE_DEVICES": ""}
	if uuid != "" {
		overrides["CUDA_VISIBLE_DEVICES"] = uuid
		overrides["CUDA_DEVICE_ORDER"] = "PCI_BUS_ID"
	}
	if threads > 0 {
		for _, key := range []string{"OMP_NUM_THREADS", "MKL_NUM_THREADS", "OPENBLAS_NUM_THREADS", "IMAGE_TOOLS_CPU_THREADS"} {
			overrides[key] = strconv.Itoa(threads)
		}
		// Software Vulkan (llvmpipe) sizes its own pool from this variable; a CPU
		// job must stay inside the threads admission reserved.
		overrides["LP_NUM_THREADS"] = strconv.Itoa(threads)
	}
	if len(pythonLibraries) > 0 {
		current := strings.TrimSpace(os.Getenv("LD_LIBRARY_PATH"))
		if current != "" {
			pythonLibraries = append(pythonLibraries, current)
		}
		overrides["LD_LIBRARY_PATH"] = strings.Join(pythonLibraries, string(os.PathListSeparator))
	}
	if uuid == "" && threads <= 0 && len(pythonLibraries) == 0 {
		// No admission binding: keep the historical environment (CUDA hidden is
		// only applied when a binding exists) and add containment alone.
		cmd.Env = modelenv.Apply(nil, modelenv.ModeInference)
		return cmd
	}
	var env []string
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if _, overridden := overrides[key]; !overridden {
			env = append(env, value)
		}
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	cmd.Env = modelenv.Apply(env, modelenv.ModeInference)
	return cmd
}

func withLibraryPath(cmd *exec.Cmd, libraries []string) *exec.Cmd {
	if len(libraries) == 0 {
		return cmd
	}
	current := strings.TrimSpace(os.Getenv("LD_LIBRARY_PATH"))
	if current != "" {
		libraries = append(libraries, current)
	}
	base := cmd.Env
	if base == nil {
		base = os.Environ()
	}
	key := "LD_LIBRARY_PATH="
	env := make([]string, 0, len(base)+1)
	for _, value := range base {
		if !strings.HasPrefix(value, key) {
			env = append(env, value)
		}
	}
	cmd.Env = append(env, key+strings.Join(libraries, string(os.PathListSeparator)))
	return cmd
}
