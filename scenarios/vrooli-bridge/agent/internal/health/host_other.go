//go:build !linux && !darwin

package health

// readHostMetrics has no cheap reader on this platform yet; the memory and
// swap observations then read Unknown instead of guessing.
func readHostMetrics() hostMetrics {
	return hostMetrics{
		PSISomeAvg60:  -1,
		ReadError:     "memory is not measured on this platform yet",
		SwapReadError: "swap is not measured on this platform yet",
	}
}
