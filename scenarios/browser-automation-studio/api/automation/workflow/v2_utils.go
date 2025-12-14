package workflow

import (
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Utility wrappers for type conversion used across the workflow package.
// Delegates to internal/typeconv for actual implementation.

// toInt32 delegates to typeconv.ToInt32 for numeric conversion.
func toInt32(v any) (int32, bool) {
	return typeconv.ToInt32(v)
}

// toFloat64 delegates to typeconv.ToFloat64 for numeric conversion.
func toFloat64(v any) (float64, bool) {
	return typeconv.ToFloat64(v)
}

// anyToJsonValue delegates to typeconv.AnyToJsonValue for proto conversion.
func anyToJsonValue(v any) *commonv1.JsonValue {
	return typeconv.AnyToJsonValue(v)
}

// jsonValueToAny delegates to typeconv.JsonValueToAny for proto conversion.
func jsonValueToAny(v *commonv1.JsonValue) any {
	return typeconv.JsonValueToAny(v)
}
