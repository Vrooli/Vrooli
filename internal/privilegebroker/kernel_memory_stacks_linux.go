//go:build linux

package privilegebroker

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

var kernelMemorySamplerCandidates = []string{
	"/usr/sbin/memleak-bpfcc",
	"/usr/bin/memleak-bpfcc",
	"/sbin/memleak-bpfcc",
}

func trustedKernelMemorySampler() (string, error) {
	for _, candidate := range kernelMemorySamplerCandidates {
		info, err := os.Lstat(candidate)
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 || info.Mode().Perm()&0o022 != 0 {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("root-owned memleak-bpfcc is not installed in a trusted system path")
}

func sampleKernelMemoryStacks(ctx context.Context, executor Executor) (hostinventory.KernelMemoryStackReport, error) {
	tool, err := trustedKernelMemorySampler()
	if err != nil {
		return hostinventory.KernelMemoryStackReport{}, err
	}
	return runKernelMemoryStackSampler(ctx, executor, tool)
}
