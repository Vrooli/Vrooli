package dependencies

import (
	"fmt"
	"strings"

	"test-genie/internal/structure/types"
)

// Re-export shared types from structure/types for consistency across packages.
// This allows the dependencies package to use the same observation and result
// types as the structure package.
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

// FailureClassMissingDependency indicates a required dependency is missing.
// This is specific to the dependencies package.
const FailureClassMissingDependency FailureClass = "missing_dependency"

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

// RunResult represents the complete outcome of running all dependency validations.
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
	CommandsChecked  int
	RuntimesDetected int
	ManagersDetected int
	ResourcesChecked int
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	return s.CommandsChecked + s.RuntimesDetected + s.ManagersDetected + s.ResourcesChecked
}

// String returns a human-readable summary of validation counts.
func (s ValidationSummary) String() string {
	parts := []string{}
	if s.CommandsChecked > 0 {
		parts = append(parts, fmt.Sprintf("%d commands", s.CommandsChecked))
	}
	if s.RuntimesDetected > 0 {
		parts = append(parts, fmt.Sprintf("%d runtimes", s.RuntimesDetected))
	}
	if s.ManagersDetected > 0 {
		parts = append(parts, fmt.Sprintf("%d managers", s.ManagersDetected))
	}
	if s.ResourcesChecked > 0 {
		parts = append(parts, fmt.Sprintf("%d resources", s.ResourcesChecked))
	}
	if len(parts) == 0 {
		return "no checks performed"
	}
	return strings.Join(parts, ", ")
}
