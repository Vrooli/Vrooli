package services

import "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/maputil"

// Re-export maputil functions as package-level aliases for backward compatibility
// within the services package. These delegate to the shared maputil package.
var (
	getFloat64Value = maputil.GetFloat64
	getIntValue     = maputil.GetInt
	getInt64Value   = maputil.GetInt64
	getFloat64Slice = maputil.GetFloat64Slice
	getStringValue  = maputil.GetString
	getBoolValue    = maputil.GetBool
)
