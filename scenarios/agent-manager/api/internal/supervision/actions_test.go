package supervision

import (
	"context"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestActionLedgerPersistsEveryTransitionAndIdempotentRequest(t *testing.T) {
	repo, db := testRepository(t)
	watch, _, _, err := repo.Create(context.Background(), watchSpec(), "action-watch", 1)
	if err != nil {
		t.Fatal(err)
	}
	request := &domainpb.RequestCohortWatchActionRequest{WatchId: watch.GetWatchId(), ExpectedWatchRevision: watch.GetRevision(), IdempotencyKey: "action-1", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, TargetRunId: watch.GetSpec().GetSubjects()[0].GetRunId(), RequestedBy: "parent", Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT, Message: "report status"}
	action, replay, err := repo.RequestAction(context.Background(), request)
	if err != nil || replay || action.GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED {
		t.Fatalf("request = %+v replay=%v err=%v", action, replay, err)
	}
	duplicate, replay, err := repo.RequestAction(context.Background(), request)
	if err != nil || !replay || duplicate.GetActionId() != action.GetActionId() {
		t.Fatalf("duplicate = %+v replay=%v err=%v", duplicate, replay, err)
	}
	action, err = repo.TransitionAction(context.Background(), action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED, domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, "authorized")
	if err != nil {
		t.Fatal(err)
	}
	action, err = repo.TransitionAction(context.Background(), action.GetActionId(), domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED, domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED, "safe turn boundary")
	if err != nil || !action.GetAcknowledgedAt().IsValid() {
		t.Fatalf("apply = %+v err=%v", action, err)
	}
	var transitions int
	if err := db.Get(&transitions, `SELECT COUNT(*) FROM cohort_watch_action_transitions WHERE action_id=?`, action.GetActionId()); err != nil || transitions != 3 {
		t.Fatalf("transitions=%d err=%v", transitions, err)
	}
}
