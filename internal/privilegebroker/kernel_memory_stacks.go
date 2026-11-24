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

// Stack-sample rankings. The sampler ranks by bytes by default, which hides a
// leak made of many tiny objects; "count" surfaces it by outstanding
// allocation count instead. The set is closed so the argv stays fixed.
const (
	KernelMemorySortSize  = "size"
	KernelMemorySortCount = "count"
)

// NormalizeKernelMemorySort resolves an empty request to the default ranking
// and rejects anything outside the closed set.
func NormalizeKernelMemorySort(sort string) (string, error) {
	switch strings.TrimSpace(sort) {
	case "":
		return KernelMemorySortSize, nil
	case KernelMemorySortSize:
		return KernelMemorySortSize, nil
	case KernelMemorySortCount:
		return KernelMemorySortCount, nil
	default:
		return "", fmt.Errorf("unsupported kernel memory sort %q", sort)
	}
}

func runKernelMemoryStackSampler(ctx context.Context, executor Executor, tool, sortBy string) (hostinventory.KernelMemoryStackReport, error) {
	sortBy, err := NormalizeKernelMemorySort(sortBy)
	if err != nil {
		return hostinventory.KernelMemoryStackReport{}, err
	}
	if executor == nil {
		return hostinventory.KernelMemoryStackReport{}, fmt.Errorf("privilege broker has no command executor")
	}
	// This fixed command attaches briefly to kernel allocation/free tracepoints.
	// It samples one in every 100 allocations, returns at most ten symbol-only
	// stacks, and never dumps individual allocation addresses.
	output, err := executor.Run(ctx, tool,
		"--combined-only", "--sample-rate", fmt.Sprint(kernelMemoryStackSampleEvery),
		"--sort", sortBy, "--top", fmt.Sprint(kernelMemoryStackLimit),
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
		SortBy:          sortBy,
		Output:          strings.TrimSpace(string(output)),
		OutputTruncated: truncated,
	}, nil
}
