package supervision

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type fakeActionController struct {
	runs      map[uuid.UUID]*domain.Run
	continued int
	stopped   int
	parked    int
	woken     int
}

func (c *fakeActionController) GetRun(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	return c.runs[id], nil
}
func (c *fakeActionController) ContinueRun(_ context.Context, id uuid.UUID, _, _ string) error {
	c.continued++
	c.runs[id].Status = domain.RunStatusRunning
	return nil
}
func (c *fakeActionController) StopRun(_ context.Context, id uuid.UUID) error {
	c.stopped++
	c.runs[id].Status = domain.RunStatusCancelled
	return nil
}
func (c *fakeActionController) ParkRun(_ context.Context, id uuid.UUID, _ string, _ *time.Time) error {
	c.parked++
	c.runs[id].Status = domain.RunStatusParked
	return nil
}
func (c *fakeActionController) WakeRun(_ context.Context, id uuid.UUID, _ string) error {
	c.woken++
	c.runs[id].Status = domain.RunStatusRunning
	return nil
}

func actionFixture(t *testing.T, childStatus, parentStatus domain.RunStatus) (*ActionService, *Repository, *fakeActionController, *domainpb.CohortWatch, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo, _ := testRepository(t)
	childID, parentID := uuid.New(), uuid.New()
	spec := &domainpb.WatchSpec{FamilyExecutionId: "family-actions", ParentRunId: parentID.String(), Subjects: []*domainpb.WatchSubject{{FamilyExecutionId: "family-actions", PlanId: "plan-a", RunId: childID.String()}}, Triggers: &domainpb.WatchTriggers{Terminal: true}}
	watch, _, _, err := repo.Create(context.Background(), spec, "watch-"+t.Name(), 1)
	if err != nil {
		t.Fatal(err)
	}
	controller := &fakeActionController{runs: map[uuid.UUID]*domain.Run{
		childID:  {ID: childID, Status: childStatus, SessionID: "child-session"},
		parentID: {ID: parentID, Status: parentStatus, SessionID: "parent-session"},
	}}
	return NewActionService(repo, controller), repo, controller, watch, childID, parentID
}

func TestActionAuthorizationAndStateMatrix(t *testing.T) { // [REQ:REQ-P2-009]
	cases := []struct {
		name         string
		kind         domainpb.WatchActionKind
		authority    domainpb.WatchAuthority
		childStatus  domain.RunStatus
		parentStatus domain.RunStatus
		targetParent bool
	}{
		{"observe", domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE, domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM, domain.RunStatusRunning, domain.RunStatusRunning, false},
		{"nudge", domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT, domain.RunStatusNeedsReview, domain.RunStatusRunning, false},
		{"continue", domainpb.WatchActionKind_WATCH_ACTION_KIND_CONTINUE, domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR, domain.RunStatusNeedsReview, domain.RunStatusRunning, false},
		{"stop", domainpb.WatchActionKind_WATCH_ACTION_KIND_STOP, domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT, domain.RunStatusRunning, domain.RunStatusRunning, false},
		{"park", domainpb.WatchActionKind_WATCH_ACTION_KIND_PARK, domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR, domain.RunStatusRunning, domain.RunStatusRunning, true},
		{"escalate", domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE, domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM, domain.RunStatusRunning, domain.RunStatusParked, true},
		{"wake-parent", domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT, domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM, domain.RunStatusRunning, domain.RunStatusParked, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, _, _, watch, childID, parentID := actionFixture(t, tc.childStatus, tc.parentStatus)
			target := childID.String()
			if tc.targetParent {
				target = parentID.String()
			}
			requestedBy := "operator"
			if tc.authority == domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT {
				requestedBy = parentID.String()
			} else if tc.authority == domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM {
				requestedBy = "agent-manager"
			}
			response, err := service.Request(context.Background(), &domainpb.RequestCohortWatchActionRequest{WatchId: watch.GetWatchId(), ExpectedWatchRevision: watch.GetRevision(), IdempotencyKey: "action-" + tc.name, Kind: tc.kind, TargetRunId: target, RequestedBy: requestedBy, Authority: tc.authority, Message: "bounded evidence"})
			if err != nil || response.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED {
				t.Fatalf("response=%+v err=%v", response, err)
			}
		})
	}
}

