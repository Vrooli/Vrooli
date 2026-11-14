//go:build !linux

package hostinventory

import "fmt"

func ReadKernelMemory() (KernelMemoryReport, error) {
	return KernelMemoryReport{}, fmt.Errorf("kernel slab diagnostics are supported only on Linux")
}
