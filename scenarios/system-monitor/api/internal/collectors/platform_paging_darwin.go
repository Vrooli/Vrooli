//go:build darwin

package collectors

import (
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/darwinsys"
	"golang.org/x/sys/unix"
)

func readPlatformPaging(_ map[string]uint64) pagingReading {
	stats, statsErr := darwinsys.HostStatistics64()
	level, levelErr := unix.SysctlUint32("kern.memorystatus_vm_pressure_level")
	return darwinPagingReading(stats, statsErr, level, levelErr, uint64(unix.Getpagesize()))
}
