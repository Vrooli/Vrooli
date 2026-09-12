//go:build !linux && !darwin && !windows

package hostinventory

import "errors"

func hostMemoryFacts() (HostMemory, error) {
	return HostMemory{}, errors.New("host memory is unsupported on this platform")
}
