//go:build !linux

package collectors

// readForkRate reports unsupported off Linux. Darwin and Windows expose no
// cheap cumulative process-creation counter, and the alternatives (repeated
// process-table walks, WMI queries) cost more than the signal is worth on a
// per-cycle budget. Reporting "unsupported" keeps the metric honest rather than
// letting a platform without the counter read as a quiet zero.
func readForkRate() forkRateReading {
	return forkRateUnsupported("no cumulative process-creation counter on this operating system")
}
