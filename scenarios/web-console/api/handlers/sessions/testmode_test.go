package sessions

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/database"

	"web-console/internal/backend"
	"web-console/internal/policy"
)

// A test-lease create must never produce a persistent (tmux-backed) session.
// A persistent session outlives the API process and is re-adopted by startup
// recovery, so a leaked one is permanent clutter on the operator's tmux server
// — exactly the failure this shape exists to prevent.
func TestApplyTestLeaseShape_ForcesStandardBackend(t *testing.T) {
	ctx := database.WithTestMode(context.Background())
	never := policy.Policy{Mode: policy.Never}

	bid, pol := applyTestLeaseShape(ctx, backend.Persistent, &never)

	if bid != backend.Standard {
		t.Errorf("backend = %q, want %q", bid, backend.Standard)
	}
	if pol == nil {
		t.Fatal("policy must not be nil under a test lease")
	}
	if pol.Mode == policy.Never {
		t.Error("test-lease session inherited a never-expire policy; a leak would be permanent")
	}
}

// The forced expiry must be a policy the domain actually accepts, otherwise
// Create rejects every test-lease session and the BAS suite fails closed for
// the wrong reason.
func TestApplyTestLeaseShape_ForcedPolicyIsValid(t *testing.T) {
	ctx := database.WithTestMode(context.Background())

	_, pol := applyTestLeaseShape(ctx, backend.Standard, nil)

	if pol == nil {
		t.Fatal("expected a forced policy")
	}
	if err := policy.Validate(*pol); err != nil {
		t.Fatalf("forced test-lease policy is invalid: %v", err)
	}
	if ttl := policy.ResolveTTL(*pol); ttl <= 0 {
		t.Errorf("forced policy TTL = %v, want a positive expiry", ttl)
	}
}

// An operator's own request carries no test-mode marker and must pass through
// completely untouched — same backend, same policy pointer.
func TestApplyTestLeaseShape_LeavesOperatorRequestsAlone(t *testing.T) {
	requested := policy.Policy{Mode: policy.Never}

	bid, pol := applyTestLeaseShape(context.Background(), backend.Persistent, &requested)

	if bid != backend.Persistent {
		t.Errorf("backend = %q, want %q (operator request must not be rewritten)", bid, backend.Persistent)
	}
	if pol != &requested {
		t.Error("operator policy was replaced; the requested policy must survive untouched")
	}
}

// A nil policy on a non-test request means "use the server default"; the shape
// helper must not invent one.
func TestApplyTestLeaseShape_PreservesNilPolicyOutsideTestMode(t *testing.T) {
	if _, pol := applyTestLeaseShape(context.Background(), backend.Standard, nil); pol != nil {
		t.Errorf("policy = %+v, want nil preserved", pol)
	}
}
