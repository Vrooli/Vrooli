package performance

import (
	"fmt"
	"time"

	"test-genie/internal/structure/types"
)

// Re-export shared types for convenience.
// This allows the phase orchestrator to use performance.Observation, etc.
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

// RunResult represents the complete outcome of running all performance validations.
type RunResult struct {
	// Success indicates whether all validations passed.
	Success bool

	// Error contains the first validation error encountered.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains all validation observations.
	Observations []Observation

	// Summary provides timing metrics for each benchmark.
	Summary BenchmarkSummary
}

// BenchmarkSummary tracks benchmark results.
type BenchmarkSummary struct {
	GoBuildDuration time.Duration
	GoBuildPassed   bool
	UIBuildDuration time.Duration
	UIBuildPassed   bool
	UIBuildSkipped  bool
}

// String returns a human-readable summary.
func (s BenchmarkSummary) String() string {
	goStatus := "passed"
	if !s.GoBuildPassed {
		goStatus = "failed"
	}
	uiStatus := "passed"
	if s.UIBuildSkipped {
		uiStatus = "skipped"
	} else if !s.UIBuildPassed {
		uiStatus = "failed"
	}
	return fmt.Sprintf("Go build: %s (%s), UI build: %s (%s)",
		s.GoBuildDuration.Round(time.Second), goStatus,
		s.UIBuildDuration.Round(time.Second), uiStatus)
}
