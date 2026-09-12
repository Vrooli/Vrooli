//go:build darwin

package collectors

func readPlatformPaging(_ map[string]uint64) pagingReading {
	return pagingReading{supported: false, reason: "host_statistics64 paging backend is not available in this build", provenance: "host_statistics64"}
}
