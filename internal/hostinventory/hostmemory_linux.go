//go:build linux

package hostinventory

import "os"

func hostMemoryFacts() (HostMemory, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return HostMemory{}, err
	}
	memory, _, err := ParseLinuxMeminfo(string(data))
	if err != nil {
		return HostMemory{}, err
	}
	return HostMemory{TotalBytes: memory.TotalBytes, AvailableBytes: memory.AvailableBytes, Trustworthy: memory.TotalBytes > 0 && memory.AvailableBytes > 0}, nil
}
