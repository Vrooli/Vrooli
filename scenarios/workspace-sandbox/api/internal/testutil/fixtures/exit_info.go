package fixtures

import (
	"time"

	"workspace-sandbox/internal/process"
)

// ExitInfoOpt mutates an ExitInfo during construction.
type ExitInfoOpt func(*process.ExitInfo)

// NewExitInfo returns an ExitInfo carrying the given exit code with
// StoppedAt=now. Use WithSignal / WithOOMKilled for non-default cases.
func NewExitInfo(code int, opts ...ExitInfoOpt) process.ExitInfo {
	e := process.ExitInfo{
		ExitCode:  code,
		StoppedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

// WithSignal records that the process was killed by a signal.
func WithSignal(sig int) ExitInfoOpt {
	return func(e *process.ExitInfo) { e.Signal = sig }
}

// WithOOMKilled flags the exit as an OOM-kill (kernel-reported).
func WithOOMKilled(b bool) ExitInfoOpt {
	return func(e *process.ExitInfo) { e.OOMKilled = b }
}

// WithStoppedAt overrides the default StoppedAt timestamp.
func WithStoppedAt(ts time.Time) ExitInfoOpt {
	return func(e *process.ExitInfo) { e.StoppedAt = ts }
}
