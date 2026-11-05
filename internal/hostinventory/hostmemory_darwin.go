//go:build darwin

package hostinventory

import "golang.org/x/sys/unix"

func hostMemoryFacts() (HostMemory, error) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return HostMemory{}, err
	}
	free, err := unix.SysctlUint64("vm.page_free_count")
	if err != nil {
		return HostMemory{}, err
	}
	inactive, err := unix.SysctlUint64("vm.page_inactive_count")
	if err != nil {
		return HostMemory{}, err
	}
	speculative, err := unix.SysctlUint64("vm.page_speculative_count")
	if err != nil {
		return HostMemory{}, err
	}
	available := (free + inactive + speculative) * uint64(unix.Getpagesize())
	return HostMemory{TotalBytes: total, AvailableBytes: available, Trustworthy: total > 0 && available > 0}, nil
}
