package sessionpermission

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGrantDurationBudgetAllowsBeforeDeadlineDeniesStaleActionAndAllowsClose(t *testing.T) {
	profile := Profile{
		Effect: EffectObserve, Approval: ApprovalNone, View: true,
		URLAllow: []string{"https://quota.example.test/*"}, MaxDuration: 30 * time.Second,
	}
	policy, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{
		SessionID: "session-owner-a", LeaseID: "lease-a", OwnerID: "owner-a", CallerID: "run-a",
		CurrentURL: "https://quota.example.test/page", Kind: "observe",
	}
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("action before granted duration: %v", err)
	}

	// Make the same bound grant stale without waiting in wall clock time.
	policy.started = time.Now().Add(-31 * time.Second)
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("action after duration budget accepted: %v", err)
	}
	request.Kind = "close"
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("owning caller could not close an expired session: %v", err)
	}
}
