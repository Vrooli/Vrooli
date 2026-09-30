package compiler

import "github.com/vrooli/browser-automation-studio/automation/contracts"

// Utility wrappers for type conversion used across the compiler package.
// Delegates to the existing automation contracts value converters.

// toInt32 delegates to contracts.ToInt32 for numeric conversion.
func toInt32(v any) (int32, bool) {
	return contracts.ToInt32(v)
}

// toFloat64 delegates to contracts.ToFloat64 for numeric conversion.
func toFloat64(v any) (float64, bool) {
	return contracts.ToFloat64(v)
}
