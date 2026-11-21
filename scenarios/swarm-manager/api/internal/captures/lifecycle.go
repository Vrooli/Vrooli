package captures

import (
	"errors"
	"fmt"
)

type CaptureStatus string

const (
	CaptureStatusClassifying CaptureStatus = "classifying"
	CaptureStatusFailed      CaptureStatus = "failed"
)

var ErrInvalidCaptureTransition = errors.New("invalid capture status transition")

// TransitionStatus owns the durable capture state changes. A successful
// classification removes the intake record after proposals are recorded, so
// it does not persist a "classified" state.
func (c *capture) TransitionStatus(next CaptureStatus) error {
	current := CaptureStatus(c.Status)
	if current == next && next == CaptureStatusClassifying {
		return nil // Classify is idempotently re-enterable while work is active.
	}
	allowed := current == CaptureStatusClassifying && next == CaptureStatusFailed ||
		current == CaptureStatusFailed && next == CaptureStatusClassifying
	if !allowed {
		return fmt.Errorf("capture status %q -> %q is not allowed: %w", current, next, ErrInvalidCaptureTransition)
	}
	c.Status = string(next)
	return nil
}

func (h *Handler) failCapture(cap *capture, reason string) error {
	if err := cap.TransitionStatus(CaptureStatusFailed); err != nil {
		return err
	}
	cap.FailureReason = reason
	return h.writeCapture(cap)
}
