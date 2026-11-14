//go:build windows

package health

// processAccounting is intentionally unavailable on Windows: the syscall
// package exposes no getrusage equivalent. The health response omits the two
// accounting metrics rather than reporting a fabricated value.
func processAccounting() (cpuSeconds float64, peakResidentMB float64, ok bool) {
	return 0, 0, false
}
