package policy

import (
	"context"
	"errors"
	"os"
	"time"
)

// RightsContextSource loads the trusted operator-owned business context. A
// request payload must never supply this value.
type RightsContextSource func(context.Context) (RightsContext, error)

// RightsEvaluator is the shared current-context contract for model selection,
// execution, fallback and packaging callers.
type RightsEvaluator interface {
	Evaluate(context.Context, *ArtifactRights, RightsUse) (RightsDecision, error)
	EvaluateAt(context.Context, *ArtifactRights, RightsUse, time.Time) (RightsDecision, error)
	EvaluateWithContext(context.Context, *ArtifactRights, RightsUse, time.Time) (RightsContext, RightsDecision, error)
	RequireCurrent(context.Context, RightsDecision, *ArtifactRights, RightsUse) error
	RequireCurrentWithContext(context.Context, RightsDecision, *ArtifactRights, RightsUse, RightsContext) error
}

// CurrentRightsEvaluator loads the trusted context for each evaluation so a
// context flip cannot be hidden by a cached decision.
type CurrentRightsEvaluator struct {
	contextSource RightsContextSource
	now           func() time.Time
}

// NewRightsEvaluator constructs the shared evaluator with an injected trusted
// context source. The source is called on every Evaluate and RequireCurrent.
func NewRightsEvaluator(source RightsContextSource) *CurrentRightsEvaluator {
	return &CurrentRightsEvaluator{contextSource: source, now: time.Now}
}

// NewDefaultRightsEvaluator uses the canonical operator context path contract.
// getenv is injectable for tests; nil uses the current process environment.
func NewDefaultRightsEvaluator(getenv func(string) string) *CurrentRightsEvaluator {
	if getenv == nil {
		getenv = os.Getenv
	}
	return NewRightsEvaluator(func(context.Context) (RightsContext, error) {
		return LoadDefaultRightsContext(getenv)
	})
}

// Evaluate evaluates current rights against the latest trusted business
// context. Context loading errors return an explicit unknown decision and the
// source error, so callers can report the evidence gap while failing closed.
func (e *CurrentRightsEvaluator) Evaluate(ctx context.Context, artifact *ArtifactRights, use RightsUse) (RightsDecision, error) {
	return e.EvaluateAt(ctx, artifact, use, e.currentTime())
}

// EvaluateAt evaluates using a caller-bound timestamp. This supports receipt
// verification at a previously admitted decision time while still loading the
// latest trusted context at the owner boundary.
func (e *CurrentRightsEvaluator) EvaluateAt(ctx context.Context, artifact *ArtifactRights, use RightsUse, at time.Time) (RightsDecision, error) {
	_, decision, err := e.EvaluateWithContext(ctx, artifact, use, at)
	return decision, err
}

// EvaluateWithContext returns the exact trusted context used to build the
// decision receipt, avoiding a second read that could cross a context change.
func (e *CurrentRightsEvaluator) EvaluateWithContext(ctx context.Context, artifact *ArtifactRights, use RightsUse, at time.Time) (RightsContext, RightsDecision, error) {
	if at.IsZero() {
		at = e.currentTime()
	}
	if e == nil || e.contextSource == nil {
		return RightsContext{}, unknownContextDecision(artifact, use, at), errors.New("trusted model rights context source is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return RightsContext{}, unknownContextDecision(artifact, use, at), err
	}
	current, err := e.contextSource(ctx)
	if err != nil {
		return RightsContext{}, unknownContextDecision(artifact, use, at), err
	}
	return current, EvaluateRights(artifact, use, current, at), nil
}

// RequireCurrent rechecks that a prior decision is still valid for the latest
// trusted context, artifact facts and use at an execution boundary.
func (e *CurrentRightsEvaluator) RequireCurrent(ctx context.Context, decision RightsDecision, artifact *ArtifactRights, use RightsUse) error {
	if e == nil || e.contextSource == nil {
		return errors.New("trusted model rights context source is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := e.contextSource(ctx)
	if err != nil {
		return err
	}
	return e.RequireCurrentWithContext(ctx, decision, artifact, use, current)
}

// RequireCurrentWithContext verifies a caller's just-read trusted context. It
// is for owner paths that need the same snapshot for their receipt or audit.
func (e *CurrentRightsEvaluator) RequireCurrentWithContext(ctx context.Context, decision RightsDecision, artifact *ArtifactRights, use RightsUse, current RightsContext) error {
	if e == nil || e.contextSource == nil {
		return errors.New("trusted model rights context source is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return RequireCurrentRightsDecision(decision, artifact, use, current)
}

func (e *CurrentRightsEvaluator) currentTime() time.Time {
	if e != nil && e.now != nil {
		return e.now()
	}
	return time.Now()
}

func unknownContextDecision(artifact *ArtifactRights, use RightsUse, now time.Time) RightsDecision {
	decision := EvaluateRights(artifact, use, RightsContext{}, now)
	decision.Reason = "trusted_model_rights_context_unavailable"
	return bindDecision(decision)
}
