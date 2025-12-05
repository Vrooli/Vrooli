package dependencies

import (
	"fmt"
	"strings"

	"test-genie/internal/shared"
)

// Re-export shared types for consistency across packages.
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
	FailureClassSystem            = shared.FailureClassSystem
	FailureClassMissingDependency = shared.FailureClassMissingDependency

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

// RunResult is an alias for the generic shared.RunResult with ValidationSummary.
type RunResult = shared.RunResult[ValidationSummary]

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
