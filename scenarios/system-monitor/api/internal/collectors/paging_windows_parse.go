package collectors

const (
	windowsPDHProvenance       = "windows PDH \\Memory and \\Paging File counters"
	windowsCommitProvenance    = "windows GlobalMemoryStatusEx commit charge"
	windowsLowMemoryProvenance = "windows QueryMemoryResourceNotification(LowMemoryResourceNotification)"
)

// windowsPagingSample is one raw read of the Windows memory-pressure sources.
// Each source carries its own error so one missing counter (for example a
// host with no paging file) never hides the others.
type windowsPagingSample struct {
	// PagesInputTotal is the raw cumulative value of \Memory\Pages Input/sec:
	// pages read from disk to resolve hard faults. It includes memory-mapped
	// file reads, not only pagefile reads.
	PagesInputTotal uint64
	PagesInputErr   error
	// PageReadsTotal is the raw cumulative value of \Memory\Page Reads/sec:
	// read operations (each may bring in several pages).
	PageReadsTotal uint64
	PageReadsErr   error
	// PagingFileUsagePercent is \Paging File(_Total)\% Usage.
	PagingFileUsagePercent float64
	PagingFileErr          error
	// CommitCharge and CommitLimit come from GlobalMemoryStatusEx
	// (ullTotalPageFile - ullAvailPageFile, ullTotalPageFile).
	CommitCharge uint64
	CommitLimit  uint64
	CommitErr    error
	// LowMemory is the QueryMemoryResourceNotification state for a
	// LowMemoryResourceNotification handle.
	LowMemory    bool
	LowMemoryErr error
}

func failedSignal(err error, provenance, units string) PressureSignal {
	return PressureSignal{Status: SignalFailed, Reason: err.Error(), Provenance: provenance, Units: units}
}

// windowsPagingReading maps a Windows sample onto the paging seam. Windows
// does not separate pagefile reads from mapped-file reads, so swap_in and
// swap_out stay unsupported rather than borrowing Pages Input/sec.
func windowsPagingReading(sample windowsPagingSample) pagingReading {
	reading := pagingReading{
		counters:     map[string]uint64{},
		rateCounters: []string{"pages_input", "page_reads"},
		aliases:      map[string]string{"pages_input": "page_ins", "page_reads": "page_reads"},
		unsupportedAliases: map[string]string{
			"swap_in":        "Windows counts pagefile and memory-mapped file reads together in Pages Input/sec, so swap-in cannot be isolated",
			"swap_out":       "pagefile writes are not separated from mapped-file writes on Windows",
			"major_faults":   "Windows has no host-wide major-fault counter; see page_ins (includes mapped-file reads)",
			"compressions":   "memory-compression counters are not collected on Windows",
			"decompressions": "memory-compression counters are not collected on Windows",
		},
		gauges:     map[string]PressureSignal{},
		provenance: windowsPDHProvenance,
	}
	if sample.PagesInputErr == nil {
		reading.counters["pages_input"] = sample.PagesInputTotal
	} else {
		reading.unsupportedAliases["page_ins"] = "\\Memory\\Pages Input/sec: " + sample.PagesInputErr.Error()
	}
	if sample.PageReadsErr == nil {
		reading.counters["page_reads"] = sample.PageReadsTotal
	} else {
		reading.unsupportedAliases["page_reads"] = "\\Memory\\Page Reads/sec: " + sample.PageReadsErr.Error()
	}
	if sample.PagingFileErr == nil {
		reading.gauges[SignalPagingFileUsagePercent] = measuredSignal(sample.PagingFileUsagePercent, windowsPDHProvenance, "percent")
	} else {
		reading.gauges[SignalPagingFileUsagePercent] = failedSignal(sample.PagingFileErr, windowsPDHProvenance, "percent")
	}
	switch {
	case sample.CommitErr != nil:
		reading.gauges[SignalCommitChargePercent] = failedSignal(sample.CommitErr, windowsCommitProvenance, "percent")
	case sample.CommitLimit == 0 || sample.CommitCharge > sample.CommitLimit:
		reading.gauges[SignalCommitChargePercent] = PressureSignal{Status: SignalFailed, Reason: "GlobalMemoryStatusEx returned an invalid commit limit", Provenance: windowsCommitProvenance, Units: "percent"}
	default:
		reading.gauges[SignalCommitChargePercent] = measuredSignal(float64(sample.CommitCharge)/float64(sample.CommitLimit)*100, windowsCommitProvenance, "percent")
	}
	if sample.LowMemoryErr == nil {
		level := 0.0
		if sample.LowMemory {
			level = 1
		}
		reading.gauges[SignalNativePressureLevel] = measuredSignal(level, windowsLowMemoryProvenance, "native level")
	} else {
		reading.gauges[SignalNativePressureLevel] = failedSignal(sample.LowMemoryErr, windowsLowMemoryProvenance, "native level")
	}
	reading.supported = len(reading.counters) > 0
	if !reading.supported {
		reading.reason = "no PDH paging counter could be read"
	}
	return reading
}
