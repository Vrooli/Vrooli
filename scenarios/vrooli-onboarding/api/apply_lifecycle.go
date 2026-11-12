package main

import "fmt"

type applyItemOutcome string

const (
	applyItemPending          applyItemOutcome = "pending"
	applyItemApplying         applyItemOutcome = "applying"
	applyItemApplied          applyItemOutcome = "applied"
	applyItemAlreadySatisfied applyItemOutcome = "already_satisfied"
	applyItemBlocked          applyItemOutcome = "blocked"
	applyItemFailed           applyItemOutcome = "failed"
	applyItemTimedOut         applyItemOutcome = "timed_out"
	applyItemNeedsElevation   applyItemOutcome = "needs_elevation"
	applyItemNotApplicable    applyItemOutcome = "not_applicable"
	applyItemSkippedSelf      applyItemOutcome = "skipped_self"
)

var applyItemOutcomeTransitions = map[applyItemOutcome]map[applyItemOutcome]struct{}{
	applyItemPending: {
		applyItemApplying: {}, applyItemAlreadySatisfied: {}, applyItemBlocked: {},
		applyItemNotApplicable: {}, applyItemSkippedSelf: {},
	},
	applyItemApplying: {
		applyItemApplied: {}, applyItemFailed: {}, applyItemTimedOut: {}, applyItemNeedsElevation: {},
	},
	applyItemApplied:          {},
	applyItemAlreadySatisfied: {},
	applyItemBlocked:          {},
	applyItemFailed:           {},
	applyItemTimedOut:         {},
	applyItemNeedsElevation:   {},
	applyItemNotApplicable:    {},
	applyItemSkippedSelf:      {},
}

func (outcome applyItemOutcome) Valid() bool {
	_, ok := applyItemOutcomeTransitions[outcome]
	return ok
}

func (result *applyItemResult) TransitionOutcome(next applyItemOutcome) error {
	if result == nil {
		return fmt.Errorf("apply item result is required")
	}
	if !result.Outcome.Valid() {
		return fmt.Errorf("apply item has invalid current outcome %q", result.Outcome)
	}
	if !next.Valid() {
		return fmt.Errorf("invalid apply item outcome %q", next)
	}
	if result.Outcome == next {
		return nil
	}
	if _, allowed := applyItemOutcomeTransitions[result.Outcome][next]; !allowed {
		return fmt.Errorf("invalid apply item outcome transition %q -> %q", result.Outcome, next)
	}
	result.Outcome = next
	return nil
}

// applyRunStatus is the durable run lifecycle. Its string encoding is part of
// the on-disk JSON contract, so adding the type does not require a data rewrite.
type applyRunStatus string

const (
	applyRunPending                 applyRunStatus = "pending"
	applyRunApplying                applyRunStatus = "applying"
	applyRunAlreadySatisfied        applyRunStatus = "already_satisfied"
	applyRunIndeterminate           applyRunStatus = "indeterminate"
	applyRunCancelled               applyRunStatus = "cancelled"
	applyRunPartiallyApplied        applyRunStatus = "partially_applied"
	applyRunApplied                 applyRunStatus = "applied"
	applyRunConfigurationIncomplete applyRunStatus = "configuration_incomplete"
	applyRunFailed                  applyRunStatus = "failed"
	applyRunInterrupted             applyRunStatus = "interrupted"
)

var applyRunTransitions = map[applyRunStatus]map[applyRunStatus]struct{}{
	applyRunPending: {
		applyRunApplying:         {},
		applyRunAlreadySatisfied: {},
		applyRunFailed:           {},
		applyRunIndeterminate:    {},
	},
	applyRunApplying: {
		applyRunCancelled:               {},
		applyRunPartiallyApplied:        {},
		applyRunApplied:                 {},
		applyRunConfigurationIncomplete: {},
		applyRunIndeterminate:           {},
		applyRunInterrupted:             {},
	},
	applyRunAlreadySatisfied:        {applyRunIndeterminate: {}},
	applyRunCancelled:               {applyRunIndeterminate: {}},
	applyRunPartiallyApplied:        {applyRunIndeterminate: {}},
	applyRunApplied:                 {applyRunIndeterminate: {}},
	applyRunConfigurationIncomplete: {applyRunIndeterminate: {}},
	applyRunFailed:                  {applyRunIndeterminate: {}},
	applyRunInterrupted:             {applyRunIndeterminate: {}},
	applyRunIndeterminate:           {},
}

func (status applyRunStatus) Valid() bool {
	_, ok := applyRunTransitions[status]
	return ok
}

func (status applyRunStatus) Persistable() bool {
	return status.Valid() && status != applyRunInterrupted
}

func (status applyRunStatus) Terminal() bool {
	switch status {
	case applyRunPending, applyRunApplying:
		return false
	case applyRunAlreadySatisfied, applyRunCancelled, applyRunPartiallyApplied,
		applyRunApplied, applyRunConfigurationIncomplete, applyRunFailed, applyRunIndeterminate,
		applyRunInterrupted:
		return true
	default:
		// Unknown durable values are invalid and must be retained for recovery.
		return false
	}
}

func (run *applyRun) TransitionStatus(next applyRunStatus) error {
	if run == nil {
		return fmt.Errorf("apply run is required")
	}
	if !run.Status.Valid() {
		return fmt.Errorf("apply run has invalid current status %q", run.Status)
	}
	if !next.Valid() {
		return fmt.Errorf("invalid apply run status %q", next)
	}
	if run.Status == next {
		return nil
	}
	if _, allowed := applyRunTransitions[run.Status][next]; !allowed {
		return fmt.Errorf("invalid apply run transition %q -> %q", run.Status, next)
	}
	run.Status = next
	return nil
}
