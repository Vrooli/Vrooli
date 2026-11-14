//go:build darwin

package collectors

import (
	"context"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"time"
)

func collectPlatformCPU(_ context.Context, c *CPUCollector, snapshot hostinventory.Snapshot) platformCPUReading {
	cores, status, err := readDarwinCPUCounters()
	return darwinCPUReading(&c.platformState, snapshot, cores, status, err, time.Now())
}