func TestUnsafeActionsPersistRejectedState(t *testing.T) {
	service, _, _, watch, childID, _ := actionFixture(t, domain.RunStatusComplete, domain.RunStatusRunning)
	cases := []*domainpb.RequestCohortWatchActionRequest{
		{IdempotencyKey: "hard-gate", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, TargetRunId: childID.String(), RequestedBy: "operator", Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR, WaiveHardValidationGate: true},
		{IdempotencyKey: "system-mutation", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_STOP, TargetRunId: childID.String(), RequestedBy: "agent-manager", Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM},
		{IdempotencyKey: "terminal-child", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, TargetRunId: childID.String(), RequestedBy: "operator", Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR},
	}
	for _, request := range cases {
		request.WatchId, request.ExpectedWatchRevision = watch.GetWatchId(), watch.GetRevision()
		response, err := service.Request(context.Background(), request)
		if err != nil || response.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED || response.GetAction().GetRejectionReason() == "" {
			t.Fatalf("response=%+v err=%v", response, err)
		}
	}
}

func TestNudgeWaitsForSafeTurnBoundaryAndRecoversExactlyOnce(t *testing.T) {
	service, repo, controller, watch, childID, parentID := actionFixture(t, domain.RunStatusRunning, domain.RunStatusRunning)
	request := &domainpb.RequestCohortWatchActionRequest{WatchId: watch.GetWatchId(), ExpectedWatchRevision: watch.GetRevision(), IdempotencyKey: "safe-nudge", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, TargetRunId: childID.String(), RequestedBy: parentID.String(), Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT, Message: "summarize blocker"}
	response, err := service.Request(context.Background(), request)
	if err != nil || response.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED || controller.continued != 0 {
		t.Fatalf("queued response=%+v continued=%d err=%v", response, controller.continued, err)
	}
	controller.runs[childID].Status = domain.RunStatusNeedsReview
	recovered := NewActionService(repo, controller)
	applied, err := recovered.RecoverPending(context.Background())
	if err != nil || applied != 1 || controller.continued != 1 {
		t.Fatalf("recovery applied=%d continued=%d err=%v", applied, controller.continued, err)
	}
	replay, err := recovered.Request(context.Background(), request)
	if err != nil || !replay.GetIdempotentReplay() || controller.continued != 1 {
		t.Fatalf("replay=%+v continued=%d err=%v", replay, controller.continued, err)
	}
}

func TestActionCooldownAndMaximumCountRejectRepeatedMutation(t *testing.T) {
	service, _, controller, watch, childID, _ := actionFixture(t, domain.RunStatusRunning, domain.RunStatusRunning)
	first := &domainpb.RequestCohortWatchActionRequest{WatchId: watch.GetWatchId(), ExpectedWatchRevision: watch.GetRevision(), IdempotencyKey: "stop-first", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_STOP, TargetRunId: childID.String(), RequestedBy: "operator", Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR, MaximumCount: 1, Cooldown: durationpb.New(time.Hour)}
	response, err := service.Request(context.Background(), first)
	if err != nil || response.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED {
		t.Fatalf("first=%+v err=%v", response, err)
	}
	controller.runs[childID].Status = domain.RunStatusRunning
	second := proto.Clone(first).(*domainpb.RequestCohortWatchActionRequest)
	second.IdempotencyKey = "stop-second"
	response, err = service.Request(context.Background(), second)
	if err != nil || response.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED {
		t.Fatalf("second=%+v err=%v", response, err)
	}
}

func TestPendingActionsReserveBudgetBeforeDelivery(t *testing.T) {
	service, _, controller, watch, childID, parentID := actionFixture(t, domain.RunStatusRunning, domain.RunStatusRunning)
	request := &domainpb.RequestCohortWatchActionRequest{WatchId: watch.GetWatchId(), ExpectedWatchRevision: watch.GetRevision(), IdempotencyKey: "pending-first", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_NUDGE, TargetRunId: childID.String(), RequestedBy: parentID.String(), Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT, MaximumCount: 1}
	first, err := service.Request(context.Background(), request)
	if err != nil || first.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED {
		t.Fatalf("first=%v err=%v", first, err)
	}
	request.IdempotencyKey = "pending-second"
	second, err := service.Request(context.Background(), request)
	if err != nil || second.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED {
		t.Fatalf("second=%v err=%v", second, err)
	}
	controller.runs[childID].Status = domain.RunStatusNeedsReview
	recovered, err := service.RecoverPending(context.Background())
	if err != nil || recovered != 1 || controller.continued != 1 {
		t.Fatalf("recovered=%d continued=%d err=%v", recovered, controller.continued, err)
	}
	recovered, err = service.RecoverPending(context.Background())
	if err != nil || recovered != 0 || controller.continued != 1 {
		t.Fatal("recovery duplicated delivery")
	}
}
