//go:build linux

package collectors

import "os"

func readPlatformPaging(vmstat map[string]uint64) pagingReading {
	return linuxPagingReading(vmstat, uint64(os.Getpagesize()))
}
