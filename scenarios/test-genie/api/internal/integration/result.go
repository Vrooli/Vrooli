package integration

import (
	"fmt"

	"test-genie/internal/structure/types"
)

// Re-export shared types for convenience.
// This allows code to use integration.Result, integration.Observation, etc.
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

// Additional failure class for missing dependencies (like bats).
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

// RunResult represents the complete outcome of running all integration validations.
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
	APIHealthChecked     bool
	CLIValidated         bool
	PrimaryBatsRan       bool
	AdditionalBatsSuites int
	WebSocketValidated   bool
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	total := s.AdditionalBatsSuites
	if s.APIHealthChecked {
		total += 2 // health endpoint status + response time
	}
	if s.CLIValidated {
		total += 3 // binary exists, help works, version works
	}
	if s.PrimaryBatsRan {
		total++
	}
	if s.WebSocketValidated {
		total += 2 // connection + optional ping-pong
	}
	return total
}

// String returns a human-readable summary.
func (s ValidationSummary) String() string {
	var parts []string
	if s.APIHealthChecked {
		parts = append(parts, "API health")
	}
	if s.CLIValidated {
		parts = append(parts, "CLI validated")
	}
	if s.PrimaryBatsRan {
		parts = append(parts, "primary bats suite")
	}
	if s.AdditionalBatsSuites > 0 {
		parts = append(parts, fmt.Sprintf("%d additional bats suites", s.AdditionalBatsSuites))
	}
	if s.WebSocketValidated {
		parts = append(parts, "WebSocket")
	}
	if len(parts) == 0 {
		return "no checks performed"
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
