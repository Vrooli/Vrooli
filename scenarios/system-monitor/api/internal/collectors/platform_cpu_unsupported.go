//go:build !linux && !darwin && !windows

package collectors

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func collectPlatformCPU(context.Context, *CPUCollector, hostinventory.Snapshot) platformCPUReading {
	return platformCPUUnsupported("CPU backend is unavailable on this operating system")
}
