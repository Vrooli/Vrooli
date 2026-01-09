package types

import "fmt"

// ResultBuilder provides a fluent interface for building unit test Results.
// This reduces boilerplate when constructing results with multiple observations.
type ResultBuilder struct {
	result Result
}

// NewResultBuilder creates a new ResultBuilder with default values.
func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		result: Result{Success: true},
	}
}

// Success marks the result as successful.
func (b *ResultBuilder) Success() *ResultBuilder {
	b.result.Success = true
	return b
}

// Fail marks the result as failed with the given error, class, and remediation.
func (b *ResultBuilder) Fail(err error, class FailureClass, remediation string) *ResultBuilder {
	b.result.Success = false
	b.result.Error = err
	b.result.FailureClass = class
	b.result.Remediation = remediation
	return b
}

// FailMisconfiguration marks the result as a misconfiguration failure.
func (b *ResultBuilder) FailMisconfiguration(err error, remediation string) *ResultBuilder {
	return b.Fail(err, FailureClassMisconfiguration, remediation)
}

// FailMissingDependency marks the result as a missing dependency failure.
func (b *ResultBuilder) FailMissingDependency(err error, remediation string) *ResultBuilder {
	return b.Fail(err, FailureClassMissingDependency, remediation)
}

// FailTestFailure marks the result as a test failure.
func (b *ResultBuilder) FailTestFailure(err error, remediation string) *ResultBuilder {
	return b.Fail(err, FailureClassTestFailure, remediation)
}

// FailSystem marks the result as a system-level failure.
func (b *ResultBuilder) FailSystem(err error, remediation string) *ResultBuilder {
	return b.Fail(err, FailureClassSystem, remediation)
}

// WithCoverage sets the coverage percentage.
func (b *ResultBuilder) WithCoverage(coverage string) *ResultBuilder {
	b.result.Coverage = coverage
	return b
}

// Skip marks the result as skipped with the given reason.
func (b *ResultBuilder) Skip(reason string) *ResultBuilder {
	b.result.Success = true
	b.result.Skipped = true
	b.result.SkipReason = reason
	return b
}

// Skipf marks the result as skipped with a formatted reason.
func (b *ResultBuilder) Skipf(format string, args ...interface{}) *ResultBuilder {
	return b.Skip(fmt.Sprintf(format, args...))
}

// AddObservation adds an observation to the result.
func (b *ResultBuilder) AddObservation(obs Observation) *ResultBuilder {
	b.result.Observations = append(b.result.Observations, obs)
	return b
}

// AddSection adds a section header observation.
func (b *ResultBuilder) AddSection(icon, message string) *ResultBuilder {
	return b.AddObservation(NewSectionObservation(icon, message))
}

// AddSuccess adds a success observation.
func (b *ResultBuilder) AddSuccess(message string) *ResultBuilder {
	return b.AddObservation(NewSuccessObservation(message))
}

// AddSuccessf adds a formatted success observation.
func (b *ResultBuilder) AddSuccessf(format string, args ...interface{}) *ResultBuilder {
	return b.AddSuccess(fmt.Sprintf(format, args...))
}

// AddWarning adds a warning observation.
func (b *ResultBuilder) AddWarning(message string) *ResultBuilder {
	return b.AddObservation(NewWarningObservation(message))
}

// AddWarningf adds a formatted warning observation.
func (b *ResultBuilder) AddWarningf(format string, args ...interface{}) *ResultBuilder {
	return b.AddWarning(fmt.Sprintf(format, args...))
}

// AddError adds an error observation.
func (b *ResultBuilder) AddError(message string) *ResultBuilder {
	return b.AddObservation(NewErrorObservation(message))
}

// AddErrorf adds a formatted error observation.
func (b *ResultBuilder) AddErrorf(format string, args ...interface{}) *ResultBuilder {
	return b.AddError(fmt.Sprintf(format, args...))
}

// AddInfo adds an informational observation.
func (b *ResultBuilder) AddInfo(message string) *ResultBuilder {
	return b.AddObservation(NewInfoObservation(message))
}

// AddInfof adds a formatted informational observation.
func (b *ResultBuilder) AddInfof(format string, args ...interface{}) *ResultBuilder {
	return b.AddInfo(fmt.Sprintf(format, args...))
}

// AddSkip adds a skip observation.
func (b *ResultBuilder) AddSkip(message string) *ResultBuilder {
	return b.AddObservation(NewSkipObservation(message))
}

// AddSkipf adds a formatted skip observation.
func (b *ResultBuilder) AddSkipf(format string, args ...interface{}) *ResultBuilder {
	return b.AddSkip(fmt.Sprintf(format, args...))
}

// Build returns the constructed Result.
func (b *ResultBuilder) Build() Result {
	return b.result
}
