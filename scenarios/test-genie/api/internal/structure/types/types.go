// Package types re-exports shared types for backward compatibility.
// New code should import from test-genie/internal/shared directly.
package types

import "test-genie/internal/shared"

// Re-export types from shared package for backward compatibility.
type (
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	FailureClass    = shared.FailureClass
	Result          = shared.Result
)

// Re-export constants from shared package.
const (
	ObservationSection = shared.ObservationSection
	ObservationSuccess = shared.ObservationSuccess
	ObservationWarning = shared.ObservationWarning
	ObservationError   = shared.ObservationError
	ObservationInfo    = shared.ObservationInfo
	ObservationSkip    = shared.ObservationSkip

	FailureClassNone             = shared.FailureClassNone
	FailureClassMisconfiguration = shared.FailureClassMisconfiguration
	FailureClassSystem           = shared.FailureClassSystem
)

// Re-export constructor functions from shared package.
var (
	NewSectionObservation = shared.NewSectionObservation
	NewSuccessObservation = shared.NewSuccessObservation
	NewWarningObservation = shared.NewWarningObservation
	NewErrorObservation   = shared.NewErrorObservation
	NewInfoObservation    = shared.NewInfoObservation
	NewSkipObservation    = shared.NewSkipObservation

	OK                   = shared.OK
	OKWithCount          = shared.OKWithCount
	Fail                 = shared.Fail
	FailMisconfiguration = shared.FailMisconfiguration
	FailSystem           = shared.FailSystem
)
