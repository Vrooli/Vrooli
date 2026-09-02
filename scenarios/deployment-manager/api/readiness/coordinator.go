package readiness

import (
	"context"
	"fmt"
	"time"
)

type ReleaseAnchor interface {
	SetReadinessApproval(context.Context, string, string, string) error
}

type Coordinator struct {
	Goals  goalOpener
	Anchor ReleaseAnchor
}

// TriggeredSpec evaluates the declared trigger set before opening a goal. The
// first fired trigger is retained in the goal metadata so the review explains
// why it was required; an empty set supports voluntary operator reviews.
func TriggeredSpec(spec GoalSpec, inputs []TriggerInput, now time.Time) (GoalSpec, bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, input := range inputs {
		if input.Fired(now) {
			spec.Trigger = string(input.Kind)
			return spec, true, nil
		}
	}
	return spec, false, nil
}

// OpenAndAnchor opens or links the swarm goal and records its deterministic
// reference on the deployment-manager release. The release is not marked
// approved here; approval is written only after swarm-manager closes the goal.
func (c *Coordinator) OpenAndAnchor(ctx context.Context, releaseID string, spec GoalSpec) (string, bool, error) {
	if c == nil || c.Goals == nil || c.Anchor == nil {
		return "", false, fmt.Errorf("readiness coordinator is not configured")
	}
	goalRef, deduped, err := c.Goals.Open(ctx, spec)
	if err != nil {
		return "", false, err
	}
	if err := c.Anchor.SetReadinessApproval(ctx, releaseID, goalRef, ""); err != nil {
		return "", false, fmt.Errorf("anchor readiness goal: %w", err)
	}
	return goalRef, deduped, nil
}

// OpenTriggered evaluates the declared release inputs and opens a goal only
// when a trigger fires. Callers may use OpenAndAnchor directly for a voluntary
// review, which intentionally has no trigger.
func (c *Coordinator) OpenTriggered(ctx context.Context, releaseID string, spec GoalSpec, inputs []TriggerInput, now time.Time) (string, bool, bool, error) {
	spec, fired, err := TriggeredSpec(spec, inputs, now)
	if err != nil || !fired {
		return "", false, fired, err
	}
	goal, deduped, err := c.OpenAndAnchor(ctx, releaseID, spec)
	return goal, deduped, true, err
}

// RecordApproval is called only after swarm-manager reports the goal closed.
// It writes the exact deployed commit, making a later commit automatically
// stale at the release gate.
func (c *Coordinator) RecordApproval(ctx context.Context, releaseID, goalRef, commit string) error {
	if c == nil || c.Anchor == nil {
		return fmt.Errorf("readiness coordinator is not configured")
	}
	if goalRef == "" || commit == "" {
		return fmt.Errorf("goal reference and commit are required")
	}
	return c.Anchor.SetReadinessApproval(ctx, releaseID, goalRef, commit)
}
