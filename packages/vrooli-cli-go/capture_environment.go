package vroolicli

import (
	"context"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// CaptureEnvironment maps a typed host inventory response into the generic
// commonv1.CaptureEnvironment embedded in execution metrics. It is the single
// shared HostInventoryResponse → CaptureEnvironment adapter; it lives here (on
// the universal CLI path, reachable from any scenario regardless of module name)
// rather than in api-core, so api-core's metrics collector takes host facts by
// value and gains no host-inventory dependency.
//
// A nil response yields nil so callers can pass the result straight to
// metrics.WithEnvironment, which backfills os/arch/num_cpu from the stdlib.
func CaptureEnvironment(resp *cliv1.HostInventoryResponse) *commonv1.CaptureEnvironment {
	if resp == nil {
		return nil
	}
	env := &commonv1.CaptureEnvironment{
		Os:            resp.GetOs(),
		Arch:          resp.GetArch(),
		NumCpu:        resp.GetCpu().GetCores(),
		TotalMemBytes: int64(resp.GetMemory().GetTotalBytes()),
	}
	for _, gpu := range resp.GetGpus() {
		if gpu == nil {
			continue
		}
		env.Gpus = append(env.Gpus, &commonv1.GpuInfo{
			Index:         gpu.GetIndex(),
			Name:          gpu.GetName(),
			Vendor:        gpuVendor(gpu.GetName(), gpu.GetSource()),
			MemTotalBytes: int64(gpu.GetVramBytes()),
		})
	}
	return env
}

// HostCaptureEnvironment fetches `vrooli host inventory --json` and maps it to a
// commonv1.CaptureEnvironment in one call. It is the convenience entry point for
// callers that want host facts without holding a HostInventoryResponse.
func (c *Client) HostCaptureEnvironment(ctx context.Context) (*commonv1.CaptureEnvironment, error) {
	resp, err := c.HostInventory(ctx)
	if err != nil {
		return nil, err
	}
	return CaptureEnvironment(resp), nil
}

// gpuVendor infers a coarse vendor tag from a GPU name and/or its probe source.
// Unknown hardware degrades to "unknown" rather than a fabricated vendor.
func gpuVendor(name, source string) string {
	hay := strings.ToLower(name + " " + source)
	switch {
	case strings.Contains(hay, "nvidia"):
		return "nvidia"
	case strings.Contains(hay, "amd"), strings.Contains(hay, "radeon"), strings.Contains(hay, "rocm"):
		return "amd"
	case strings.Contains(hay, "apple"):
		return "apple"
	case strings.Contains(hay, "intel"):
		return "intel"
	default:
		return "unknown"
	}
}
