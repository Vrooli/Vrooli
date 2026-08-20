//go:build darwin

package collectors

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"golang.org/x/sys/unix"
)

func collectPlatformCPU(_ context.Context, c *CPUCollector, snapshot hostinventory.Snapshot) platformCPUReading {
	raw, err := unix.SysctlRaw("kern.cp_time")
	if err != nil {
		return platformCPUReading{status: "failed", reason: fmt.Sprintf("kern.cp_time: %v", err), provenance: "darwin sysctl kern.cp_time"}
	}
	counters, err := darwinCPUCounters(raw)
	if err != nil {
		return platformCPUReading{status: "failed", reason: err.Error(), provenance: "darwin sysctl kern.cp_time"}
	}
	usage, measured := deltaCPUUsage(&c.platformState, counters, time.Now(), 4)
	if !measured {
		return platformCPUReading{status: "failed", reason: "not_yet_sampled: CPU delta requires a previous sample", provenance: "darwin sysctl kern.cp_time"}
	}
	return platformCPUReading{usage: usage, status: "measured", provenance: "darwin sysctl kern.cp_time"}
}

func darwinCPUCounters(raw []byte) ([]uint64, error) {
	if len(raw) >= 5*8 {
		counters := make([]uint64, len(raw)/8)
		for i := range counters {
			counters[i] = binary.LittleEndian.Uint64(raw[i*8:])
		}
		return counters, nil
	}
	if len(raw) >= 5*4 {
		counters := make([]uint64, len(raw)/4)
		for i := range counters {
			counters[i] = uint64(binary.LittleEndian.Uint32(raw[i*4:]))
		}
		return counters, nil
	}
	return nil, fmt.Errorf("kern.cp_time returned %d bytes; expected at least 20", len(raw))
}
