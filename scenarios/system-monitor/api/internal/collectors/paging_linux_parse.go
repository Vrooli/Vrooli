package collectors

// linuxPagingReading maps /proc/vmstat onto the paging seam. pswpin/pswpout
// count pages moved between RAM and swap, so the swap aliases convert to
// bytes with the kernel page size.
func linuxPagingReading(vmstat map[string]uint64, pageSize uint64) pagingReading {
	return pagingReading{
		counters:     vmstat,
		rateCounters: []string{"pswpin", "pswpout", "pgmajfault", "pgfault", "workingset_refault_file", "workingset_refault_anon", "allocstall_movable", "allocstall_normal", "oom_kill"},
		aliases:      map[string]string{"pswpin": "swap_in", "pswpout": "swap_out", "pgmajfault": "major_faults", "pgfault": "page_faults"},
		unsupportedAliases: map[string]string{
			"page_ins":       "Linux has no host-wide page-in counter distinct from major faults; see major_faults",
			"page_reads":     "page read operations are a Windows counter",
			"compressions":   "Linux zswap/zram compression counters are not collected",
			"decompressions": "Linux zswap/zram decompression counters are not collected",
		},
		gauges: map[string]PressureSignal{
			SignalPagingFileUsagePercent: unsupportedSignal("paging-file usage is a Windows counter; Linux swap usage is reported by the memory collector", "/proc/vmstat", "percent"),
			SignalCommitChargePercent:    unsupportedSignal("commit charge is a Windows counter", "/proc/vmstat", "percent"),
			SignalNativePressureLevel:    unsupportedSignal("Linux exposes PSI rather than a native pressure level", "/proc/vmstat", "native level"),
		},
		pageSize:   pageSize,
		supported:  true,
		provenance: "/proc/vmstat",
	}
}
