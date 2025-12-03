package content

import "test-genie/internal/structure/types"

// Re-export shared types for convenience within this package.
// This allows existing code to use content.Result, content.Observation, etc.
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
