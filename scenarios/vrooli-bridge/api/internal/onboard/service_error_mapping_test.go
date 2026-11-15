package onboard_test

import (
	"testing"

	"connectrpc.com/connect"
	"vrooli-bridge/internal/onboard"
)

func TestAttemptTransitionRefusalMapsToFailedPrecondition(t *testing.T) {
	err := onboard.ToConnectError(onboard.ErrInvalidAttemptTransition{
		Current: onboard.AttemptSucceeded, Attempted: onboard.AttemptFailed,
	})
	if code := connect.CodeOf(err); code != connect.CodeFailedPrecondition {
		t.Fatalf("transition code = %s, want failed precondition", code)
	}
}
