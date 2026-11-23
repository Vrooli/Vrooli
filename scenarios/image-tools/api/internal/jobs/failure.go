package jobs

import "errors"

// FailureKind says whose fault a failed job was. Verdicts and usage measures
// count only FailureFault against a model: a request the service refused, a
// phrase a parser cannot select or a restart that stopped the job say nothing
// about whether the model does its work.
type FailureKind string

const (
	// FailureFault is the default: the model or the runtime did not do what it was asked.
	FailureFault FailureKind = "fault"
	// FailureRefused is a policy decision: consent not given, an output the safety review refused.
	FailureRefused FailureKind = "refused"
	// FailureInvalidInput is a request that cannot be served as given: a phrase the model has no
	// label for, a malformed prompt, a mask that does not fit the image.
	FailureInvalidInput FailureKind = "invalid_input"
	// FailureInterrupted is the service stopping the job: a restart, a drain.
	FailureInterrupted FailureKind = "interrupted"
	// FailureCapacity is the host not delivering what the lease promised: GPU memory
	// another service took mid-run. The model was not at fault and can run again.
	FailureCapacity FailureKind = "capacity"
)

// MetaFailureKind is the job meta key that records the FailureKind of a failed job.
const MetaFailureKind = "failure_kind"

// A run that fails with FailureCapacity did the model no wrong and can run again once
// the host has the memory back, so the job returns to the queue once, after a backoff,
// instead of failing for a user who waited for it. These keys record that retry.
const (
	// MetaCapacityRetries counts the times the job went back to the queue after its run lost capacity.
	MetaCapacityRetries = "capacity_retries"
	// MetaCapacityRetryReason is the first line of the failure that sent it back.
	MetaCapacityRetryReason = "capacity_retry_reason"
	// maxRunCapacityRetries is how often a job may go back after a capacity failure.
	maxRunCapacityRetries = 1
	// runCapacityBackoff is the wait before a job that lost capacity is admitted again: the
	// service that took the memory needs time to give it back, and re-admission then asks the
	// ledger again for what is actually free.
	runCapacityBackoff = maxAdmissionRetry
)

// CountsAgainstModel reports whether a failure of this kind lowers a model's record.
func (k FailureKind) CountsAgainstModel() bool { return k == FailureFault || k == "" }

// ParseFailureKind reads a recorded kind; an empty or unknown value is a fault, so a
// job that predates the record still counts as it always did.
func ParseFailureKind(value string) FailureKind {
	switch kind := FailureKind(value); kind {
	case FailureRefused, FailureInvalidInput, FailureInterrupted, FailureCapacity:
		return kind
	default:
		return FailureFault
	}
}

// classified is implemented by an error that knows its own kind.
type classified interface{ FailureKind() FailureKind }

type kindedError struct {
	kind FailureKind
	err  error
}

func (e *kindedError) Error() string            { return e.err.Error() }
func (e *kindedError) Unwrap() error            { return e.err }
func (e *kindedError) FailureKind() FailureKind { return e.kind }

// Kinded marks err with kind without changing its message.
func Kinded(kind FailureKind, err error) error {
	if err == nil {
		return nil
	}
	return &kindedError{kind: kind, err: err}
}

// KindOf returns the first kind an error in err's tree declares, else FailureFault.
func KindOf(err error) FailureKind {
	var c classified
	if errors.As(err, &c) {
		return ParseFailureKind(string(c.FailureKind()))
	}
	return FailureFault
}
