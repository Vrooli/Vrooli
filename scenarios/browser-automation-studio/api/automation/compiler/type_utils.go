package compiler

import "github.com/vrooli/browser-automation-studio/internal/typeconv"

// Utility wrappers for type conversion used across the compiler package.
// Delegates to internal/typeconv for actual implementation.

// toInt32 delegates to typeconv.ToInt32 for numeric conversion.
func toInt32(v any) (int32, bool) {
	return typeconv.ToInt32(v)
}

// toFloat64 delegates to typeconv.ToFloat64 for numeric conversion.
func toFloat64(v any) (float64, bool) {
	return typeconv.ToFloat64(v)
}
