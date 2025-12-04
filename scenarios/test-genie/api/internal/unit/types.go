package unit

import (
	"test-genie/internal/unit/types"
)

// Re-export shared types for convenience.
// This allows code to use unit.Result, unit.Observation, etc.
type (
	FailureClass    = types.FailureClass
	ObservationType = types.ObservationType
	Observation     = types.Observation
	Result          = types.Result
	LanguageRunner  = types.LanguageRunner
	CommandExecutor = types.CommandExecutor
	DefaultExecutor = types.DefaultExecutor
)

// Re-export constants.
const (
	FailureClassNone              = types.FailureClassNone
	FailureClassMisconfiguration  = types.FailureClassMisconfiguration
	FailureClassMissingDependency = types.FailureClassMissingDependency
	FailureClassTestFailure       = types.FailureClassTestFailure
	FailureClassSystem            = types.FailureClassSystem

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
