//go:build darwin && !cgo

package hostinventory

import (
	"fmt"
	platform "github.com/vrooli/platform-go"
)

func hostMemoryFacts() (HostMemory, error) {
	return HostMemory{}, fmt.Errorf("native Mach memory counters require the macOS CGO build: %w", platform.ErrUnsupported)
}
