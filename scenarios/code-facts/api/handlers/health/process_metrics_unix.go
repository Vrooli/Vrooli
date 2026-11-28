//go:build !windows

package health

import "syscall"

// processAccounting reports cumulative CPU time and peak resident memory from
// the kernel's per-process accounting. Windows exposes neither through the
// syscall package, so the metric is reported as unavailable there rather than
// making this package Unix-only.
func processAccounting() (cpuSeconds float64, peakResidentMB float64, ok bool) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0, 0, false
	}
	// Linux reports ru_maxrss in KiB.
	return timevalSeconds(usage.Utime) + timevalSeconds(usage.Stime), float64(usage.Maxrss) / 1024, true
}

func timevalSeconds(value syscall.Timeval) float64 {
	return float64(value.Sec) + float64(value.Usec)/1_000_000
}
