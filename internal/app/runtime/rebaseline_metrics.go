package runtimeapp

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/internal/rootmetrics"
)

type rootAttempt struct {
	*rootmetrics.Attempt
	serving, candidate string
	candidateFallback  string
}

func beginRootAttempt(home, kind string) *rootAttempt {
	return &rootAttempt{Attempt: rootmetrics.Begin(home, kind)}
}
func (a *rootAttempt) event(stage, reason string)           { a.Attempt.Event(stage, reason) }
func (a *rootAttempt) identities(serving, candidate string) { a.Attempt.Identities(serving, candidate) }

func (a *rootAttempt) attributeLater(serving, candidate string) {
	if a.candidateFallback != "" {
		candidate = a.candidateFallback
	}
	a.serving, a.candidate = serving, candidate
}

// Finish runs after transition/publication locks have been released. Binding
// probes to the already-observed contracts rejects images changed meanwhile.
func (a *rootAttempt) finish(ctx context.Context, outcome string) {
	if a.serving != "" && ctx != nil {
		attribution, cancel := context.WithTimeout(ctx, 5*time.Second)
		a.Attempt.Attribute(attribution, a.serving, a.candidate)
		cancel()
	}
	a.event("terminal", outcome)
	a.Attempt.Flush()
}
