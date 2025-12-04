package validation

import (
	"test-genie/internal/requirements/types"
	structtypes "test-genie/internal/structure/types"
)

// Re-export shared types for convenience within this package.
type (
	FailureClass    = structtypes.FailureClass
	ObservationType = structtypes.ObservationType
	Observation     = structtypes.Observation
	Result          = structtypes.Result
)

// Re-export constants.
const (
	FailureClassNone             = structtypes.FailureClassNone
	FailureClassMisconfiguration = structtypes.FailureClassMisconfiguration
	FailureClassSystem           = structtypes.FailureClassSystem

	ObservationSection = structtypes.ObservationSection
	ObservationSuccess = structtypes.ObservationSuccess
	ObservationWarning = structtypes.ObservationWarning
	ObservationError   = structtypes.ObservationError
	ObservationInfo    = structtypes.ObservationInfo
	ObservationSkip    = structtypes.ObservationSkip
)

// Re-export constructor functions.
var (
	NewSectionObservation = structtypes.NewSectionObservation
	NewSuccessObservation = structtypes.NewSuccessObservation
	NewWarningObservation = structtypes.NewWarningObservation
	NewErrorObservation   = structtypes.NewErrorObservation
	NewInfoObservation    = structtypes.NewInfoObservation
	NewSkipObservation    = structtypes.NewSkipObservation

	OK                   = structtypes.OK
	OKWithCount          = structtypes.OKWithCount
	Fail                 = structtypes.Fail
	FailMisconfiguration = structtypes.FailMisconfiguration
	FailSystem           = structtypes.FailSystem
)

// ValidationResult extends Result with validation-specific data.
type ValidationResult struct {
	Result

	// ErrorCount is the number of validation errors.
	ErrorCount int

	// WarningCount is the number of validation warnings.
	WarningCount int

	// Issues contains all validation issues (for detailed reporting).
	Issues []types.ValidationIssue
}
