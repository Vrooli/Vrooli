package export

import (
	"github.com/google/uuid"
	basexport "github.com/vrooli/browser-automation-studio/services/export"
)

// ErrMovieSpecUnavailable is returned when no movie spec is available for export.
// This is a re-export of services/export.ErrMovieSpecUnavailable for backward compatibility.
var ErrMovieSpecUnavailable = basexport.ErrMovieSpecUnavailable

// BuildSpec constructs a validated ReplayMovieSpec for export by merging client-provided
// and server-generated specs, validating execution ID matching, and filling in defaults.
//
// This is a thin wrapper that delegates to services/export.BuildSpec.
func BuildSpec(baseline, incoming *basexport.ReplayMovieSpec, executionID uuid.UUID) (*basexport.ReplayMovieSpec, error) {
	return basexport.BuildSpec(baseline, incoming, executionID)
}

// Clone creates a deep copy of a ReplayMovieSpec via JSON marshaling.
//
// This is a thin wrapper that delegates to services/export.Clone.
func Clone(spec *basexport.ReplayMovieSpec) (*basexport.ReplayMovieSpec, error) {
	return basexport.Clone(spec)
}

// Harmonize validates and fills in missing fields in spec using values from baseline.
// It ensures the execution ID matches and applies sensible defaults for all required fields.
//
// This is a thin wrapper that delegates to services/export.Harmonize.
func Harmonize(spec, baseline *basexport.ReplayMovieSpec, executionID uuid.UUID) error {
	return basexport.Harmonize(spec, baseline, executionID)
}
