package execution

import (
	"context"
	"errors"
)

// TerminalOutcome classifies the run-level result persisted on a
// suite_executions row. It is the denominator vocabulary the reliability ledger
// aggregates over: every terminal run writes exactly one row carrying one of
// these values.
type TerminalOutcome string

const (
	// TerminalOutcomePassed is a run that completed with all phases succeeding.
	TerminalOutcomePassed TerminalOutcome = "passed"
	// TerminalOutcomeFailed is a run that completed but had failing phases.
	TerminalOutcomeFailed TerminalOutcome = "failed"
	// TerminalOutcomeErrored is a run that never produced a result because the
	// engine returned an error (or a nil result) — a catastrophic outcome that,
	// before this column, was invisible to availability metrics.
	TerminalOutcomeErrored TerminalOutcome = "errored"
	// TerminalOutcomeAborted is a run cancelled via context (operator abort).
	TerminalOutcomeAborted TerminalOutcome = "aborted"
	// TerminalOutcomeTimeout is a run whose context deadline elapsed.
	TerminalOutcomeTimeout TerminalOutcome = "timeout"
)

// String returns the wire/storage token.
func (o TerminalOutcome) String() string { return string(o) }

// outcomeForSuccess maps a completed run's success flag to its terminal outcome.
func outcomeForSuccess(success bool) TerminalOutcome {
	if success {
		return TerminalOutcomePassed
	}
	return TerminalOutcomeFailed
}

// classifyTerminalError derives a catastrophic terminal outcome (no result was
// produced) from the engine error and the surrounding context. Context
// cancellation and deadline are distinguished from a generic engine error so the
// ledger can separate operator aborts and timeouts from genuine failures.
func classifyTerminalError(ctx context.Context, err error) TerminalOutcome {
	if ctx != nil {
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return TerminalOutcomeTimeout
		case context.Canceled:
			return TerminalOutcomeAborted
		}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return TerminalOutcomeTimeout
	case errors.Is(err, context.Canceled):
		return TerminalOutcomeAborted
	default:
		return TerminalOutcomeErrored
	}
}
