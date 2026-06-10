package business

import (
	"fmt"

	reqparsing "test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
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

// RunResult mirrors shared.RunResult[ValidationSummary] and additionally
// carries the structured validation output (Issues + parsed Index) so the
// business phase can produce typed ArchitectureFindings without re-running
// discovery/parsing. It is a named struct (not the shared alias) because
// the generic shared type cannot grow phase-specific fields.
type RunResult struct {
	// Success indicates whether all validations passed.
	Success bool

	// Error contains the first error encountered.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains all validation observations.
	Observations []Observation

	// Summary provides validation counts.
	Summary ValidationSummary

	// Issues is the structured structural-validation output (one entry
	// per rule violation), populated whenever validation ran.
	Issues []types.ValidationIssue

	// Index is the parsed requirements module index, populated whenever
	// parsing succeeded. Downstream finding producers read requirement
	// metadata (validations, prd_ref, tags, criticality) from it.
	Index *reqparsing.ModuleIndex
}

// ValidationSummary tracks validation counts by category.
type ValidationSummary struct {
	ModulesFound      int
	RequirementsFound int
	ValidationErrors  int
	ValidationWarns   int
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	return s.ModulesFound + s.RequirementsFound
}

// String returns a human-readable summary.
func (s ValidationSummary) String() string {
	return fmt.Sprintf("%d modules, %d requirements, %d errors, %d warnings",
		s.ModulesFound, s.RequirementsFound, s.ValidationErrors, s.ValidationWarns)
}
