//go:build !linux && !darwin && !windows

package collectors

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func collectPlatformMemory(context.Context, *MemoryCollector, hostinventory.Snapshot) platformMemoryReading {
	return platformMemoryUnsupported("memory backend is unavailable on this operating system")
}
