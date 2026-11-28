//go:build linux

package collectors

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func collectPlatformMemory(_ context.Context, c *MemoryCollector, snapshot hostinventory.Snapshot) platformMemoryReading {
	if snapshot.Memory.TotalBytes == 0 {
		return platformMemoryReading{status: "failed", reason: "Linux memory snapshot was not measured", provenance: "linux /proc/meminfo"}
	}
	details := c.getMemoryDetails(snapshot)
	swap := c.getSwapUsage(snapshot)
	return platformMemoryReading{
		usage:      c.getMemoryUsage(snapshot),
		status:     "measured",
		provenance: "linux /proc/meminfo",
		details:    details,
		swap:       swap,
	}
}
