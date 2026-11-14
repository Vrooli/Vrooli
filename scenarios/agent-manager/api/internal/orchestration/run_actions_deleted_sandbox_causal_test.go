package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/orchestration/testutil/mocks"
	"github.com/google/uuid"
)

// This control deliberately refers only to the pre-existing action booleans,
// so it compiles against the old source and fails on the stale action projection.
func TestDeletedSandboxDoesNotOfferDecisionActions(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusDeleted}, nil
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
	got, err := o.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Actions.CanApprove || got.Actions.CanReject {
		t.Fatalf("deleted sandbox still offers decision actions: %+v", got.Actions)
	}
}
