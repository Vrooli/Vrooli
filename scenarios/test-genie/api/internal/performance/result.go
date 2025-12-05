package performance

import (
	"fmt"
	"time"

	"test-genie/internal/shared"
)

// Re-export shared types for convenience.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	Result          = shared.Result
)

// Re-export constants.
const (
	FailureClassNone              = shared.FailureClassNone
	FailureClassMisconfiguration  = shared.FailureClassMisconfiguration
	FailureClassMissingDependency = shared.FailureClassMissingDependency
	FailureClassSystem            = shared.FailureClassSystem

	ObservationSection = shared.ObservationSection
	ObservationSuccess = shared.ObservationSuccess
	ObservationWarning = shared.ObservationWarning
	ObservationError   = shared.ObservationError
	ObservationInfo    = shared.ObservationInfo
	ObservationSkip    = shared.ObservationSkip
)

// Re-export constructor functions.
var (
	NewSectionObservation = shared.NewSectionObservation
	NewSuccessObservation = shared.NewSuccessObservation
	NewWarningObservation = shared.NewWarningObservation
	NewErrorObservation   = shared.NewErrorObservation
	NewInfoObservation    = shared.NewInfoObservation
	NewSkipObservation    = shared.NewSkipObservation

	OK                    = shared.OK
	OKWithCount           = shared.OKWithCount
	Fail                  = shared.Fail
	FailMisconfiguration  = shared.FailMisconfiguration
	FailMissingDependency = shared.FailMissingDependency
	FailSystem            = shared.FailSystem
)

// RunResult is an alias for the generic shared.RunResult with BenchmarkSummary.
type RunResult = shared.RunResult[BenchmarkSummary]

// BenchmarkSummary tracks benchmark results.
type BenchmarkSummary struct {
	GoBuildDuration time.Duration
	GoBuildPassed   bool
	UIBuildDuration time.Duration
	UIBuildPassed   bool
	UIBuildSkipped  bool

	// Lighthouse audit results
	LighthousePassed  bool
	LighthouseSkipped bool
	LighthousePages   int // Number of pages audited
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

	// Base summary
	summary := fmt.Sprintf("Go build: %s (%s), UI build: %s (%s)",
		s.GoBuildDuration.Round(time.Second), goStatus,
		s.UIBuildDuration.Round(time.Second), uiStatus)

	// Add Lighthouse summary if applicable
	if s.LighthouseSkipped {
		summary += ", Lighthouse: skipped"
	} else if s.LighthousePages > 0 {
		lhStatus := "passed"
		if !s.LighthousePassed {
			lhStatus = "failed"
		}
		summary += fmt.Sprintf(", Lighthouse: %s (%d pages)", lhStatus, s.LighthousePages)
	}

	return summary
}
