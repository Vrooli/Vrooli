//go:build linux

package collectors

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func collectPlatformCPU(_ context.Context, c *CPUCollector, snapshot hostinventory.Snapshot) platformCPUReading {
	first := c.lastCPUStats == nil
	usage := c.getCPUUsage()
	if first {
		return platformCPUReading{
			status:      "failed",
			reason:      "not_yet_sampled: CPU delta requires a previous sample",
			provenance:  "linux /proc/stat",
			loadAverage: c.getLoadAverage(snapshot),
		}
	}
	contextSwitches := c.getContextSwitches()
	return platformCPUReading{
		usage:           usage,
		status:          "measured",
		provenance:      "linux /proc/stat",
		loadAverage:     c.getLoadAverage(snapshot),
		contextSwitches: &contextSwitches,
	}
}
