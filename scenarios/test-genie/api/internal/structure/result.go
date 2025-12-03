package structure

import "fmt"

// FailureClass categorizes the type of validation failure.
type FailureClass string

const (
	// FailureClassNone indicates no failure occurred.
	FailureClassNone FailureClass = ""
	// FailureClassMisconfiguration indicates the scenario is misconfigured.
	FailureClassMisconfiguration FailureClass = "misconfiguration"
	// FailureClassSystem indicates a system-level error (I/O, permissions, etc).
	FailureClassSystem FailureClass = "system"
)

// ObservationType categorizes the kind of observation.
type ObservationType int

const (
	ObservationSection ObservationType = iota
	ObservationSuccess
	ObservationWarning
	ObservationError
	ObservationInfo
	ObservationSkip
)

// Observation represents a single validation observation with formatting hints.
type Observation struct {
	Type    ObservationType
	Icon    string
	Message string
}

// NewSectionObservation creates a section header observation.
func NewSectionObservation(icon, message string) Observation {
	return Observation{Type: ObservationSection, Icon: icon, Message: message}
}

// NewSuccessObservation creates a success observation.
func NewSuccessObservation(message string) Observation {
	return Observation{Type: ObservationSuccess, Message: message}
}

// NewWarningObservation creates a warning observation.
func NewWarningObservation(message string) Observation {
	return Observation{Type: ObservationWarning, Message: message}
}

// NewErrorObservation creates an error observation.
func NewErrorObservation(message string) Observation {
	return Observation{Type: ObservationError, Message: message}
}

// NewInfoObservation creates an informational observation.
func NewInfoObservation(message string) Observation {
	return Observation{Type: ObservationInfo, Message: message}
}

// NewSkipObservation creates a skip observation.
func NewSkipObservation(message string) Observation {
	return Observation{Type: ObservationSkip, Message: message}
}

// Result represents the outcome of a validation step.
type Result struct {
	// Success indicates whether the validation passed.
	Success bool

	// Error contains the validation error, if any.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains detailed validation observations.
	Observations []Observation

	// ItemsChecked tracks how many items were validated.
	ItemsChecked int
}

// OK creates a successful result.
func OK() Result {
	return Result{Success: true}
}

// OKWithCount creates a successful result with item count.
func OKWithCount(count int) Result {
	return Result{Success: true, ItemsChecked: count}
}

// Fail creates a failure result with the given error and remediation.
func Fail(err error, class FailureClass, remediation string) Result {
	return Result{
		Success:      false,
		Error:        err,
		FailureClass: class,
		Remediation:  remediation,
	}
}

// FailMisconfiguration creates a misconfiguration failure.
func FailMisconfiguration(err error, remediation string) Result {
	return Fail(err, FailureClassMisconfiguration, remediation)
}

// FailSystem creates a system-level failure.
func FailSystem(err error, remediation string) Result {
	return Fail(err, FailureClassSystem, remediation)
}

// WithObservations adds observations to the result.
func (r Result) WithObservations(obs ...Observation) Result {
	r.Observations = append(r.Observations, obs...)
	return r
}

// RunResult represents the complete outcome of running all structure validations.
type RunResult struct {
	// Success indicates whether all validations passed.
	Success bool

	// Error contains the first validation error encountered.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains all validation observations.
	Observations []Observation

	// Summary provides counts of items checked per category.
	Summary ValidationSummary
}

// ValidationSummary tracks validation counts by category.
type ValidationSummary struct {
	DirsChecked    int
	FilesChecked   int
	JSONFilesValid int
}

// TotalChecks returns the total number of items checked.
func (s ValidationSummary) TotalChecks() int {
	return s.DirsChecked + s.FilesChecked + s.JSONFilesValid
}

// String returns a human-readable summary.
func (s ValidationSummary) String() string {
	return fmt.Sprintf("%d dirs, %d files, %d JSON files",
		s.DirsChecked, s.FilesChecked, s.JSONFilesValid)
}
