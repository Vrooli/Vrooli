package discovery

import "test-genie/internal/shared"

// Re-export shared types for convenience within this package.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	Result          = shared.Result
)

// Re-export constants.
const (
	FailureClassNone             = shared.FailureClassNone
	FailureClassMisconfiguration = shared.FailureClassMisconfiguration
	FailureClassSystem           = shared.FailureClassSystem

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

	OK                   = shared.OK
	OKWithCount          = shared.OKWithCount
	Fail                 = shared.Fail
	FailMisconfiguration = shared.FailMisconfiguration
	FailSystem           = shared.FailSystem
)

// DiscoveryResult extends Result with discovery-specific data.
type DiscoveryResult struct {
	Result

	// ModuleCount is the number of module files discovered (excluding index files).
	ModuleCount int

	// Files contains all discovered files.
	Files []DiscoveredFile
}

// DiscoveredFile represents a found requirement file.
// Re-exported from requirements/discovery for convenience.
type DiscoveredFile struct {
	AbsolutePath string
	RelativePath string
	IsIndex      bool
	ModuleDir    string
}
