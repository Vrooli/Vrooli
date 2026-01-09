package structure

import (
	"fmt"

	"test-genie/internal/shared"
)

// Re-export shared types for convenience.
// This allows existing code to use structure.Result, structure.Observation, etc.
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
	DirsChecked    int
	FilesChecked   int
	JSONFilesValid int
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	return s.DirsChecked + s.FilesChecked + s.JSONFilesValid
}

// String returns a human-readable summary.
func (s ValidationSummary) String() string {
	return fmt.Sprintf("%d dirs, %d files, %d JSON files",
		s.DirsChecked, s.FilesChecked, s.JSONFilesValid)
}
