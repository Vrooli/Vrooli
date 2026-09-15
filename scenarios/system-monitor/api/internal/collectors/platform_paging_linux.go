//go:build linux

package collectors

func readPlatformPaging(vmstat map[string]uint64) pagingReading {
	return pagingReading{counters: vmstat, supported: true, provenance: "/proc/vmstat"}
}
