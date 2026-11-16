package ai

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type (
	gpuDeviceContextKey  struct{}
	cpuThreadsContextKey struct{}
	pythonLibrariesKey   struct{}
)

func pythonRuntimeLibraries(python string) []string {
	if python == "" {
		return nil
	}
	venv := filepath.Dir(filepath.Dir(python))
	matches, _ := filepath.Glob(filepath.Join(venv, "lib", "python*", "site-packages", "nvidia", "*", "lib"))
	var dirs []string
	for _, path := range matches {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			dirs = append(dirs, path)
		}
	}
	return dirs
}

func withPythonRuntimeLibraries(ctx context.Context, python string) context.Context {
	return context.WithValue(ctx, pythonLibrariesKey{}, pythonRuntimeLibraries(python))
}

func pythonProbeCommand(ctx context.Context, python string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, python, args...)
	return withLibraryPath(cmd, pythonRuntimeLibraries(python))
}

func probeTorchCUDA(ctx context.Context, python string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if python == "" {
		return false
	}
	out, err := exec.CommandContext(ctx, python, "-c", "import torch; print('cuda-ready' if torch.cuda.is_available() else 'cpu-only')").Output()
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

// SupportsGPUUUID names backends whose CUDA runtime honors per-process visible
// devices. Vulkan/NCNN selection must be qualified separately, rather than
// pretending a CUDA environment variable binds those runtimes too.
func (p *execProvider) SupportsGPUUUID() bool {
	switch p.name {
	case "diffusers", "iopaint", "onnxruntime", "sam2-transformers", "triposr":
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

func inferenceCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	uuid, _ := ctx.Value(gpuDeviceContextKey{}).(string)
	threads := admittedCPUThreads(ctx)
	pythonLibraries, _ := ctx.Value(pythonLibrariesKey{}).([]string)
	if uuid == "" && threads <= 0 && len(pythonLibraries) == 0 {
		return cmd
	}
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
	}
	if len(pythonLibraries) > 0 {
		current := strings.TrimSpace(os.Getenv("LD_LIBRARY_PATH"))
		if current != "" {
			pythonLibraries = append(pythonLibraries, current)
		}
		overrides["LD_LIBRARY_PATH"] = strings.Join(pythonLibraries, string(os.PathListSeparator))
	}
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if _, overridden := overrides[key]; !overridden {
			cmd.Env = append(cmd.Env, value)
		}
	}
	for key, value := range overrides {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
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
	key := "LD_LIBRARY_PATH="
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, key) {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, key+strings.Join(libraries, string(os.PathListSeparator)))
	return cmd
}
