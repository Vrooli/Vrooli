package metrics

import "search-hub/internal/telemetry"

// Schema is retained as a compatibility seam for metrics stores and tests.
// Ownership of the tables lives in the telemetry domain.
func Schema() string { return telemetry.Schema() }
