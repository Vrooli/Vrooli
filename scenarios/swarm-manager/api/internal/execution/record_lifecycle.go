package execution

import (
	"fmt"
	"strings"
)

// ErrInvalidRecordStatusTransition marks an execution record write that is
// outside the lifecycle table. Readers may still encounter historical state
// during migrations; normal service mutations must pass through TransitionStatus.
var ErrInvalidRecordStatusTransition = fmt.Errorf("invalid execution status transition")

var recordStatusTransitions = map[Status]map[Status]struct{}{
	StatusPending: statusEdges(StatusStarting, StatusCancelling, StatusCanceled, StatusFailed),
	StatusStarting: statusEdges(
		StatusRunning, StatusCancelling, StatusNeedsReview, StatusValidating,
		StatusNeedsFixup, StatusCompleted, StatusAbstained, StatusBudgetExhausted,
		StatusInterrupted, StatusNeedsAttention, StatusFailed, StatusCanceled,
	),
	StatusRunning: statusEdges(
		StatusCancelling, StatusNeedsReview, StatusValidating, StatusNeedsFixup,
		StatusCompleted, StatusAbstained, StatusBudgetExhausted, StatusInterrupted,
		StatusNeedsAttention, StatusFailed, StatusCanceled,
	),
	StatusCancelling: statusEdges(StatusCanceled, StatusInterrupted, StatusFailed),
	StatusNeedsReview: statusEdges(
		StatusCancelling, StatusNeedsAttention, StatusValidating, StatusNeedsFixup,
		StatusCompleted, StatusAbstained, StatusInterrupted, StatusFailed, StatusCanceled,
	),
	StatusValidating:  statusEdges(StatusNeedsFixup, StatusCompleted, StatusFailed, StatusCanceled),
	StatusNeedsFixup:  statusEdges(StatusStarting, StatusCompleted, StatusFailed, StatusCanceled),
	StatusInterrupted: statusEdges(StatusStarting, StatusCanceled),
	StatusNeedsAttention: statusEdges(
		StatusStarting, StatusCancelling, StatusFailed, StatusCanceled,
	),
	StatusFailed:          statusEdges(StatusStarting, StatusValidating, StatusCompleted),
	StatusAbstained:       statusEdges(StatusStarting),
	StatusBudgetExhausted: statusEdges(StatusStarting),
	// Spec-sync's archive step follows workflow success; if archive or cleanup
	// fails, the durable execution reports the failed overall operation.
	StatusCompleted: statusEdges(StatusFailed, StatusValidating),
}

func statusEdges(values ...Status) map[Status]struct{} {
	edges := make(map[Status]struct{}, len(values))
	for _, value := range values {
		edges[value] = struct{}{}
	}
	return edges
}

// Valid reports whether status is part of the execution-record lifecycle.
func (status Status) Valid() bool {
	switch status {
	case StatusPending, StatusStarting, StatusRunning, StatusCancelling,
		StatusNeedsReview, StatusValidating, StatusNeedsFixup, StatusCompleted,
		StatusAbstained, StatusBudgetExhausted, StatusInterrupted,
		StatusNeedsAttention, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// TransitionStatus is the only production owner for changing an existing
// execution record's lifecycle. Creation initializes pending directly and
// legacy migration normalizes old values before this method is used.
func (record *Record) TransitionStatus(next Status) error {
	if record == nil {
		return fmt.Errorf("%w: nil execution record", ErrInvalidRecordStatusTransition)
	}
	if !record.Status.Valid() || !next.Valid() {
		return fmt.Errorf("%w: %q -> %q contains an unknown state", ErrInvalidRecordStatusTransition, record.Status, next)
	}
	if record.Status == next {
		return nil
	}
	if _, ok := recordStatusTransitions[record.Status][next]; !ok {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidRecordStatusTransition, record.Status, next)
	}
	record.Status = next
	return nil
}

// NormalizePersistedStatus owns the one legacy status rewrite at the storage
// boundary. Other record status changes go through TransitionStatus.
func (record *Record) NormalizePersistedStatus() bool {
	if record == nil {
		return false
	}
	if record.Status == "scheduled" {
		record.Status = StatusPending
		return true
	}
	if record.Status == StatusRunning && strings.TrimSpace(record.RunID) == "" {
		return record.TransitionStatus(StatusFailed) == nil
	}
	return false
}
