package sessions

import (
	"context"
	"testing"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	"web-console/backends/codex"
)

type nativeGoalOwnerFixture struct {
	ManagedCodexService
	origin codex.GoalMutationOrigin
}

func (f *nativeGoalOwnerFixture) SetGoal(_ context.Context, _, _, _, _, _ string, origin codex.GoalMutationOrigin, _ *int64) (codex.GoalResult, error) {
	f.origin = origin
	return codex.GoalResult{ThreadID: "thread", Objective: "provider-retained objective", Status: "paused"}, nil
}
func TestNativeGoalAdapterPreservesProviderReturnedObjective(t *testing.T) {
	owner := &nativeGoalOwnerFixture{}
	adapter := &Adapter{ManagedCodex: owner}
	got, err := adapter.SetNativeGoal(context.Background(), &sessionsv1.SetNativeGoalRequest{SessionId: "session", ExpectedNativeOwner: "owner", ExpectedThreadId: "thread", Status: "paused", Origin: sessionsv1.NativeGoalMutationOrigin_NATIVE_GOAL_MUTATION_ORIGIN_AUTOMATIC})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetThreadId() != "thread" || got.GetObjective() != "provider-retained objective" || got.GetStatus() != "paused" || owner.origin != codex.GoalOriginAutomatic {
		t.Fatalf("provider acknowledgment/provenance lost: %+v origin=%q", got, owner.origin)
	}
}
