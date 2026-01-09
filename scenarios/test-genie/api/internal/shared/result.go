package shared

// RunResult is a generic result type for phase runners.
// TSummary is the phase-specific summary type (e.g., ValidationSummary, BenchmarkSummary).
type RunResult[TSummary any] struct {
	// Success indicates whether all validations/tests passed.
	Success bool

	// Error contains the first error encountered.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains all validation observations.
	Observations []Observation

	// Summary provides phase-specific counts/metrics.
	Summary TSummary
}

// FailureClass categorizes the type of validation failure.
type FailureClass string

const (
	// FailureClassNone indicates no failure occurred.
	FailureClassNone FailureClass = ""
	// FailureClassMisconfiguration indicates the scenario is misconfigured.
	FailureClassMisconfiguration FailureClass = "misconfiguration"
	// FailureClassMissingDependency indicates a required dependency is missing.
	FailureClassMissingDependency FailureClass = "missing_dependency"
	// FailureClassTestFailure indicates tests failed.
	FailureClassTestFailure FailureClass = "test_failure"
	// FailureClassExecution indicates an execution failure.
	FailureClassExecution FailureClass = "execution"
	// FailureClassTimeout indicates a timeout occurred.
	FailureClassTimeout FailureClass = "timeout"
	// FailureClassSystem indicates a system-level error (I/O, permissions, etc).
	FailureClassSystem FailureClass = "system"
)

// Observation represents a single validation observation with formatting hints.
type Observation struct {
	Type    ObservationType
	Icon    string
	Message string
}

// GetType returns the observation type for interface compatibility.
func (o Observation) GetType() ObservationType { return o.Type }

// GetIcon returns the observation icon.
func (o Observation) GetIcon() string { return o.Icon }

// GetMessage returns the observation message.
func (o Observation) GetMessage() string { return o.Message }

// NewSectionObservation creates a section header observation.
func NewSectionObservation(icon, message string) Observation {
	return Observation{Type: ObservationSection, Icon: icon, Message: message}
}

// NewSuccessObservation creates a success observation.
// Note: Success observations don't set icon by default to match legacy behavior.
func NewSuccessObservation(message string) Observation {
	return Observation{Type: ObservationSuccess, Message: message}
}

// NewWarningObservation creates a warning observation.
func NewWarningObservation(message string) Observation {
	return Observation{Type: ObservationWarning, Icon: "⚠️", Message: message}
}

// NewErrorObservation creates an error observation.
func NewErrorObservation(message string) Observation {
	return Observation{Type: ObservationError, Icon: "❌", Message: message}
}

// NewInfoObservation creates an informational observation.
func NewInfoObservation(message string) Observation {
	return Observation{Type: ObservationInfo, Icon: "ℹ️", Message: message}
}

// NewSkipObservation creates a skip observation.
func NewSkipObservation(message string) Observation {
	return Observation{Type: ObservationSkip, Icon: "⏭️", Message: message}
}

// Result represents the outcome of a single validation step.
// This is used by validators within phases.
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

// FailMissingDependency creates a missing dependency failure.
func FailMissingDependency(err error, remediation string) Result {
	return Fail(err, FailureClassMissingDependency, remediation)
}

// FailTestFailure creates a test failure result.
func FailTestFailure(err error, remediation string) Result {
	return Fail(err, FailureClassTestFailure, remediation)
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

// Summary is the interface that phase summaries should implement.
type Summary interface {
	// TotalChecks returns the total number of items checked.
	TotalChecks() int
	// String returns a human-readable summary.
	String() string
}

// SuccessResult creates a successful RunResult with the given observations and summary.
func SuccessResult[TSummary any](observations []Observation, summary TSummary) *RunResult[TSummary] {
	return &RunResult[TSummary]{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// FailureResult creates a failed RunResult with the given error and context.
func FailureResult[TSummary any](err error, class FailureClass, remediation string, observations []Observation, summary TSummary) *RunResult[TSummary] {
	return &RunResult[TSummary]{
		Success:      false,
		Error:        err,
		FailureClass: class,
		Remediation:  remediation,
		Observations: observations,
		Summary:      summary,
	}
}

// SystemFailure creates a system-level failure RunResult.
func SystemFailure[TSummary any](err error) *RunResult[TSummary] {
	return &RunResult[TSummary]{
		Success:      false,
		Error:        err,
		FailureClass: FailureClassSystem,
	}
}
