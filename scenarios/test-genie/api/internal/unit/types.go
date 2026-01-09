package unit

import (
	"test-genie/internal/shared"
	"test-genie/internal/unit/types"
)

// Re-export shared types for convenience.
// This allows code to use unit.Result, unit.Observation, etc.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	Result          = types.Result
	LanguageRunner  = types.LanguageRunner
	CommandExecutor = types.CommandExecutor
	DefaultExecutor = types.DefaultExecutor
)

// Re-export constants.
const (
	FailureClassNone              = shared.FailureClassNone
	FailureClassMisconfiguration  = shared.FailureClassMisconfiguration
	FailureClassMissingDependency = shared.FailureClassMissingDependency
	FailureClassTestFailure       = shared.FailureClassTestFailure
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

	OK                    = types.OK
	OKWithCoverage        = types.OKWithCoverage
	Skip                  = types.Skip
	Fail                  = types.Fail
	FailMisconfiguration  = types.FailMisconfiguration
	FailMissingDependency = types.FailMissingDependency
	FailTestFailure       = types.FailTestFailure
	FailSystem            = types.FailSystem

	NewDefaultExecutor = types.NewDefaultExecutor
	EnsureCommand      = types.EnsureCommand
)
