package collectors

import (
	"fmt"
	"runtime"
)

// RootMountPath is the filesystem the scenario treats as "the host disk" when no
// specific path is named.
func RootMountPath() string {
	if runtime.GOOS == "windows" {
		return `C:\`
	}
	return "/"
}

// DiskUsage is a single filesystem's capacity, measured the way `df` measures it.
type DiskUsage struct {
	MountPath      string
	TotalBytes     int64
	UsedBytes      int64
	AvailableBytes int64
	UsedPercent    float64
	// FillRateBytesPerHour is derived by the threshold scheduler from two
	// samples. A zero value means no usable predecessor exists yet.
	FillRateBytesPerHour int64
}

// diskUsedPercent reports capacity the way `df` reports it: used over
// (used + available), not used over total.
//
// The difference is the superuser reserve. `free` includes blocks no
// unprivileged process may ever allocate, so used/total quietly counts the
// reserve as usable headroom. On the 2026-07-31 incident host that gap was
// 93 GB — six percentage points — which was enough to keep every safeguard
// below its critical threshold while the filesystem was unwritable for the
// runtime supervisor.
//
// Both callers of this function previously computed used/total independently.
func diskUsedPercent(used, available int64) float64 {
	capacity := used + available
	if capacity <= 0 {
		return 0
	}
	return float64(used) / float64(capacity) * 100
}

// diskUsageFrom converts raw statfs byte counts into a DiskUsage.
func diskUsageFrom(total, free, available int64) DiskUsage {
	used := total - free
	if used < 0 {
		used = 0
	}
	return DiskUsage{
		TotalBytes:     total,
		UsedBytes:      used,
		AvailableBytes: available,
		UsedPercent:    diskUsedPercent(used, available),
	}
}

// ReadDiskUsage measures the filesystem containing path. It is the single
// production entry point for disk-pressure decisions in this scenario.
func ReadDiskUsage(path string) (DiskUsage, error) {
	total, free, available, err := statfsBytes(path)
	if err != nil {
		return DiskUsage{}, err
	}
	if total <= 0 || available < 0 || free < 0 {
		return DiskUsage{}, fmt.Errorf("filesystem capacity is unavailable for %q", path)
	}
	usage := diskUsageFrom(total, free, available)
	usage.MountPath = path
	return usage, nil
}
