package business

import (
	"fmt"

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
	FailureClassNone             = shared.FailureClassNone
	FailureClassMisconfiguration = shared.FailureClassMisconfiguration
	FailureClassSystem           = shared.FailureClassSystem

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

	OK                   = shared.OK
	OKWithCount          = shared.OKWithCount
	Fail                 = shared.Fail
	FailMisconfiguration = shared.FailMisconfiguration
	FailSystem           = shared.FailSystem
)

// RunResult is an alias for the generic shared.RunResult with ValidationSummary.
type RunResult = shared.RunResult[ValidationSummary]

// ValidationSummary tracks validation counts by category.
type ValidationSummary struct {
	ModulesFound      int
	RequirementsFound int
	ValidationErrors  int
	ValidationWarns   int
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	return s.ModulesFound + s.RequirementsFound
}

// String returns a human-readable summary.
func (s ValidationSummary) String() string {
	return fmt.Sprintf("%d modules, %d requirements, %d errors, %d warnings",
		s.ModulesFound, s.RequirementsFound, s.ValidationErrors, s.ValidationWarns)
}
