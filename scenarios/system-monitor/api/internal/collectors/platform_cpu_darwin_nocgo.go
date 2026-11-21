//go:build darwin && !cgo

package collectors

import "fmt"

func readDarwinCPUCounters() ([][]uint64, string, error) {
	return nil, "unsupported", fmt.Errorf("Mach host_processor_info requires a native Darwin build with CGO enabled")
}
