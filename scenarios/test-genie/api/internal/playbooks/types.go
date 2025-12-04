package playbooks

// Re-export types from the types sub-package for convenience.
// This allows consumers to use playbooks.Entry instead of playbooks/types.Entry.

import "test-genie/internal/playbooks/types"

// Type aliases for convenient access
type (
	Registry         = types.Registry
	Entry            = types.Entry
	ExecutionStatus  = types.ExecutionStatus
	Outcome          = types.Outcome
	Result           = types.Result
	Observation      = types.Observation
	ObservationType  = types.ObservationType
	FailureClass     = types.FailureClass
	RunResult        = types.RunResult
	ExecutionSummary = types.ExecutionSummary
)

// Re-export constants
const (
	ObservationInfo    = types.ObservationInfo
	ObservationSuccess = types.ObservationSuccess
	ObservationWarning = types.ObservationWarning
	ObservationError   = types.ObservationError
	ObservationSkip    = types.ObservationSkip
	ObservationSection = types.ObservationSection

	FailureClassMisconfiguration  = types.FailureClassMisconfiguration
	FailureClassMissingDependency = types.FailureClassMissingDependency
	FailureClassSystem            = types.FailureClassSystem
	FailureClassExecution         = types.FailureClassExecution
)

// Re-export observation constructors
var (
	NewSuccessObservation = types.NewSuccessObservation
	NewWarningObservation = types.NewWarningObservation
	NewErrorObservation   = types.NewErrorObservation
	NewInfoObservation    = types.NewInfoObservation
	NewSkipObservation    = types.NewSkipObservation
	NewSectionObservation = types.NewSectionObservation
)

// Re-export result builders
var (
	OK                    = types.OK
	OKWithResults         = types.OKWithResults
	Fail                  = types.Fail
	FailMisconfiguration  = types.FailMisconfiguration
	FailMissingDependency = types.FailMissingDependency
	FailSystem            = types.FailSystem
	FailExecution         = types.FailExecution
)
