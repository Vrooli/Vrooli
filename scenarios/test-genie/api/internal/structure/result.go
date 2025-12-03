package structure

import (
	"fmt"

	"test-genie/internal/structure/types"
)

// Re-export shared types for convenience.
// This allows existing code to use structure.Result, structure.Observation, etc.
type (
	FailureClass    = types.FailureClass
	ObservationType = types.ObservationType
	Observation     = types.Observation
	Result          = types.Result
)

// Re-export constants.
const (
	FailureClassNone             = types.FailureClassNone
	FailureClassMisconfiguration = types.FailureClassMisconfiguration
	FailureClassSystem           = types.FailureClassSystem

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

// RunResult represents the complete outcome of running all structure validations.
// This is specific to the structure package's Runner.
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

	// Summary provides counts of items checked per category.
	Summary ValidationSummary
}

// ValidationSummary tracks validation counts by category.
type ValidationSummary struct {
	DirsChecked    int
	FilesChecked   int
	JSONFilesValid int
	SmokeChecked   bool
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	total := s.DirsChecked + s.FilesChecked + s.JSONFilesValid
	if s.SmokeChecked {
		total++
	}
	return total
}

// String returns a human-readable summary.
func (s ValidationSummary) String() string {
	parts := fmt.Sprintf("%d dirs, %d files, %d JSON files",
		s.DirsChecked, s.FilesChecked, s.JSONFilesValid)
	if s.SmokeChecked {
		parts += ", smoke test"
	}
	return parts
}
