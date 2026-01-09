package nodejs

import (
	"time"

	"test-genie/internal/structure/types"
)

// Re-export shared types for convenience within this package.
type (
	FailureClass    = types.FailureClass
	ObservationType = types.ObservationType
	Observation     = types.Observation
	Result          = types.Result
)

// Re-export constants.
const (
	FailureClassNone              = types.FailureClassNone
	FailureClassMisconfiguration  = types.FailureClassMisconfiguration
	FailureClassSystem            = types.FailureClassSystem
	FailureClassMissingDependency = "missing_dependency"

	ObservationSection = types.ObservationSection
	ObservationSuccess = types.ObservationSuccess
	ObservationWarning = types.ObservationWarning
	ObservationError   = types.ObservationError
	ObservationInfo    = types.ObservationInfo
	ObservationSkip    = types.ObservationSkip
)

// Re-export constructor functions.
var (
	NewSectionObservation = types.NewSectionObservation
	NewSuccessObservation = types.NewSuccessObservation
	NewWarningObservation = types.NewWarningObservation
	NewErrorObservation   = types.NewErrorObservation
	NewInfoObservation    = types.NewInfoObservation
	NewSkipObservation    = types.NewSkipObservation

	OK                   = types.OK
	OKWithCount          = types.OKWithCount
	Fail                 = types.Fail
	FailMisconfiguration = types.FailMisconfiguration
	FailSystem           = types.FailSystem
)

// FailMissingDependency creates a missing dependency failure.
func FailMissingDependency(err error, remediation string) Result {
	return Fail(err, FailureClassMissingDependency, remediation)
}

// BenchmarkResult represents the outcome of a Node.js build benchmark.
type BenchmarkResult struct {
	Result

	// Duration is how long the build took.
	Duration time.Duration

	// Skipped indicates the build was skipped (no UI workspace found).
	Skipped bool

	// PackageManager is the detected package manager (pnpm, yarn, npm).
	PackageManager string
}
