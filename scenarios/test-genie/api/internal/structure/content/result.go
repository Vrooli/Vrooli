package content

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

// NewInfoObservation creates an informational observation.
func NewInfoObservation(message string) Observation {
	return Observation{Type: ObservationInfo, Message: message}
}

// Result represents the outcome of a content validation step.
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

// FailMisconfiguration creates a misconfiguration failure.
func FailMisconfiguration(err error, remediation string) Result {
	return Result{
		Success:      false,
		Error:        err,
		FailureClass: FailureClassMisconfiguration,
		Remediation:  remediation,
	}
}

// FailSystem creates a system-level failure.
func FailSystem(err error, remediation string) Result {
	return Result{
		Success:      false,
		Error:        err,
		FailureClass: FailureClassSystem,
		Remediation:  remediation,
	}
}

// WithObservations adds observations to the result.
func (r Result) WithObservations(obs ...Observation) Result {
	r.Observations = append(r.Observations, obs...)
	return r
}
