//go:build !linux && !darwin && !windows

package collectors

func readPlatformPaging(_ map[string]uint64) pagingReading {
	return pagingReading{supported: false, reason: "paging counters are unsupported on this platform", provenance: "platform capability"}
}
