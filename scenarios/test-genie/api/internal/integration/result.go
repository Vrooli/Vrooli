package integration

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

// RunResult is an alias for the generic shared.RunResult with ValidationSummary.
type RunResult = shared.RunResult[ValidationSummary]

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
