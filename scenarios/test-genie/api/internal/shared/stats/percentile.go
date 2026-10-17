// Package stats contains the statistical primitives used by Test Genie's
// schedulers and reports.
package stats

import "math"

// Ordered is the set of scalar values for which Percentile can select an
// observed value.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// NearestRankIndex returns the zero-based inclusive nearest-rank index for p.
// The selected value is always one of the observed values in the sorted input.
// For p=0.9, this convention selects the maximum when n <= 9; that is the
// defined nearest-rank result for small samples, not an interpolation.
func NearestRankIndex(n int, p float64) int {
	if n <= 1 {
		return 0
	}
	index := int(math.Ceil(p*float64(n))) - 1
	if index < 0 {
		return 0
	}
	if index >= n {
		return n - 1
	}
	return index
}

// Percentile selects the inclusive nearest-rank percentile from a sorted
// ascending slice. It returns false when sorted is empty.
func Percentile[T Ordered](sorted []T, p float64) (T, bool) {
	if len(sorted) == 0 {
		var zero T
		return zero, false
	}
	return sorted[NearestRankIndex(len(sorted), p)], true
}

// PercentileValue returns the inclusive nearest-rank percentile, or the zero
// value when sorted is empty. It is a convenience for aggregate records whose
// empty-bucket representation is already zero.
func PercentileValue[T Ordered](sorted []T, p float64) T {
	value, _ := Percentile(sorted, p)
	return value
}
