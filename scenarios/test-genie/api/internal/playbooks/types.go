package playbooks

// Re-export types from the types sub-package for convenience.
// This allows consumers to use playbooks.Entry instead of playbooks/types.Entry.

import "test-genie/internal/playbooks/types"

// Type aliases for convenient access
type (
	Registry               = types.Registry
	Entry                  = types.Entry
	ExecutionStatus        = types.ExecutionStatus
	Outcome                = types.Outcome
	Result                 = types.Result
	Observation            = types.Observation
	ObservationType        = types.ObservationType
	FailureClass           = types.FailureClass
	RunResult              = types.RunResult
	ExecutionSummary       = types.ExecutionSummary
	PlaybookExecutionError = types.PlaybookExecutionError
	ExecutionArtifacts     = types.ExecutionArtifacts
	ExecutionPhase         = types.ExecutionPhase
)

// Re-export constants
// NOTE: The ordering matches shared.ObservationType for ExtractStandardObservation compatibility.
const (
	ObservationSection = types.ObservationSection
	ObservationSuccess = types.ObservationSuccess
	ObservationWarning = types.ObservationWarning
	ObservationError   = types.ObservationError
	ObservationInfo    = types.ObservationInfo
	ObservationSkip    = types.ObservationSkip

	FailureClassMisconfiguration  = types.FailureClassMisconfiguration
	FailureClassMissingDependency = types.FailureClassMissingDependency
	FailureClassSystem            = types.FailureClassSystem
	FailureClassExecution         = types.FailureClassExecution

	PhaseResolve  = types.PhaseResolve
	PhaseExecute  = types.PhaseExecute
	PhaseWait     = types.PhaseWait
	PhaseArtifact = types.PhaseArtifact
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

// Re-export error constructors
var (
	NewResolveError  = types.NewResolveError
	NewExecuteError  = types.NewExecuteError
	NewWaitError     = types.NewWaitError
	NewArtifactError = types.NewArtifactError
)
