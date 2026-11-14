package ai

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type (
	gpuDeviceContextKey  struct{}
	cpuThreadsContextKey struct{}
)

func probeTorchCUDA(ctx context.Context, python string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if python == "" {
		return false
	}
	out, err := exec.CommandContext(ctx, python, "-c", "import torch; print('cuda-ready' if torch.cuda.is_available() else 'cpu-only')").Output()
	return err == nil && strings.TrimSpace(string(out)) == "cuda-ready"
}

// SupportsGPUUUID names backends whose CUDA runtime honors per-process visible
// devices. Vulkan/NCNN selection must be qualified separately, rather than
// pretending a CUDA environment variable binds those runtimes too.
func (p *execProvider) SupportsGPUUUID() bool {
	switch p.name {
	case "diffusers", "iopaint", "sam2-transformers", "triposr":
		return true
	default:
		return false
	}
}

func inferenceCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	uuid, _ := ctx.Value(gpuDeviceContextKey{}).(string)
	threads, _ := ctx.Value(cpuThreadsContextKey{}).(int)
	if uuid == "" && threads <= 0 {
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
