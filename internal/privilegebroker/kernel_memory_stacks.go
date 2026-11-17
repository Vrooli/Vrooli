package privilegebroker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

const (
	kernelMemoryStackSampleTimeout = 15 * time.Second
	kernelMemoryStackWindowSeconds = 10
	kernelMemoryStackSampleEvery   = 100
	kernelMemoryStackLimit         = 10
	kernelMemoryStackOutputLimit   = 32 * 1024
)

func runKernelMemoryStackSampler(ctx context.Context, executor Executor, tool string) (hostinventory.KernelMemoryStackReport, error) {
	if executor == nil {
		return hostinventory.KernelMemoryStackReport{}, fmt.Errorf("privilege broker has no command executor")
	}
	// This fixed command attaches briefly to kernel allocation/free tracepoints.
	// It samples one in every 100 allocations, returns at most ten symbol-only
	// stacks, and never dumps individual allocation addresses.
	output, err := executor.Run(ctx, tool,
		"--combined-only", "--sample-rate", fmt.Sprint(kernelMemoryStackSampleEvery),
		"--sort", "size", "--top", fmt.Sprint(kernelMemoryStackLimit),
		fmt.Sprint(kernelMemoryStackWindowSeconds), "1")
	if err != nil {
		return hostinventory.KernelMemoryStackReport{}, fmt.Errorf("memleak-bpfcc sample failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	truncated := len(output) > kernelMemoryStackOutputLimit
	if truncated {
		output = output[:kernelMemoryStackOutputLimit]
	}
	return hostinventory.KernelMemoryStackReport{
		CapturedAt:      time.Now().UTC(),
		WindowSeconds:   kernelMemoryStackWindowSeconds,
		SampleEvery:     kernelMemoryStackSampleEvery,
		StackLimit:      kernelMemoryStackLimit,
		Output:          strings.TrimSpace(string(output)),
		OutputTruncated: truncated,
	}, nil
}
