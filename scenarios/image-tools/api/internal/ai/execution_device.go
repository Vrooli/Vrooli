package ai

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

type gpuDeviceContextKey struct{}

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
	if uuid == "" {
		return cmd
	}
	// Never mutate the serving process environment: concurrent jobs and probes
	// must keep their own bindings. CUDA enumerates this one UUID as logical 0.
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "CUDA_VISIBLE_DEVICES=") && !strings.HasPrefix(value, "CUDA_DEVICE_ORDER=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "CUDA_VISIBLE_DEVICES="+uuid, "CUDA_DEVICE_ORDER=PCI_BUS_ID")
	return cmd
}
