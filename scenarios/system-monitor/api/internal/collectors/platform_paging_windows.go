//go:build windows

package collectors

func readPlatformPaging(_ map[string]uint64) pagingReading {
	return pagingReading{supported: false, reason: "PDH paging counters are not available in this build", provenance: "PDH"}
}
