// This file schedules supervisor review wakes through the durable park owner.
package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// ProducerReviewTimer uses the existing durable park owner for a supervisor's
// next review. The key is the absolute review time, so restart recovery never
// restarts the interval from zero.
const ProducerReviewTimer = "review-timer"

type reviewTimerWaiter struct{}

func NewReviewTimerWaiter() Waiter { return reviewTimerWaiter{} }

func (reviewTimerWaiter) Producer() string { return ProducerReviewTimer }

func parseReviewTime(key string) (time.Time, error) {
	at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(key))
	if err != nil {
		return time.Time{}, domain.NewValidationError("key", "review-timer key must be an RFC3339 timestamp")
	}
	return at.UTC(), nil
}

func validateReviewTimer(key string, now, deadline time.Time) error {
	at, err := parseReviewTime(key)
	if err != nil {
		return err
	}
	// The agent must have time to finish its parked turn before it can resume.
	if !at.After(now.Add(DefaultParkTurnEndGrace)) {
		return domain.NewValidationError("key", "review time must allow the parked turn to end")
	}
	if !deadline.After(at) {
		return domain.NewValidationError("deadline", "park deadline must be later than the review time")
	}
	return nil
}

func (reviewTimerWaiter) Wait(ctx context.Context, key string) (string, error) {
	at, err := parseReviewTime(key)
	if err != nil {
		return "", err
	}
	timer := time.NewTimer(time.Until(at))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return fmt.Sprintf("Review time %s reached. Re-read available owner evidence; this wake does not establish worker health or completion.", at.Format(time.RFC3339Nano)), nil
	}
}
